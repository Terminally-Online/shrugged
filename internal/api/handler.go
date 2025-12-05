package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"shrugged/internal/diff"
	"shrugged/internal/docker"
	"shrugged/internal/introspect"
	"shrugged/internal/parser"
)

type DiffRequest struct {
	Previous string `json:"previous"`
	Current  string `json:"current"`
}

type DiffResult struct {
	Up           string `json:"up"`
	Down         string `json:"down"`
	Changes      int    `json:"changes"`
	Irreversible bool   `json:"irreversible"`
	Cached       bool   `json:"cached,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
}

type HealthResponse struct {
	Status     string `json:"status"`
	PoolSize   int    `json:"pool_size"`
	CacheSize  int    `json:"cache_size"`
}

type Handler struct {
	pool    *ContainerPool
	cache   *DiffCache
	limiter *RateLimiter
	timeout time.Duration
}

func NewHandler(pool *ContainerPool, cache *DiffCache, limiter *RateLimiter, timeout time.Duration) *Handler {
	return &Handler{
		pool:    pool,
		cache:   cache,
		limiter: limiter,
		timeout: timeout,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/diff":
		h.handleDiff(w, r)
	case "/health":
		h.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := HealthResponse{
		Status:    "ok",
		PoolSize:  h.pool.Size(),
		CacheSize: h.cache.Size(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Current = strings.TrimSpace(req.Current)
	if req.Current == "" {
		http.Error(w, "current schema is required", http.StatusBadRequest)
		return
	}

	cacheKey := h.cache.Key(req.Previous, req.Current)
	if cached, ok := h.cache.Get(cacheKey); ok {
		cached.Cached = true
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		_ = json.NewEncoder(w).Encode(cached)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	start := time.Now()

	result, err := h.performDiff(ctx, req.Previous, req.Current)
	if err != nil {
		http.Error(w, fmt.Sprintf("diff failed: %v", err), http.StatusInternalServerError)
		return
	}

	result.DurationMs = time.Since(start).Milliseconds()

	h.cache.Set(cacheKey, result)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) performDiff(ctx context.Context, previous, current string) (*DiffResult, error) {
	container, err := h.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire container: %w", err)
	}
	defer h.pool.Release(container)

	var previousSchema, currentSchema *parser.Schema

	if previous != "" {
		if err := docker.ExecuteSQL(ctx, container, previous); err != nil {
			return nil, fmt.Errorf("failed to apply previous schema: %w", err)
		}

		previousSchema, err = introspect.Database(ctx, container.ConnectionString())
		if err != nil {
			return nil, fmt.Errorf("failed to introspect previous schema: %w", err)
		}

		if err := docker.ResetDatabase(ctx, container); err != nil {
			return nil, fmt.Errorf("failed to reset database: %w", err)
		}
	} else {
		previousSchema = &parser.Schema{}
	}

	if err := docker.ExecuteSQL(ctx, container, current); err != nil {
		return nil, fmt.Errorf("failed to apply current schema: %w", err)
	}

	currentSchema, err = introspect.Database(ctx, container.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to introspect current schema: %w", err)
	}

	changes := diff.Compare(previousSchema, currentSchema)

	var upStatements, downStatements []string
	var hasIrreversible bool

	for _, change := range changes {
		upStatements = append(upStatements, change.SQL())
		if !change.IsReversible() {
			hasIrreversible = true
		}
	}

	for i := len(changes) - 1; i >= 0; i-- {
		downStatements = append(downStatements, changes[i].DownSQL())
	}

	return &DiffResult{
		Up:           strings.Join(upStatements, "\n\n"),
		Down:         strings.Join(downStatements, "\n\n"),
		Changes:      len(changes),
		Irreversible: hasIrreversible,
	}, nil
}
