package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/terminally-online/shrugged/internal/migrate"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long:  `Display the status of all migrations, showing which have been applied and which are pending.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, err := cfg.GetDatabaseURL(&flags)
		if err != nil {
			return err
		}
		migrationsDir := cfg.GetMigrationsDir(&flags)

		applied, err := migrate.GetAppliedWithStatus(cmd.Context(), dbURL, migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to get applied migrations: %w", err)
		}

		pending, err := migrate.GetPending(cmd.Context(), dbURL, migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to get pending migrations: %w", err)
		}

		if len(applied) == 0 && len(pending) == 0 {
			fmt.Println("No migrations found.")
			return nil
		}

		var modifiedCount int
		if len(applied) > 0 {
			fmt.Println("Applied migrations:")
			for _, m := range applied {
				if m.Modified {
					fmt.Printf("  ⚠ %s (applied %s) MODIFIED\n", m.Name, m.AppliedAt.Format("2006-01-02 15:04:05"))
					modifiedCount++
				} else {
					fmt.Printf("  ✓ %s (applied %s)\n", m.Name, m.AppliedAt.Format("2006-01-02 15:04:05"))
				}
			}
		}

		if len(pending) > 0 {
			if len(applied) > 0 {
				fmt.Println()
			}
			fmt.Println("Pending migrations:")
			for _, m := range pending {
				fmt.Printf("  ○ %s\n", m.Name)
			}
		}

		if modifiedCount > 0 {
			fmt.Println()
			fmt.Printf("WARNING: %d migration(s) have been modified after being applied.\n", modifiedCount)
			fmt.Println("This may indicate schema drift. Consider reviewing these changes.")
		}

		return nil
	},
}
