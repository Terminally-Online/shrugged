package docker

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

type Container struct {
	ID       string
	Host     string
	Port     string
	User     string
	Password string
	Database string
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

func StopContainer(ctx context.Context, containerID string) error {
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
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", container.ID,
		"psql", "-U", container.User, "-d", container.Database, "-v", "ON_ERROR_STOP=1", "-f", "-")

	file, err := exec.CommandContext(ctx, "cat", filepath).Output()
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

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
