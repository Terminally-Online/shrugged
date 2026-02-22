package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/terminally-online/shrugged/internal/updater"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update shrugged to the latest release",
	Long: `Download and install the latest shrugged release from GitHub.

The binary is verified against its SHA256 checksum before installation.
If shrugged is installed in a system directory, re-run with sudo.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if version == "dev" {
			fmt.Fprintln(cmd.OutOrStdout(), "Running a dev build — update is not available.")
			return nil
		}

		latest, outdated, err := updater.CheckForUpdate(version)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not check latest version: %v\n", err)
		}

		if !outdated && err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "shrugged %s is already up to date.\n", version)
			return nil
		}

		if latest == "" {
			release, fetchErr := updater.FetchLatestRelease()
			if fetchErr != nil {
				return fmt.Errorf("failed to determine latest release: %w", fetchErr)
			}
			latest = release.TagName
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updating shrugged %s → %s...\n", version, latest)

		if err := updater.DownloadAndReplace(latest); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated to %s.\n", latest)
		return nil
	},
}
