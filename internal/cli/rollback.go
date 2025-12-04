package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"shrugged/internal/migrate"
)

var (
	rollbackCount  int
	rollbackDryRun bool
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the last applied migration(s)",
	Long:  `Rollback one or more migrations using their corresponding .down.sql files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		rollbackable, err := migrate.GetRollbackable(ctx, cfg.DatabaseURL, cfg.MigrationsDir, rollbackCount)
		if err != nil {
			return fmt.Errorf("failed to get rollbackable migrations: %w", err)
		}

		if len(rollbackable) == 0 {
			fmt.Println("No migrations to rollback.")
			return nil
		}

		fmt.Printf("Found %d migration(s) to rollback:\n", len(rollbackable))
		for _, m := range rollbackable {
			fmt.Printf("  - %s\n", m.Name)
		}

		if rollbackDryRun {
			fmt.Println("\nDry run mode. No changes applied.")
			fmt.Println("\nRollback SQL that would be executed:")
			for _, m := range rollbackable {
				fmt.Printf("\n-- Rollback %s\n%s\n", m.Name, m.Content)
			}
			return nil
		}

		fmt.Println()
		for _, m := range rollbackable {
			fmt.Printf("Rolling back %s... ", m.Name)
			if err := migrate.Rollback(ctx, cfg.DatabaseURL, m); err != nil {
				fmt.Println("FAILED")
				return fmt.Errorf("failed to rollback migration %s: %w", m.Name, err)
			}
			fmt.Println("OK")
		}

		fmt.Printf("\nSuccessfully rolled back %d migration(s).\n", len(rollbackable))
		return nil
	},
}

func init() {
	rollbackCmd.Flags().IntVarP(&rollbackCount, "count", "n", 1, "number of migrations to rollback")
	rollbackCmd.Flags().BoolVar(&rollbackDryRun, "dry-run", false, "preview rollback without executing")
}
