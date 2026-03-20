package cli

import (
	"strings"
	"testing"
)

// runCmd executes the root command with the given arguments and returns the
// error. Uses cobra's built-in error capture so we don't depend on how each
// command writes its output.
func runCmd(args ...string) error {
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	return err
}

// TestPersistentFlags verifies that the global flags are defined on the root command.
func TestPersistentFlags(t *testing.T) {
	expected := []string{"url", "schema", "migrations-dir", "postgres-version", "config"}
	for _, name := range expected {
		if f := rootCmd.PersistentFlags().Lookup(name); f == nil {
			t.Errorf("persistent flag --%s not defined", name)
		}
	}
}

// TestGenerateFlags verifies that the generate subcommand exposes its unique flags.
func TestGenerateFlags(t *testing.T) {
	expected := []string{"out", "language", "queries", "queries-out", "clean"}
	for _, name := range expected {
		if f := generateCmd.Flags().Lookup(name); f == nil {
			t.Errorf("generate flag --%s not defined", name)
		}
	}
}

// TestApplyFlags verifies that the apply subcommand exposes its unique flags.
func TestApplyFlags(t *testing.T) {
	expected := []string{"dry-run", "force"}
	for _, name := range expected {
		if f := applyCmd.Flags().Lookup(name); f == nil {
			t.Errorf("apply flag --%s not defined", name)
		}
	}
}

// TestApplyMissingURL verifies that "apply" fails with a clear error when no
// database URL is provided and no config file exists.
func TestApplyMissingURL(t *testing.T) {
	err := runCmd("apply", "--config", "/nonexistent/shrugged.yaml")
	if err == nil {
		t.Fatal("expected error when url is missing, got nil")
	}
	if !strings.Contains(err.Error(), "database_url") {
		t.Errorf("error does not mention database_url: %v", err)
	}
}

// TestStatusMissingURL verifies that "status" fails with a clear error when no
// database URL is provided.
func TestStatusMissingURL(t *testing.T) {
	err := runCmd("status", "--config", "/nonexistent/shrugged.yaml")
	if err == nil {
		t.Fatal("expected error when url is missing, got nil")
	}
	if !strings.Contains(err.Error(), "database_url") {
		t.Errorf("error does not mention database_url: %v", err)
	}
}

// TestVersionCommand verifies that "version" runs without error.
func TestVersionCommand(t *testing.T) {
	if err := runCmd("version"); err != nil {
		t.Fatalf("version command returned error: %v", err)
	}
}

// TestSubcommandHelp verifies that each major subcommand exposes --help.
// NOTE: Run this after error-path tests to avoid cobra flag state bleed.
func TestSubcommandHelp(t *testing.T) {
	subcommands := []string{"diff", "migrate", "apply", "validate", "generate", "status", "rollback"}
	for _, sub := range subcommands {
		t.Run(sub, func(t *testing.T) {
			err := runCmd(sub, "--help")
			// cobra returns nil for --help invocations.
			if err != nil {
				t.Errorf("%s --help returned error: %v", sub, err)
			}
		})
	}
}
