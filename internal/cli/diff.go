package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"git.ca.plug.to/terminally-online/shrugged/internal/diff"
	"git.ca.plug.to/terminally-online/shrugged/internal/docker"
	"git.ca.plug.to/terminally-online/shrugged/internal/introspect"
	"git.ca.plug.to/terminally-online/shrugged/internal/parser"
	"git.ca.plug.to/terminally-online/shrugged/internal/ui"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show differences between schema file and migrations",
	Long: `Compare the declarative schema file against the result of applying all migrations.

This spins up a temporary Postgres container, applies all migrations to get the
"current" state, then compares against the desired schema file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		postgresVersion := cfg.GetPostgresVersion(&flags)
		migrationsDir := cfg.GetMigrationsDir(&flags)
		schemaFile := cfg.GetSchema(&flags)

		dockerCfg := docker.PostgresConfig{
			Version:  postgresVersion,
			User:     "shrugged",
			Password: "shrugged",
			Database: "shrugged",
		}

		s := ui.NewSpinner()
		s.Start("Starting Postgres container...")
		container, err := docker.StartPostgres(ctx, dockerCfg)
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to start postgres: %w", err)
		}
		defer func() {
			s.Start("Stopping container...")
			_ = docker.StopContainer(context.Background(), container.ID)
			s.Stop()
		}()

		currentSchema, err := buildCurrentState(ctx, s, container, migrationsDir)
		if err != nil {
			s.Stop()
			return err
		}

		schemaSQL, err := parser.LoadSchema(schemaFile)
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to load schema file: %w", err)
		}

		s.Update("Applying schema...")
		if err := docker.ResetDatabase(ctx, container); err != nil {
			s.Stop()
			return fmt.Errorf("failed to reset database: %w", err)
		}

		if err := docker.ExecuteSQL(ctx, container, schemaSQL); err != nil {
			s.Stop()
			return fmt.Errorf("failed to apply schema: %w", err)
		}

		s.Update("Introspecting desired state...")
		desiredSchema, err := introspect.Database(ctx, container.ConnectionString())
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to introspect desired state: %w", err)
		}

		s.Stop()

		changes := diff.Compare(currentSchema, desiredSchema)

		if len(changes) == 0 {
			fmt.Println("No changes detected. Schema is in sync with migrations.")
			return nil
		}

		fmt.Printf("Found %d change(s):\n\n", len(changes))
		for _, change := range changes {
			fmt.Println(change.SQL())
			fmt.Println()
		}

		return nil
	},
}

func buildCurrentState(ctx context.Context, s *ui.Spinner, container *docker.Container, migrationsDir string) (*parser.Schema, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &parser.Schema{}, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	sqlCount := countSQLFiles(entries)
	if sqlCount > 0 {
		s.Update(fmt.Sprintf("Applying %d migration(s)...", sqlCount))
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".sql" || strings.HasSuffix(name, ".down.sql") {
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

	s.Update("Introspecting current state...")
	return introspect.Database(ctx, container.ConnectionString())
}

func countSQLFiles(entries []os.DirEntry) int {
	count := 0
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && filepath.Ext(name) == ".sql" && !strings.HasSuffix(name, ".down.sql") {
			count++
		}
	}
	return count
}
