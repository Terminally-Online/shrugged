package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configContent := `
database_url: postgres://user:pass@localhost:5432/testdb
schema: custom_schema.sql
migrations_dir: custom_migrations
postgres_version: "15"
`
	configPath := filepath.Join(tmpDir, "shrugged.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/testdb" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://user:pass@localhost:5432/testdb")
	}
	if cfg.Schema != "custom_schema.sql" {
		t.Errorf("Schema = %q, want %q", cfg.Schema, "custom_schema.sql")
	}
	if cfg.MigrationsDir != "custom_migrations" {
		t.Errorf("MigrationsDir = %q, want %q", cfg.MigrationsDir, "custom_migrations")
	}
	if cfg.PostgresVersion != "15" {
		t.Errorf("PostgresVersion = %q, want %q", cfg.PostgresVersion, "15")
	}
}

func TestGetters_Defaults(t *testing.T) {
	cfg := &Config{}

	if schema := cfg.GetSchema(nil); schema != "schema.sql" {
		t.Errorf("GetSchema default = %q, want %q", schema, "schema.sql")
	}
	if migrationsDir := cfg.GetMigrationsDir(nil); migrationsDir != "migrations" {
		t.Errorf("GetMigrationsDir default = %q, want %q", migrationsDir, "migrations")
	}
	if pgVersion := cfg.GetPostgresVersion(nil); pgVersion != "16" {
		t.Errorf("GetPostgresVersion default = %q, want %q", pgVersion, "16")
	}
}

func TestGetters_FlagOverrides(t *testing.T) {
	cfg := &Config{
		DatabaseURL:     "postgres://config/db",
		Schema:          "config_schema.sql",
		MigrationsDir:   "config_migrations",
		PostgresVersion: "14",
	}

	flags := &Flags{
		URL:             "postgres://flag/db",
		Schema:          "flag_schema.sql",
		MigrationsDir:   "flag_migrations",
		PostgresVersion: "15",
	}

	dbURL, err := cfg.GetDatabaseURL(flags)
	if err != nil {
		t.Fatalf("GetDatabaseURL() error = %v", err)
	}
	if dbURL != "postgres://flag/db" {
		t.Errorf("GetDatabaseURL = %q, want %q", dbURL, "postgres://flag/db")
	}
	if schema := cfg.GetSchema(flags); schema != "flag_schema.sql" {
		t.Errorf("GetSchema = %q, want %q", schema, "flag_schema.sql")
	}
	if migrationsDir := cfg.GetMigrationsDir(flags); migrationsDir != "flag_migrations" {
		t.Errorf("GetMigrationsDir = %q, want %q", migrationsDir, "flag_migrations")
	}
	if pgVersion := cfg.GetPostgresVersion(flags); pgVersion != "15" {
		t.Errorf("GetPostgresVersion = %q, want %q", pgVersion, "15")
	}
}

func TestGetDatabaseURL_MissingURL(t *testing.T) {
	cfg := &Config{}
	_, err := cfg.GetDatabaseURL(nil)
	if err == nil {
		t.Error("expected error for missing database_url")
	}
}

func TestGetDatabaseURL_FromConfig(t *testing.T) {
	cfg := &Config{DatabaseURL: "postgres://config/db"}
	dbURL, err := cfg.GetDatabaseURL(nil)
	if err != nil {
		t.Fatalf("GetDatabaseURL() error = %v", err)
	}
	if dbURL != "postgres://config/db" {
		t.Errorf("GetDatabaseURL = %q, want %q", dbURL, "postgres://config/db")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/shrugged.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configContent := `
database_url: [invalid yaml
`
	configPath := filepath.Join(tmpDir, "shrugged.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoad_EnvVarExpansion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	_ = os.Setenv("TEST_DB_URL", "postgres://env:pass@localhost/envdb")
	defer func() { _ = os.Unsetenv("TEST_DB_URL") }()

	configContent := `
database_url: ${TEST_DB_URL}
`
	configPath := filepath.Join(tmpDir, "shrugged.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://env:pass@localhost/envdb" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://env:pass@localhost/envdb")
	}
}

func TestLoad_EnvVarExpansionDollarSign(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	_ = os.Setenv("TEST_SCHEMA_PATH", "env_schema.sql")
	defer func() { _ = os.Unsetenv("TEST_SCHEMA_PATH") }()

	configContent := `
database_url: postgres://localhost/testdb
schema: $TEST_SCHEMA_PATH
`
	configPath := filepath.Join(tmpDir, "shrugged.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Schema != "env_schema.sql" {
		t.Errorf("Schema = %q, want %q", cfg.Schema, "env_schema.sql")
	}
}

