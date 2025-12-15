package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/terminally-online/shrugged/internal/codegen"
	_ "github.com/terminally-online/shrugged/internal/codegen/golang"
	"github.com/terminally-online/shrugged/internal/introspect"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate language bindings from database schema",
	Long: `Generate native language bindings (models/types) from the database schema.

The generator introspects the database and creates type-safe models for tables,
enums, and composite types in the specified language.

Example:
  shrugged generate --url postgres://localhost/mydb --language go --out ./models`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		dbURL, err := cfg.GetDatabaseURL(&flags)
		if err != nil {
			return err
		}

		language := cfg.GetLanguage(&flags)
		outDir := cfg.GetOut(&flags)

		generator, err := codegen.Get(language)
		if err != nil {
			return fmt.Errorf("failed to get generator: %w (available: %v)", err, codegen.Languages())
		}

		fmt.Printf("Connecting to database...\n")
		schema, err := introspect.Database(ctx, dbURL)
		if err != nil {
			return fmt.Errorf("failed to introspect database: %w", err)
		}

		fmt.Printf("Generating %s models to %s...\n", language, outDir)
		if err := generator.Generate(schema, outDir); err != nil {
			return fmt.Errorf("failed to generate: %w", err)
		}

		tableCount := len(schema.Tables)
		enumCount := len(schema.Enums)
		compositeCount := len(schema.CompositeTypes)

		fmt.Printf("Generated %d tables, %d enums, %d composite types\n", tableCount, enumCount, compositeCount)

		return nil
	},
}

func init() {
	generateCmd.Flags().StringVar(&flags.Out, "out", "", "output directory for generated files")
	generateCmd.Flags().StringVar(&flags.Language, "language", "", "target language (default: go)")
}
