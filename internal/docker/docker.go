package docker

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type PoolDatabase struct {
	Name          string
	BaseURL       string
	ConnectionURL string
}

func (d *PoolDatabase) ConnectionString() string {
	return d.ConnectionURL
}

func CreatePoolDatabase(ctx context.Context, baseURL string, name string) (*PoolDatabase, error) {
	conn, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	_, _ = conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))

	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", name))
	if err != nil {
		return nil, fmt.Errorf("failed to create database %s: %w", name, err)
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}
	parsed.Path = "/" + name

	return &PoolDatabase{
		Name:          name,
		BaseURL:       baseURL,
		ConnectionURL: parsed.String(),
	}, nil
}

func DropPoolDatabase(ctx context.Context, baseURL string, name string) error {
	conn, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	_, err = conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name))
	if err != nil {
		return fmt.Errorf("failed to drop database %s: %w", name, err)
	}

	return nil
}

func ResetPoolDatabase(ctx context.Context, db *PoolDatabase) error {
	conn, err := pgx.Connect(ctx, db.ConnectionURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	_, err = conn.Exec(ctx, `
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO public;
	`)
	if err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	return nil
}

func ExecutePoolSQL(ctx context.Context, db *PoolDatabase, sql string) error {
	conn, err := pgx.Connect(ctx, db.ConnectionURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	_, err = conn.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	return nil
}

type Container struct {
	ID       string
	Host     string
	Port     string
	User     string
	Password string
	Database string
	isCI     bool
}

func (c *Container) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

type PostgresConfig struct {
	Version  string
	User     string
	Password string
	Database string
}

func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Version:  "16",
		User:     "shrugged",
		Password: "shrugged",
		Database: "shrugged",
	}
}

func StartPostgres(ctx context.Context, cfg PostgresConfig) (*Container, error) {
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return startCIPostgres(ctx, dbURL)
	}

	port, err := findFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find free port: %w", err)
	}

	image := fmt.Sprintf("postgres:%s", cfg.Version)

	args := []string{
		"run", "-d",
		"--rm",
		"-e", fmt.Sprintf("POSTGRES_USER=%s", cfg.User),
		"-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", cfg.Password),
		"-e", fmt.Sprintf("POSTGRES_DB=%s", cfg.Database),
		"-p", fmt.Sprintf("%s:5432", port),
		image,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to start container: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	containerID := strings.TrimSpace(string(output))

	container := &Container{
		ID:       containerID,
		Host:     "localhost",
		Port:     port,
		User:     cfg.User,
		Password: cfg.Password,
		Database: cfg.Database,
	}

	if err := waitForPostgres(ctx, container); err != nil {
		_ = StopContainer(context.Background(), containerID)
		return nil, err
	}

	return container, nil
}

func startCIPostgres(ctx context.Context, dbURL string) (*Container, error) {
	parsed, err := url.Parse(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}

	password, _ := parsed.User.Password()
	port := parsed.Port()
	if port == "" {
		port = "5432"
	}

	container := &Container{
		ID:       "ci-postgres",
		Host:     parsed.Hostname(),
		Port:     port,
		User:     parsed.User.Username(),
		Password: password,
		Database: strings.TrimPrefix(parsed.Path, "/"),
		isCI:     true,
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	_, err = conn.Exec(ctx, `
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO public;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to reset database: %w", err)
	}

	return container, nil
}

func StopContainer(ctx context.Context, containerID string) error {
	if containerID == "ci-postgres" {
		return nil
	}
	cmd := exec.CommandContext(ctx, "docker", "stop", containerID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

func waitForPostgres(ctx context.Context, container *Container) error {
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		cmd := exec.CommandContext(ctx, "docker", "exec", container.ID,
			"pg_isready", "-U", container.User, "-d", container.Database)

		if err := cmd.Run(); err == nil {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for postgres to be ready")
}

func ResetDatabase(ctx context.Context, container *Container) error {
	resetSQL := `
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO public;
	`
	return ExecuteSQL(ctx, container, resetSQL)
}

func ExecuteSQL(ctx context.Context, container *Container, sql string) error {
	if container.isCI {
		conn, err := pgx.Connect(ctx, container.ConnectionString())
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer func() { _ = conn.Close(ctx) }()

		_, err = conn.Exec(ctx, sql)
		if err != nil {
			return fmt.Errorf("failed to execute SQL: %w", err)
		}
		return nil
	}

	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", container.ID,
		"psql", "-U", container.User, "-d", container.Database, "-v", "ON_ERROR_STOP=1")

	cmd.Stdin = strings.NewReader(sql)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %s\n%w", string(output), err)
	}

	return nil
}

func ExecuteSQLFile(ctx context.Context, container *Container, filepath string) error {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if container.isCI {
		return ExecuteSQL(ctx, container, string(file))
	}

	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", container.ID,
		"psql", "-U", container.User, "-d", container.Database, "-v", "ON_ERROR_STOP=1", "-f", "-")

	cmd.Stdin = strings.NewReader(string(file))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute SQL file: %s\n%w", string(output), err)
	}

	return nil
}

func findFreePort() (string, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	defer func() { _ = listener.Close() }()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port), nil
}
