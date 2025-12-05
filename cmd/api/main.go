package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"shrugged/internal/api"
	"shrugged/internal/docker"
)

func main() {
	port := getEnv("PORT", "8080")
	poolMinSize := getEnvInt("POOL_MIN_SIZE", 2)
	poolMaxSize := getEnvInt("POOL_MAX_SIZE", 5)
	cacheTTL := getEnvDuration("CACHE_TTL", 30*time.Minute)
	cacheMaxSize := getEnvInt("CACHE_MAX_SIZE", 1000)
	rateLimit := getEnvInt("RATE_LIMIT", 30)
	rateWindow := getEnvDuration("RATE_WINDOW", 1*time.Minute)
	requestTimeout := getEnvDuration("REQUEST_TIMEOUT", 60*time.Second)
	postgresVersion := getEnv("POSTGRES_VERSION", "16")
	allowedOrigins := getEnv("ALLOWED_ORIGINS", "*")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dockerCfg := docker.PostgresConfig{
		Version:  postgresVersion,
		User:     "shrugged",
		Password: "shrugged",
		Database: "shrugged",
	}

	pool := api.NewContainerPool(dockerCfg, poolMinSize, poolMaxSize)
	cache := api.NewDiffCache(cacheTTL, cacheMaxSize)
	limiter := api.NewRateLimiter(rateLimit, rateWindow)
	handler := api.NewHandler(pool, cache, limiter, requestTimeout)

	log.Println("Warming container pool...")
	if err := pool.Start(ctx); err != nil {
		log.Fatalf("Failed to start container pool: %v", err)
	}
	log.Printf("Pool warmed with %d containers", pool.Size())

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	corsHandler := corsMiddleware(allowedOrigins)(limiter.Middleware(mux))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: requestTimeout + 10*time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Starting API server on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Cleaning up container pool...")
	pool.Shutdown(shutdownCtx)

	log.Println("Server stopped")
}

func corsMiddleware(allowedOrigins string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if allowedOrigins == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	fmt.Println(`
 _____ _                                  _    _    _____ _____
/  ___| |                                | |  | |  |  _  |  _  |
\ '--.| |__  _ __ _   _  __ _  __ _  ___ | |  | |  | | | | | | |
 '--. \ '_ \| '__| | | |/ _' |/ _' |/ _ \| |/\| |  | | | | | | |
/\__/ / | | | |  | |_| | (_| | (_| |  __/\  /\  /  \ \_/ / |_| /
\____/|_| |_|_|   \__,_|\__, |\__, |\___| \/  \/    \___/|___/
                         __/ | __/ |
                        |___/ |___/        Playground API
`)
}
