package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"shrugged/internal/diff"
	"shrugged/internal/docker"
	"shrugged/internal/introspect"
	"shrugged/internal/migrate"
	"shrugged/internal/parser"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Generate a migration from schema differences",
	Long: `Compare the schema file to the migrations and generate a new migration file.

This spins up a temporary Postgres container, applies all existing migrations,
then diffs against the desired schema to produce a new migration.`,
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

		fmt.Println("Starting Postgres container...")
		container, err := docker.StartPostgres(ctx, dockerCfg)
		if err != nil {
			return fmt.Errorf("failed to start postgres: %w", err)
		}
		defer func() {
			fmt.Println("Stopping container...")
			_ = docker.StopContainer(context.Background(), container.ID)
		}()

		currentSchema, err := buildCurrentState(ctx, container, migrationsDir)
		if err != nil {
			return err
		}

		schemaSQL, err := parser.LoadFile(schemaFile)
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
			fmt.Println("\nNo changes detected. Nothing to migrate.")
			return nil
		}

		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			return fmt.Errorf("failed to create migrations directory: %w", err)
		}

		timestamp := time.Now().UTC().Format("20060102150405")
		upFilename := filepath.Join(migrationsDir, fmt.Sprintf("%s.sql", timestamp))
		downFilename := filepath.Join(migrationsDir, fmt.Sprintf("%s.down.sql", timestamp))

		upFile, err := os.Create(upFilename)
		if err != nil {
			return fmt.Errorf("failed to create migration file: %w", err)
		}
		defer func() { _ = upFile.Close() }()

		for _, change := range changes {
			if _, err := upFile.WriteString(change.SQL() + "\n\n"); err != nil {
				return fmt.Errorf("failed to write migration: %w", err)
			}
		}

		downFile, err := os.Create(downFilename)
		if err != nil {
			return fmt.Errorf("failed to create down migration file: %w", err)
		}
		defer func() { _ = downFile.Close() }()

		var hasIrreversible bool
		for i := len(changes) - 1; i >= 0; i-- {
			change := changes[i]
			downSQL := change.DownSQL()
			if !change.IsReversible() {
				hasIrreversible = true
			}
			if _, err := downFile.WriteString(downSQL + "\n\n"); err != nil {
				return fmt.Errorf("failed to write down migration: %w", err)
			}
		}

		if err := migrate.UpdateSum(migrationsDir); err != nil {
			return fmt.Errorf("failed to update sum file: %w", err)
		}

		fmt.Printf("\nCreated migration: %s\n", upFilename)
		fmt.Printf("Created rollback:  %s\n", downFilename)
		fmt.Printf("Contains %d change(s)\n", len(changes))
		if hasIrreversible {
			fmt.Println("\n⚠ WARNING: Some changes are not fully reversible. Review the down migration carefully.")
		}
		return nil
	},
}
