package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"shrugged/internal/introspect"
)

var (
	outputFile string
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Dump the current database schema",
	Long:  `Inspect the live database and output the current schema as SQL.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		fmt.Println("Connecting to database...")
		schema, err := introspect.Database(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("failed to introspect database: %w", err)
		}

		sql := schema.ToSQL()

		if outputFile != "" {
			if err := os.WriteFile(outputFile, []byte(sql), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Schema written to %s\n", outputFile)
		} else {
			fmt.Println(sql)
		}

		return nil
	},
}

func init() {
	inspectCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default: stdout)")
}
