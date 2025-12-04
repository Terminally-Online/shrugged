package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"shrugged/internal/diff"
	"shrugged/internal/docker"
	"shrugged/internal/introspect"
	"shrugged/internal/parser"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show differences between schema file and migrations",
	Long: `Compare the declarative schema file against the result of applying all migrations.

This spins up a temporary Postgres container, applies all migrations to get the
"current" state, then compares against the desired schema file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		dockerCfg := docker.PostgresConfig{
			Version:  cfg.PostgresVersion,
			User:     "shrugged",
			Password: "shrugged",
			Database: "shrugged",
		}

		fmt.Println("Starting Postgres container...")
		container, err := docker.StartPostgres(ctx, dockerCfg)
		if err != nil {
			return fmt.Errorf("failed to start postgres: %w", err)
		}
		defer func() {
			fmt.Println("Stopping container...")
			_ = docker.StopContainer(context.Background(), container.ID)
		}()

		currentSchema, err := buildCurrentState(ctx, container, cfg.MigrationsDir)
		if err != nil {
			return err
		}

		schemaSQL, err := parser.LoadFile(cfg.Schema)
		if err != nil {
			return fmt.Errorf("failed to load schema file: %w", err)
		}

		fmt.Println("Resetting database for schema application...")
		if err := docker.ResetDatabase(ctx, container); err != nil {
			return fmt.Errorf("failed to reset database: %w", err)
		}

		fmt.Println("Applying schema file...")
		if err := docker.ExecuteSQL(ctx, container, schemaSQL); err != nil {
			return fmt.Errorf("failed to apply schema: %w", err)
		}

		fmt.Println("Introspecting desired state...")
		desiredSchema, err := introspect.Database(ctx, container.ConnectionString())
		if err != nil {
			return fmt.Errorf("failed to introspect desired state: %w", err)
		}

		changes := diff.Compare(currentSchema, desiredSchema)

		if len(changes) == 0 {
			fmt.Println("\nNo changes detected. Schema is in sync with migrations.")
			return nil
		}

		fmt.Printf("\nFound %d change(s):\n\n", len(changes))
		for _, change := range changes {
			fmt.Println(change.SQL())
			fmt.Println()
		}

		return nil
	},
}

func buildCurrentState(ctx context.Context, container *docker.Container, migrationsDir string) (*parser.Schema, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No migrations directory found, starting from empty database...")
			return &parser.Schema{}, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	sqlCount := countSQLFiles(entries)
	if sqlCount > 0 {
		fmt.Printf("Applying %d migration(s)...\n", sqlCount)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		path := filepath.Join(migrationsDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		if err := docker.ExecuteSQL(ctx, container, string(content)); err != nil {
			return nil, fmt.Errorf("failed to apply migration %s: %w", entry.Name(), err)
		}
	}

	fmt.Println("Introspecting current state...")
	return introspect.Database(ctx, container.ConnectionString())
}

func countSQLFiles(entries []os.DirEntry) int {
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			count++
		}
	}
	return count
}
