package docker

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestContainer_ConnectionString(t *testing.T) {
	c := &Container{
		ID:       "abc123",
		Host:     "localhost",
		Port:     "5432",
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if got := c.ConnectionString(); got != expected {
		t.Errorf("ConnectionString() = %q, want %q", got, expected)
	}
}

func TestDefaultPostgresConfig(t *testing.T) {
	cfg := DefaultPostgresConfig()

	if cfg.Version != "16" {
		t.Errorf("Version = %q, want %q", cfg.Version, "16")
	}
	if cfg.User != "shrugged" {
		t.Errorf("User = %q, want %q", cfg.User, "shrugged")
	}
	if cfg.Password != "shrugged" {
		t.Errorf("Password = %q, want %q", cfg.Password, "shrugged")
	}
	if cfg.Database != "shrugged" {
		t.Errorf("Database = %q, want %q", cfg.Database, "shrugged")
	}
}

func TestStartPostgres_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := DefaultPostgresConfig()
	container, err := StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}

	defer func() {
		_ = StopContainer(context.Background(), container.ID)
	}()

	if container.Port == "" {
		t.Error("container.Port is empty")
	}

	// StartPostgres has two modes and the environment picks which: with no
	// DATABASE_URL it starts a container and reaches it on localhost, and with
	// one it adopts the database already there, wherever the URL points. This
	// asserted the first shape unconditionally, so it failed on merit anywhere
	// the second one is the right answer — which is every runner that hands a
	// database to the job rather than a docker socket.
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		if container.ID == "" {
			t.Error("container.ID is empty")
		}
		if container.Host != "localhost" {
			t.Errorf("container.Host = %q, want %q", container.Host, "localhost")
		}
		return
	}

	if container.ID != "" {
		t.Errorf("container.ID = %q, want empty: no container is started when DATABASE_URL is set", container.ID)
	}
	parsed, err := url.Parse(dbURL)
	if err != nil {
		t.Fatalf("DATABASE_URL is not a URL: %v", err)
	}
	if container.Host != parsed.Hostname() {
		t.Errorf("container.Host = %q, want %q from DATABASE_URL", container.Host, parsed.Hostname())
	}
}

func TestExecuteSQL_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := DefaultPostgresConfig()
	container, err := StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}

	defer func() {
		_ = StopContainer(context.Background(), container.ID)
	}()

	sql := `
		CREATE TABLE test_table (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		);
	`
	if err := ExecuteSQL(ctx, container, sql); err != nil {
		t.Errorf("ExecuteSQL() error = %v", err)
	}

	checkSQL := `SELECT COUNT(*) FROM test_table;`
	if err := ExecuteSQL(ctx, container, checkSQL); err != nil {
		t.Errorf("ExecuteSQL() check query error = %v", err)
	}
}

func TestResetDatabase_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := DefaultPostgresConfig()
	container, err := StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}

	defer func() {
		_ = StopContainer(context.Background(), container.ID)
	}()

	createSQL := `CREATE TABLE to_be_dropped (id SERIAL PRIMARY KEY);`
	if err := ExecuteSQL(ctx, container, createSQL); err != nil {
		t.Fatalf("ExecuteSQL() create error = %v", err)
	}

	if err := ResetDatabase(ctx, container); err != nil {
		t.Errorf("ResetDatabase() error = %v", err)
	}

	checkSQL := `SELECT COUNT(*) FROM to_be_dropped;`
	err = ExecuteSQL(ctx, container, checkSQL)
	if err == nil {
		t.Error("expected error querying dropped table, got nil")
	}
}
