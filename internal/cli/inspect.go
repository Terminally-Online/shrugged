package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"git.ca.plug.to/terminally-online/shrugged/internal/introspect"
	"git.ca.plug.to/terminally-online/shrugged/internal/ui"
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

		dbURL, err := cfg.GetDatabaseURL(&flags)
		if err != nil {
			return err
		}

		s := ui.NewSpinner()
		s.Start("Introspecting database...")
		schema, err := introspect.Database(ctx, dbURL)
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to introspect database: %w", err)
		}
		s.Stop()

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
