package cli

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/terminally-online/shrugged/internal/config"
	"github.com/terminally-online/shrugged/internal/updater"
)

var (
	cfgFile       string
	cfg           *config.Config
	flags         config.Flags
	version       = "dev"
	updateCheckWg sync.WaitGroup
)

var rootCmd = &cobra.Command{
	Use:   "shrugged",
	Short: "PostgreSQL schema migration tool",
	Long: `Shrugged is a PostgreSQL schema migration tool that provides
automatic schema diffing and migration generation.

No cloud dependencies. No paywalled features. Just migrations.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "update" {
			return nil
		}

		var err error
		if _, statErr := os.Stat(cfgFile); os.IsNotExist(statErr) {
			cfg = &config.Config{}
		} else {
			cfg, err = config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
		}

		updateCheckWg.Add(1)
		go func() {
			defer updateCheckWg.Done()
			latest, outdated, err := updater.CheckForUpdate(version)
			if err != nil || !outdated {
				return
			}
			fmt.Fprintf(os.Stderr, "\nA new version of shrugged is available: %s (you have %s)\nRun 'shrugged update' to upgrade.\n\n", latest, version)
		}()

		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("shrugged %s\n", version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "shrugged.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&flags.URL, "url", "", "database connection URL")
	rootCmd.PersistentFlags().StringVar(&flags.Schema, "schema", "", "path to schema file")
	rootCmd.PersistentFlags().StringVar(&flags.MigrationsDir, "migrations-dir", "", "path to migrations directory")
	rootCmd.PersistentFlags().StringVar(&flags.PostgresVersion, "postgres-version", "", "postgres version for Docker containers")

	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(sumCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
}

func SetVersion(v string) {
	version = v
}

func Execute() error {
	err := rootCmd.Execute()
	updateCheckWg.Wait()
	return err
}

func Root() *cobra.Command {
	return rootCmd
}
