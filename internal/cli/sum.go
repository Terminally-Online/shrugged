package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"git.ca.plug.to/terminally-online/shrugged/internal/migrate"
)

var sumCmd = &cobra.Command{
	Use:   "sum",
	Short: "Regenerate the sum file from migration files",
	Long: `Regenerate shrugged.sum by hashing all migration files on disk.

Use this after resolving merge conflicts that leave the sum file
out of sync with the actual migration files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		migrationsDir := cfg.GetMigrationsDir(&flags)

		if err := migrate.UpdateSum(migrationsDir); err != nil {
			return fmt.Errorf("failed to update sum file: %w", err)
		}

		entries, err := migrate.GenerateSum(migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to read migrations: %w", err)
		}

		fmt.Printf("Regenerated shrugged.sum with %d migration files.\n", len(entries))
		return nil
	},
}
