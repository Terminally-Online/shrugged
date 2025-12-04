package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Migration struct {
	Name      string
	Content   string
	Checksum  string
	AppliedAt time.Time
	Modified  bool
}

const migrationsTable = "shrugged_migrations"

func ComputeChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func EnsureMigrationsTable(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			checksum TEXT
		)
	`, migrationsTable))
	if err != nil {
		return err
	}

	_, err = conn.Exec(ctx, fmt.Sprintf(`
		ALTER TABLE %s ADD COLUMN IF NOT EXISTS checksum TEXT
	`, migrationsTable))
	return err
}

func GetApplied(ctx context.Context, databaseURL string) ([]Migration, error) {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	if err := EnsureMigrationsTable(ctx, conn); err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, fmt.Sprintf(`
		SELECT name, applied_at, COALESCE(checksum, '')
		FROM %s
		ORDER BY name
	`, migrationsTable))
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.Name, &m.AppliedAt, &m.Checksum); err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		migrations = append(migrations, m)
	}

	return migrations, nil
}

func GetPending(ctx context.Context, databaseURL, migrationsDir string) ([]Migration, error) {
	applied, err := GetApplied(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[string]Migration)
	for _, m := range applied {
		appliedMap[m.Name] = m
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var pending []Migration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		if strings.HasSuffix(name, ".down.sql") {
			continue
		}

		if _, exists := appliedMap[name]; exists {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", name, err)
		}

		pending = append(pending, Migration{
			Name:     name,
			Content:  string(content),
			Checksum: ComputeChecksum(string(content)),
		})
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Name < pending[j].Name
	})

	return pending, nil
}

func GetAppliedWithStatus(ctx context.Context, databaseURL, migrationsDir string) ([]Migration, error) {
	applied, err := GetApplied(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	for i, m := range applied {
		path := filepath.Join(migrationsDir, m.Name)
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				applied[i].Modified = true
				continue
			}
			return nil, fmt.Errorf("failed to read migration %s: %w", m.Name, err)
		}

		applied[i].Content = string(content)
		currentChecksum := ComputeChecksum(string(content))

		if m.Checksum != "" && m.Checksum != currentChecksum {
			applied[i].Modified = true
		}
	}

	return applied, nil
}

func HasModifiedMigrations(ctx context.Context, databaseURL, migrationsDir string) ([]Migration, error) {
	applied, err := GetAppliedWithStatus(ctx, databaseURL, migrationsDir)
	if err != nil {
		return nil, err
	}

	var modified []Migration
	for _, m := range applied {
		if m.Modified {
			modified = append(modified, m)
		}
	}

	return modified, nil
}

func Apply(ctx context.Context, databaseURL string, m Migration) error {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	if err := EnsureMigrationsTable(ctx, conn); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, m.Content); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	checksum := m.Checksum
	if checksum == "" {
		checksum = ComputeChecksum(m.Content)
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (name, checksum) VALUES ($1, $2)
	`, migrationsTable), m.Name, checksum); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit(ctx)
}

func GetLastApplied(ctx context.Context, databaseURL string) (*Migration, error) {
	applied, err := GetApplied(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if len(applied) == 0 {
		return nil, nil
	}
	return &applied[len(applied)-1], nil
}

func GetRollbackable(ctx context.Context, databaseURL, migrationsDir string, count int) ([]Migration, error) {
	applied, err := GetApplied(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if len(applied) == 0 {
		return nil, nil
	}

	if count > len(applied) {
		count = len(applied)
	}

	var rollbackable []Migration
	for i := len(applied) - 1; i >= len(applied)-count; i-- {
		m := applied[i]
		downName := strings.TrimSuffix(m.Name, ".sql") + ".down.sql"
		downPath := filepath.Join(migrationsDir, downName)

		content, err := os.ReadFile(downPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("down migration not found: %s", downName)
			}
			return nil, fmt.Errorf("failed to read down migration %s: %w", downName, err)
		}

		rollbackable = append(rollbackable, Migration{
			Name:    m.Name,
			Content: string(content),
		})
	}

	return rollbackable, nil
}

func Rollback(ctx context.Context, databaseURL string, m Migration) error {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, m.Content); err != nil {
		return fmt.Errorf("failed to execute rollback: %w", err)
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		DELETE FROM %s WHERE name = $1
	`, migrationsTable), m.Name); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return tx.Commit(ctx)
}
