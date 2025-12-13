package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"shrugged/internal/migrate"
)

var (
	dryRun      bool
	forceApply  bool
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply pending migrations to the database",
	Long:  `Apply all pending migrations to the database in order. Use --dry-run to preview without applying.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		dbURL, err := cfg.GetDatabaseURL(&flags)
		if err != nil {
			return err
		}
		migrationsDir := cfg.GetMigrationsDir(&flags)

		if err := migrate.ValidateSum(migrationsDir); err != nil {
			if !forceApply {
				return fmt.Errorf("sum file validation failed: %w\nUse --force to apply anyway", err)
			}
			fmt.Printf("⚠ WARNING: %v\n", err)
		}

		modified, err := migrate.HasModifiedMigrations(ctx, dbURL, migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to check for modified migrations: %w", err)
		}

		if len(modified) > 0 && !forceApply {
			fmt.Println("WARNING: The following migrations have been modified after being applied:")
			for _, m := range modified {
				fmt.Printf("  ⚠ %s\n", m.Name)
			}
			fmt.Println()
			fmt.Println("This may indicate schema drift between your migrations and the database.")
			fmt.Println("Use --force to apply pending migrations anyway.")
			return fmt.Errorf("refusing to apply migrations with modified history")
		}

		pending, err := migrate.GetPending(ctx, dbURL, migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to get pending migrations: %w", err)
		}

		if len(pending) == 0 {
			fmt.Println("No pending migrations.")
			return nil
		}

		fmt.Printf("Found %d pending migration(s):\n", len(pending))
		for _, m := range pending {
			fmt.Printf("  - %s\n", m.Name)
		}

		if dryRun {
			fmt.Println("\nDry run mode. No changes applied.")
			return nil
		}

		fmt.Println()
		for _, m := range pending {
			fmt.Printf("Applying %s... ", m.Name)
			if err := migrate.Apply(ctx, dbURL, m); err != nil {
				fmt.Println("FAILED")
				return fmt.Errorf("failed to apply migration %s: %w", m.Name, err)
			}
			fmt.Println("OK")
		}

		return nil
	},
}

func init() {
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview migrations without applying")
	applyCmd.Flags().BoolVar(&forceApply, "force", false, "apply even if previous migrations have been modified")
}
