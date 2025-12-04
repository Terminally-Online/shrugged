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
	defer os.RemoveAll(tmpDir)

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

func TestLoad_Defaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configContent := `
database_url: postgres://localhost/testdb
`
	configPath := filepath.Join(tmpDir, "shrugged.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Schema != "schema.sql" {
		t.Errorf("Schema default = %q, want %q", cfg.Schema, "schema.sql")
	}
	if cfg.MigrationsDir != "migrations" {
		t.Errorf("MigrationsDir default = %q, want %q", cfg.MigrationsDir, "migrations")
	}
	if cfg.PostgresVersion != "16" {
		t.Errorf("PostgresVersion default = %q, want %q", cfg.PostgresVersion, "16")
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configContent := `
schema: schema.sql
`
	configPath := filepath.Join(tmpDir, "shrugged.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Error("expected error for missing database_url")
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
	defer os.RemoveAll(tmpDir)

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
	defer os.RemoveAll(tmpDir)

	os.Setenv("TEST_DB_URL", "postgres://env:pass@localhost/envdb")
	defer os.Unsetenv("TEST_DB_URL")

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
	defer os.RemoveAll(tmpDir)

	os.Setenv("TEST_SCHEMA_PATH", "env_schema.sql")
	defer os.Unsetenv("TEST_SCHEMA_PATH")

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

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     Config{DatabaseURL: "postgres://localhost/db"},
			wantErr: false,
		},
		{
			name:    "missing database_url",
			cfg:     Config{DatabaseURL: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
