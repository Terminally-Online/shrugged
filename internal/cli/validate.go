package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"shrugged/internal/docker"
	"shrugged/internal/introspect"
	"shrugged/internal/parser"
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

		schemaSQL, err := parser.LoadFile(schemaFile)
		if err != nil {
			return fmt.Errorf("failed to load schema file: %w", err)
		}

		dockerCfg := docker.PostgresConfig{
			Version:  postgresVersion,
			User:     "shrugged",
			Password: "shrugged",
			Database: "shrugged",
		}

		fmt.Printf("Starting Postgres %s container...\n", postgresVersion)
		container, err := docker.StartPostgres(ctx, dockerCfg)
		if err != nil {
			return fmt.Errorf("failed to start postgres: %w", err)
		}
		defer func() {
			fmt.Println("Stopping container...")
			_ = docker.StopContainer(context.Background(), container.ID)
		}()

		fmt.Println("Applying schema file...")
		if err := docker.ExecuteSQL(ctx, container, schemaSQL); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}

		fmt.Println("Introspecting schema...")
		schema, err := introspect.Database(ctx, container.ConnectionString())
		if err != nil {
			return fmt.Errorf("failed to introspect schema: %w", err)
		}

		warnings := schema.Lint()
		if len(warnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, w := range warnings {
				fmt.Printf("  - %s\n", w)
			}
			fmt.Println()
		}

		fmt.Printf("Schema is valid. Found %d object(s).\n", schema.ObjectCount())
		return nil
	},
}
