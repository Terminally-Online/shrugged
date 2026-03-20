package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testHandler returns a Handler wired with real (but empty) pool, cache, and
// limiter. The pool has zero warm databases and is never started, so it will
// only fail when actual DB operations are attempted — not during validation.
func testHandler() *Handler {
	pool := NewDatabasePool(0, 5)
	cache := NewDiffCache(time.Minute, 100)
	limiter := NewRateLimiter(1000, time.Minute)
	return NewHandler(pool, cache, limiter, 30*time.Second)
}

// post sends a POST request to the handler with the given path and JSON body.
func post(h *Handler, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// get sends a GET request to the handler with the given path (can include query string).
func get(h *Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// postRaw sends a POST with a raw body string (for malformed JSON tests).
func postRaw(h *Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// TestRouting verifies unknown paths return 404.
func TestRouting(t *testing.T) {
	h := testHandler()
	rr := get(h, "/nonexistent")
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown route: got %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// TestHealth verifies the /health route contract.
func TestHealth(t *testing.T) {
	h := testHandler()

	t.Run("GET returns 200 with JSON", func(t *testing.T) {
		rr := get(h, "/health")
		if rr.Code != http.StatusOK {
			t.Fatalf("GET /health: got %d, want 200", rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		var resp HealthResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Status != "ok" {
			t.Errorf("status = %q, want ok", resp.Status)
		}
	})

	t.Run("POST returns 405", func(t *testing.T) {
		rr := post(h, "/health", nil)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST /health: got %d, want 405", rr.Code)
		}
	})
}

// TestDiff validates the /diff route HTTP layer.
func TestDiff(t *testing.T) {
	h := testHandler()

	t.Run("GET returns 405", func(t *testing.T) {
		rr := get(h, "/diff")
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("got %d, want 405", rr.Code)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		rr := postRaw(h, "/diff", `{bad json`)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("empty current returns 400", func(t *testing.T) {
		rr := post(h, "/diff", DiffRequest{Current: "   "})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("missing current returns 400", func(t *testing.T) {
		rr := post(h, "/diff", DiffRequest{})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})
}

// TestValidate validates the /validate route HTTP layer.
func TestValidate(t *testing.T) {
	h := testHandler()

	t.Run("GET returns 405", func(t *testing.T) {
		rr := get(h, "/validate")
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("got %d, want 405", rr.Code)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		rr := postRaw(h, "/validate", `{bad json`)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("empty schema returns 400", func(t *testing.T) {
		rr := post(h, "/validate", ValidateRequest{Schema: "  "})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("missing schema returns 400", func(t *testing.T) {
		rr := post(h, "/validate", ValidateRequest{})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})
}

// TestMigrate validates the /migrate route HTTP layer.
func TestMigrate(t *testing.T) {
	h := testHandler()

	t.Run("GET returns 405", func(t *testing.T) {
		rr := get(h, "/migrate")
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("got %d, want 405", rr.Code)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		rr := postRaw(h, "/migrate", `{bad json`)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("empty schema returns 400", func(t *testing.T) {
		rr := post(h, "/migrate", MigrateRequest{Schema: ""})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})
}

// TestGenerate validates the /generate route HTTP layer.
func TestGenerate(t *testing.T) {
	h := testHandler()

	t.Run("GET returns 405", func(t *testing.T) {
		rr := get(h, "/generate")
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("got %d, want 405", rr.Code)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		rr := postRaw(h, "/generate", `{bad json`)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("missing schema returns 400", func(t *testing.T) {
		rr := post(h, "/generate", GenerateRequest{OutDir: "/tmp/out"})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("missing out_dir returns 400", func(t *testing.T) {
		rr := post(h, "/generate", GenerateRequest{Schema: "CREATE TABLE t (id int);"})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("unsupported language returns 400", func(t *testing.T) {
		rr := post(h, "/generate", GenerateRequest{
			Schema:   "CREATE TABLE t (id int);",
			OutDir:   "/tmp/out",
			Language: "cobol",
		})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})
}

// TestApply validates the /apply route HTTP layer.
func TestApply(t *testing.T) {
	h := testHandler()

	t.Run("GET returns 405", func(t *testing.T) {
		rr := get(h, "/apply")
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("got %d, want 405", rr.Code)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		rr := postRaw(h, "/apply", `{bad json`)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("missing url returns 400", func(t *testing.T) {
		rr := post(h, "/apply", ApplyRequest{MigrationsDir: "/tmp/migrations"})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("missing migrations_dir returns 400", func(t *testing.T) {
		rr := post(h, "/apply", ApplyRequest{URL: "postgres://localhost/db"})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})
}

// TestStatus validates the /status route HTTP layer.
func TestStatus(t *testing.T) {
	h := testHandler()

	t.Run("POST returns 405", func(t *testing.T) {
		rr := post(h, "/status", nil)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("got %d, want 405", rr.Code)
		}
	})

	t.Run("missing url returns 400", func(t *testing.T) {
		rr := get(h, "/status?migrations_dir=/tmp/migrations")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("missing migrations_dir returns 400", func(t *testing.T) {
		rr := get(h, "/status?url=postgres://localhost/db")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("missing both params returns 400", func(t *testing.T) {
		rr := get(h, "/status")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})
}
