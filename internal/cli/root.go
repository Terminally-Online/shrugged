package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"shrugged/internal/config"
)

var (
	cfgFile string
	cfg     *config.Config
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "shrugged",
	Short: "PostgreSQL schema migration tool",
	Long: `Shrugged is a PostgreSQL schema migration tool that provides
automatic schema diffing and migration generation.

No cloud dependencies. No paywalled features. Just migrations.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
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

	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
}

func SetVersion(v string) {
	version = v
}

func Execute() error {
	return rootCmd.Execute()
}

func Root() *cobra.Command {
	return rootCmd
}
