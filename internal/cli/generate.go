package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"git.ca.plug.to/terminally-online/shrugged/internal/codegen"
	_ "git.ca.plug.to/terminally-online/shrugged/internal/codegen/golang"
	"git.ca.plug.to/terminally-online/shrugged/internal/docker"
	"git.ca.plug.to/terminally-online/shrugged/internal/introspect"
	"git.ca.plug.to/terminally-online/shrugged/internal/parser"
	"git.ca.plug.to/terminally-online/shrugged/internal/ui"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate language bindings from database schema",
	Long: `Generate native language bindings (models/types) from the database schema.

The generator introspects the database and creates type-safe models for tables,
enums, and composite types in the specified language.

If no database URL is provided, a temporary Postgres container is started and
the schema file is applied automatically.

Use --clean to remove orphaned query files that no longer have corresponding SQL queries.

Example:
  shrugged generate --language go --out ./models
  shrugged generate --url postgres://localhost/mydb --language go --out ./models
  shrugged generate --clean`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		language := cfg.GetLanguage(&flags)
		outDir := cfg.GetOut(&flags)

		generator, err := codegen.Get(language)
		if err != nil {
			return fmt.Errorf("failed to get generator: %w (available: %v)", err, codegen.Languages())
		}

		dbURL, err := cfg.GetDatabaseURL(&flags)
		useEphemeral := err != nil || dbURL == ""

		s := ui.NewSpinner()

		var container *docker.Container
		if useEphemeral {
			schemaFile := cfg.GetSchema(&flags)
			if schemaFile == "" {
				return fmt.Errorf("schema file is required when no database URL is provided")
			}

			schemaSQL, err := parser.LoadSchema(schemaFile)
			if err != nil {
				return fmt.Errorf("failed to load schema file: %w", err)
			}

			postgresVersion := cfg.GetPostgresVersion(&flags)
			dockerCfg := docker.PostgresConfig{
				Version:  postgresVersion,
				User:     "shrugged",
				Password: "shrugged",
				Database: "shrugged",
			}

			s.Start("Starting Postgres container...")
			container, err = docker.StartPostgres(ctx, dockerCfg)
			if err != nil {
				s.Stop()
				return fmt.Errorf("failed to start postgres: %w", err)
			}
			defer func() {
				s.Start("Stopping container...")
				_ = docker.StopContainer(context.Background(), container.ID)
				s.Stop()
			}()

			s.Update("Applying schema...")
			if err := docker.ExecuteSQL(ctx, container, schemaSQL); err != nil {
				s.Stop()
				return fmt.Errorf("failed to apply schema: %w", err)
			}

			dbURL = container.ConnectionString()
		}

		s.Start("Introspecting database...")
		schema, err := introspect.Database(ctx, dbURL)
		if err != nil {
			s.Stop()
			return fmt.Errorf("failed to introspect database: %w", err)
		}

		s.Update(fmt.Sprintf("Generating %s models...", language))
		if err := generator.Generate(schema, outDir); err != nil {
			s.Stop()
			return fmt.Errorf("failed to generate: %w", err)
		}

		tableCount := len(schema.Tables)
		enumCount := len(schema.Enums)
		compositeCount := len(schema.CompositeTypes)

		queriesPath := cfg.GetQueries(&flags)
		if queriesPath != "" {
			queriesOutDir := cfg.GetQueriesOut(&flags)

			s.Update("Parsing queries...")
			queryFiles, err := parser.ParseQueries(queriesPath)
			if err != nil {
				s.Stop()
				return fmt.Errorf("failed to parse queries: %w", err)
			}

			queries := parser.GetAllQueries(queryFiles)
			if len(queries) == 0 {
				s.Stop()
				fmt.Println("No queries found.")
			} else {
				s.Update(fmt.Sprintf("Introspecting %d queries...", len(queries)))
				queries, err = introspect.Queries(ctx, dbURL, queries, schema)
				if err != nil {
					s.Stop()
					return fmt.Errorf("failed to introspect queries: %w", err)
				}

				modelsPackage := determineModelsPackage(outDir)
				clean := cfg.GetClean(&flags)
				s.Update("Generating query bindings...")
				removed, err := generator.GenerateQueries(codegen.QueryGenOptions{
					Queries:       queries,
					OutDir:        queriesOutDir,
					ModelsPackage: modelsPackage,
					ModelsDir:     outDir,
					Schema:        schema,
					Clean:         clean,
				})
				if err != nil {
					s.Stop()
					return fmt.Errorf("failed to generate queries: %w", err)
				}

				s.Stop()
				fmt.Printf("Generated %d tables, %d enums, %d composite types, %d query functions\n", tableCount, enumCount, compositeCount, len(queries))
				if len(removed) > 0 {
					fmt.Printf("Removed %d orphaned files\n", len(removed))
				}
				return nil
			}
		} else {
			s.Stop()
		}

		fmt.Printf("Generated %d tables, %d enums, %d composite types\n", tableCount, enumCount, compositeCount)

		return nil
	},
}

func init() {
	generateCmd.Flags().StringVar(&flags.Out, "out", "", "output directory for generated files")
	generateCmd.Flags().StringVar(&flags.Language, "language", "", "target language (default: go)")
	generateCmd.Flags().StringVar(&flags.Queries, "queries", "", "path to queries file or directory")
	generateCmd.Flags().StringVar(&flags.QueriesOut, "queries-out", "", "output directory for query bindings")
	generateCmd.Flags().BoolVar(&flags.Clean, "clean", false, "remove orphaned query files that no longer have corresponding SQL queries")
}

func determineModelsPackage(outDir string) string {
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return filepath.Base(outDir)
	}

	modPath, modDir := findModulePathFromDir(absOut)
	if modPath == "" {
		return filepath.Base(outDir)
	}

	relPath, err := filepath.Rel(modDir, absOut)
	if err != nil {
		return filepath.Base(outDir)
	}

	if relPath == "." {
		return modPath
	}

	return modPath + "/" + filepath.ToSlash(relPath)
}

func findModulePathFromDir(startDir string) (modPath string, modDir string) {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(goModPath)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if after, ok := strings.CutPrefix(line, "module "); ok {
					return strings.TrimSpace(after), dir
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}
		dir = parent
	}
}
