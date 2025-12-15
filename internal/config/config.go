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
	Out             string `yaml:"out"`
	Language        string `yaml:"language"`
	Queries         string `yaml:"queries"`
	QueriesOut      string `yaml:"queries_out"`
}

type Flags struct {
	URL             string
	Schema          string
	MigrationsDir   string
	PostgresVersion string
	Out             string
	Language        string
	Queries         string
	QueriesOut      string
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
	cfg.Out = expandEnv(cfg.Out)
	cfg.Language = expandEnv(cfg.Language)
	cfg.Queries = expandEnv(cfg.Queries)
	cfg.QueriesOut = expandEnv(cfg.QueriesOut)

	return &cfg, nil
}

func (c *Config) GetDatabaseURL(flags *Flags) (string, error) {
	if flags != nil && flags.URL != "" {
		return flags.URL, nil
	}
	if c.DatabaseURL != "" {
		return c.DatabaseURL, nil
	}
	return "", fmt.Errorf("database_url is required (set in config or pass --url flag)")
}

func (c *Config) GetSchema(flags *Flags) string {
	if flags != nil && flags.Schema != "" {
		return flags.Schema
	}
	if c.Schema != "" {
		return c.Schema
	}
	return "schema.sql"
}

func (c *Config) GetMigrationsDir(flags *Flags) string {
	if flags != nil && flags.MigrationsDir != "" {
		return flags.MigrationsDir
	}
	if c.MigrationsDir != "" {
		return c.MigrationsDir
	}
	return "migrations"
}

func (c *Config) GetPostgresVersion(flags *Flags) string {
	if flags != nil && flags.PostgresVersion != "" {
		return flags.PostgresVersion
	}
	if c.PostgresVersion != "" {
		return c.PostgresVersion
	}
	return "16"
}

func (c *Config) GetOut(flags *Flags) string {
	if flags != nil && flags.Out != "" {
		return flags.Out
	}
	if c.Out != "" {
		return c.Out
	}
	return "models"
}

func (c *Config) GetLanguage(flags *Flags) string {
	if flags != nil && flags.Language != "" {
		return flags.Language
	}
	if c.Language != "" {
		return c.Language
	}
	return "go"
}

func (c *Config) GetQueries(flags *Flags) string {
	if flags != nil && flags.Queries != "" {
		return flags.Queries
	}
	if c.Queries != "" {
		return c.Queries
	}
	return ""
}

func (c *Config) GetQueriesOut(flags *Flags) string {
	if flags != nil && flags.QueriesOut != "" {
		return flags.QueriesOut
	}
	if c.QueriesOut != "" {
		return c.QueriesOut
	}
	return "queries"
}

func expandEnv(s string) string {
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		envVar := s[2 : len(s)-1]
		return os.Getenv(envVar)
	}
	return os.ExpandEnv(s)
}
