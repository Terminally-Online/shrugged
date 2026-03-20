package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/terminally-online/shrugged/internal/codegen"
	_ "github.com/terminally-online/shrugged/internal/codegen/golang"
	_ "github.com/terminally-online/shrugged/internal/codegen/typescript"
	"github.com/terminally-online/shrugged/internal/diff"
	"github.com/terminally-online/shrugged/internal/docker"
	"github.com/terminally-online/shrugged/internal/introspect"
	"github.com/terminally-online/shrugged/internal/migrate"
	"github.com/terminally-online/shrugged/internal/parser"
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
	Status    string `json:"status"`
	PoolSize  int    `json:"pool_size"`
	CacheSize int    `json:"cache_size"`
}

type ValidateRequest struct {
	Schema string `json:"schema"`
}

type ValidateResponse struct {
	Valid      bool     `json:"valid"`
	Objects    int      `json:"objects"`
	Warnings   []string `json:"warnings"`
	DurationMs int64    `json:"duration_ms,omitempty"`
}

type MigrateRequest struct {
	// Migrations holds the SQL content of each existing migration file, applied in order.
	Migrations []string `json:"migrations"`
	// Schema is the desired schema SQL to migrate toward.
	Schema string `json:"schema"`
}

type MigrateResponse struct {
	Up           string `json:"up"`
	Down         string `json:"down"`
	Changes      int    `json:"changes"`
	Irreversible bool   `json:"irreversible"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
}

type GenerateRequest struct {
	Schema   string `json:"schema"`
	Language string `json:"language"`
	OutDir   string `json:"out_dir"`
}

type GenerateResponse struct {
	Tables         int   `json:"tables"`
	Enums          int   `json:"enums"`
	CompositeTypes int   `json:"composite_types"`
	DurationMs     int64 `json:"duration_ms,omitempty"`
}

// NOTE: Docker is required for apply — it connects to the database at the provided URL
//       and reads migrations from the server-local migrations_dir path.
type ApplyRequest struct {
	URL           string `json:"url"`
	MigrationsDir string `json:"migrations_dir"`
	DryRun        bool   `json:"dry_run"`
	Force         bool   `json:"force"`
}

type ApplyResponse struct {
	Applied    []string `json:"applied"`
	DryRun     bool     `json:"dry_run"`
	DurationMs int64    `json:"duration_ms,omitempty"`
}

type MigrationStatus struct {
	Name      string    `json:"name"`
	Applied   bool      `json:"applied"`
	AppliedAt time.Time `json:"applied_at,omitempty"`
	Modified  bool      `json:"modified"`
}

type StatusResponse struct {
	Migrations    []MigrationStatus `json:"migrations"`
	ModifiedCount int               `json:"modified_count"`
	PendingCount  int               `json:"pending_count"`
	DurationMs    int64             `json:"duration_ms,omitempty"`
}

type Handler struct {
	pool    *DatabasePool
	cache   *DiffCache
	limiter *RateLimiter
	timeout time.Duration
}

func NewHandler(pool *DatabasePool, cache *DiffCache, limiter *RateLimiter, timeout time.Duration) *Handler {
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
	case "/validate":
		h.handleValidate(w, r)
	case "/migrate":
		h.handleMigrate(w, r)
	case "/generate":
		h.handleGenerate(w, r)
	case "/apply":
		h.handleApply(w, r)
	case "/status":
		h.handleStatus(w, r)
	default:
		http.NotFound(w, r)
	}
}

// writeJSON marshals v to JSON and writes it as the response body. Marshaling
// happens into a buffer before any bytes are sent to the client, so a 500 can
// be returned cleanly if encoding fails.
func writeJSON(w http.ResponseWriter, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
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
	writeJSON(w, resp)
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
		writeJSON(w, cached)
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
	writeJSON(w, result)
}

func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Schema = strings.TrimSpace(req.Schema)
	if req.Schema == "" {
		http.Error(w, "schema is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	start := time.Now()

	db, err := h.pool.Acquire(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to acquire database: %v", err), http.StatusInternalServerError)
		return
	}
	defer h.pool.Release(db)

	if err := docker.ExecutePoolSQL(ctx, db, req.Schema); err != nil {
		http.Error(w, fmt.Sprintf("schema validation failed: %v", err), http.StatusBadRequest)
		return
	}

	schema, err := introspect.Database(ctx, db.ConnectionString())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to introspect schema: %v", err), http.StatusInternalServerError)
		return
	}

	warnings := schema.Lint()
	if warnings == nil {
		warnings = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, ValidateResponse{
		Valid:      true,
		Objects:    schema.ObjectCount(),
		Warnings:   warnings,
		DurationMs: time.Since(start).Milliseconds(),
	})
}

// NOTE: Docker is required for migrate — it uses the pool to apply migrations and
//       diff against the desired schema. Migrations are passed as SQL strings in
//       the request body (not as filesystem paths).
func (h *Handler) handleMigrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MigrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Schema = strings.TrimSpace(req.Schema)
	if req.Schema == "" {
		http.Error(w, "schema is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	start := time.Now()

	db, err := h.pool.Acquire(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to acquire database: %v", err), http.StatusInternalServerError)
		return
	}
	defer h.pool.Release(db)

	// Apply existing migrations to build current state.
	for _, migration := range req.Migrations {
		if err := docker.ExecutePoolSQL(ctx, db, migration); err != nil {
			http.Error(w, fmt.Sprintf("failed to apply migration: %v", err), http.StatusBadRequest)
			return
		}
	}

	currentSchema, err := introspect.Database(ctx, db.ConnectionString())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to introspect current state: %v", err), http.StatusInternalServerError)
		return
	}

	if err := docker.ResetPoolDatabase(ctx, db); err != nil {
		http.Error(w, fmt.Sprintf("failed to reset database: %v", err), http.StatusInternalServerError)
		return
	}

	if err := docker.ExecutePoolSQL(ctx, db, req.Schema); err != nil {
		http.Error(w, fmt.Sprintf("failed to apply desired schema: %v", err), http.StatusBadRequest)
		return
	}

	desiredSchema, err := introspect.Database(ctx, db.ConnectionString())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to introspect desired schema: %v", err), http.StatusInternalServerError)
		return
	}

	changes := diff.Compare(currentSchema, desiredSchema)

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

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, MigrateResponse{
		Up:           strings.Join(upStatements, "\n\n"),
		Down:         strings.Join(downStatements, "\n\n"),
		Changes:      len(changes),
		Irreversible: hasIrreversible,
		DurationMs:   time.Since(start).Milliseconds(),
	})
}

// NOTE: Docker is required for generate — it applies the schema to a pool database
//       and introspects it before running the generator. out_dir must be accessible
//       from the server's filesystem.
func (h *Handler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Schema = strings.TrimSpace(req.Schema)
	if req.Schema == "" {
		http.Error(w, "schema is required", http.StatusBadRequest)
		return
	}
	if req.OutDir == "" {
		http.Error(w, "out_dir is required", http.StatusBadRequest)
		return
	}
	if req.Language == "" {
		req.Language = "go"
	}

	generator, err := codegen.Get(req.Language)
	if err != nil {
		http.Error(w, fmt.Sprintf("unsupported language %q (available: %v)", req.Language, codegen.Languages()), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	start := time.Now()

	db, err := h.pool.Acquire(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to acquire database: %v", err), http.StatusInternalServerError)
		return
	}
	defer h.pool.Release(db)

	if err := docker.ExecutePoolSQL(ctx, db, req.Schema); err != nil {
		http.Error(w, fmt.Sprintf("schema validation failed: %v", err), http.StatusBadRequest)
		return
	}

	schema, err := introspect.Database(ctx, db.ConnectionString())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to introspect schema: %v", err), http.StatusInternalServerError)
		return
	}

	if err := generator.Generate(schema, req.OutDir); err != nil {
		http.Error(w, fmt.Sprintf("code generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, GenerateResponse{
		Tables:         len(schema.Tables),
		Enums:          len(schema.Enums),
		CompositeTypes: len(schema.CompositeTypes),
		DurationMs:     time.Since(start).Milliseconds(),
	})
}

func (h *Handler) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.MigrationsDir) == "" {
		http.Error(w, "migrations_dir is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	start := time.Now()

	if err := migrate.ValidateSum(req.MigrationsDir); err != nil {
		if !req.Force {
			http.Error(w, fmt.Sprintf("sum file validation failed: %v", err), http.StatusConflict)
			return
		}
	}

	modified, err := migrate.HasModifiedMigrations(ctx, req.URL, req.MigrationsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to check modified migrations: %v", err), http.StatusInternalServerError)
		return
	}
	if len(modified) > 0 && !req.Force {
		names := make([]string, len(modified))
		for i, m := range modified {
			names[i] = m.Name
		}
		http.Error(w, fmt.Sprintf("modified migrations detected: %s", strings.Join(names, ", ")), http.StatusConflict)
		return
	}

	pending, err := migrate.GetPending(ctx, req.URL, req.MigrationsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get pending migrations: %v", err), http.StatusInternalServerError)
		return
	}

	applied := make([]string, 0, len(pending))

	if !req.DryRun {
		for _, m := range pending {
			if err := migrate.Apply(ctx, req.URL, m); err != nil {
				http.Error(w, fmt.Sprintf("failed to apply migration %s: %v", m.Name, err), http.StatusInternalServerError)
				return
			}
			applied = append(applied, m.Name)
		}
	} else {
		for _, m := range pending {
			applied = append(applied, m.Name)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, ApplyResponse{
		Applied:    applied,
		DryRun:     req.DryRun,
		DurationMs: time.Since(start).Milliseconds(),
	})
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dbURL := r.URL.Query().Get("url")
	migrationsDir := r.URL.Query().Get("migrations_dir")

	if strings.TrimSpace(dbURL) == "" {
		http.Error(w, "url query parameter is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(migrationsDir) == "" {
		http.Error(w, "migrations_dir query parameter is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	start := time.Now()

	applied, err := migrate.GetAppliedWithStatus(ctx, dbURL, migrationsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get applied migrations: %v", err), http.StatusInternalServerError)
		return
	}

	pending, err := migrate.GetPending(ctx, dbURL, migrationsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get pending migrations: %v", err), http.StatusInternalServerError)
		return
	}

	statuses := make([]MigrationStatus, 0, len(applied)+len(pending))
	var modifiedCount int

	for _, m := range applied {
		statuses = append(statuses, MigrationStatus{
			Name:      m.Name,
			Applied:   true,
			AppliedAt: m.AppliedAt,
			Modified:  m.Modified,
		})
		if m.Modified {
			modifiedCount++
		}
	}
	for _, m := range pending {
		statuses = append(statuses, MigrationStatus{
			Name:    m.Name,
			Applied: false,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, StatusResponse{
		Migrations:    statuses,
		ModifiedCount: modifiedCount,
		PendingCount:  len(pending),
		DurationMs:    time.Since(start).Milliseconds(),
	})
}

func (h *Handler) performDiff(ctx context.Context, previous, current string) (*DiffResult, error) {
	db, err := h.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire database: %w", err)
	}
	defer h.pool.Release(db)

	var previousSchema, currentSchema *parser.Schema

	if previous != "" {
		if err := docker.ExecutePoolSQL(ctx, db, previous); err != nil {
			return nil, fmt.Errorf("failed to apply previous schema: %w", err)
		}

		previousSchema, err = introspect.Database(ctx, db.ConnectionString())
		if err != nil {
			return nil, fmt.Errorf("failed to introspect previous schema: %w", err)
		}

		if err := docker.ResetPoolDatabase(ctx, db); err != nil {
			return nil, fmt.Errorf("failed to reset database: %w", err)
		}
	} else {
		previousSchema = &parser.Schema{}
	}

	if err := docker.ExecutePoolSQL(ctx, db, current); err != nil {
		return nil, fmt.Errorf("failed to apply current schema: %w", err)
	}

	currentSchema, err = introspect.Database(ctx, db.ConnectionString())
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
