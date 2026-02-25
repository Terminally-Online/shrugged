package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/terminally-online/shrugged/internal/docker"
	"github.com/terminally-online/shrugged/internal/introspect"
	"github.com/terminally-online/shrugged/internal/parser"
	"github.com/terminally-online/shrugged/internal/ui"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the schema file",
	Long: `Validate the schema file by applying it to a temporary Postgres container.

This ensures the SQL is syntactically correct and can be executed against
the configured Postgres version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		postgresVersion := cfg.GetPostgresVersion(&flags)
		schemaFile := cfg.GetSchema(&flags)

		schemaSQL, err := parser.LoadSchema(schemaFile)
		if err != nil {
			return fmt.Errorf("failed to load schema file: %w", err)
		}

		dockerCfg := docker.PostgresConfig{
			Version:  postgresVersion,
			User:     "shrugged",
			Password: "shrugged",
			Database: "shrugged",
		}

		s := ui.NewSpinner()
		s.Start("Validating schema...")
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

		s.Update("Applying schema...")
		if err := docker.ExecuteSQL(ctx, container, schemaSQL); err != nil {
			s.Stop()
			return fmt.Errorf("schema validation failed: %w", err)
		}

		s.Update("Introspecting schema...")
		schema, err := introspect.Database(ctx, container.ConnectionString())
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to introspect schema: %w", err)
		}

		s.Stop()

		warnings := schema.Lint()
		if len(warnings) > 0 {
			fmt.Println("Warnings:")
			for _, w := range warnings {
				fmt.Printf("  - %s\n", w)
			}
			fmt.Println()
		}

		fmt.Printf("Schema is valid. Found %d object(s).\n", schema.ObjectCount())
		return nil
	},
}
