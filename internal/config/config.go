package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Schema          string `yaml:"schema"`
	DatabaseURL     string `yaml:"database_url"`
	MigrationsDir   string `yaml:"migrations_dir"`
	PostgresVersion string `yaml:"postgres_version"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.DatabaseURL = expandEnv(cfg.DatabaseURL)
	cfg.Schema = expandEnv(cfg.Schema)
	cfg.MigrationsDir = expandEnv(cfg.MigrationsDir)
	cfg.PostgresVersion = expandEnv(cfg.PostgresVersion)

	if cfg.Schema == "" {
		cfg.Schema = "schema.sql"
	}
	if cfg.MigrationsDir == "" {
		cfg.MigrationsDir = "migrations"
	}
	if cfg.PostgresVersion == "" {
		cfg.PostgresVersion = "16"
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}
	return nil
}

func expandEnv(s string) string {
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		envVar := s[2 : len(s)-1]
		return os.Getenv(envVar)
	}
	return os.ExpandEnv(s)
}
