package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// deployCompose is the compose file the shrugged Coolify application deploys:
// the API and the Postgres it diffs in.
const deployCompose = "../../docker-compose.yaml"

// databasePassword is the variable the database superuser's password comes
// from, set on the shrugged Coolify application.
const databasePassword = "POSTGRES_PASSWORD"

// requiredInterpolation matches a compose value that is exactly one variable
// which must be set: `${NAME:?message}` or `${NAME?message}`.
var requiredInterpolation = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*):?\?[^}]*\}$`)

// urlPassword captures the password of a postgres URL.
var urlPassword = regexp.MustCompile(`^postgres(?:ql)?://[^:@/]+:(.*)@[^@]+$`)

// environment is a service's environment in either compose form: a mapping of
// names to values, or a list of NAME=value entries.
type environment map[string]string

func (e *environment) UnmarshalYAML(node *yaml.Node) error {
	values := map[string]string{}
	switch node.Kind {
	case yaml.MappingNode:
		if err := node.Decode(&values); err != nil {
			return err
		}
	case yaml.SequenceNode:
		var entries []string
		if err := node.Decode(&entries); err != nil {
			return err
		}
		for _, entry := range entries {
			name, value, _ := strings.Cut(entry, "=")
			values[name] = value
		}
	default:
		return fmt.Errorf("environment is neither a mapping nor a list")
	}
	*e = values
	return nil
}

// TestDatabasePasswordComesFromItsDeployment pins where the database
// superuser's password lives. The file is public in the repository, so a
// password written into it is everyone's; it belongs to the deployment, set on
// the shrugged Coolify application. Postgres initialises its cluster with it
// and the API dials with it, so both must be a required interpolation of the
// one variable: a literal or a `${NAME:-default}` puts a known password on the
// database, where a required one refuses to start without it.
func TestDatabasePasswordComesFromItsDeployment(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash(deployCompose))
	if err != nil {
		t.Fatalf("read %s: %v", deployCompose, err)
	}
	var parsed struct {
		Services map[string]struct {
			Environment environment `yaml:"environment"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parse %s: %v", deployCompose, err)
	}

	postgres, ok := parsed.Services["postgres"]
	if !ok {
		t.Fatalf("%s: no postgres service", deployCompose)
	}
	password := postgres.Environment[databasePassword]
	if match := requiredInterpolation.FindStringSubmatch(password); match == nil || match[1] != databasePassword {
		t.Errorf("%s: postgres sets %s to %q; it must be ${%s:?...}, taken from the deployment and refused when"+
			" missing", deployCompose, databasePassword, password, databasePassword)
	}

	api, ok := parsed.Services["shrugged-api"]
	if !ok {
		t.Fatalf("%s: no shrugged-api service", deployCompose)
	}
	dial := api.Environment["DATABASE_URL"]
	credential := urlPassword.FindStringSubmatch(dial)
	if credential == nil {
		t.Fatalf("%s: shrugged-api's DATABASE_URL %q carries no password", deployCompose, dial)
	}
	if match := requiredInterpolation.FindStringSubmatch(credential[1]); match == nil || match[1] != databasePassword {
		t.Errorf("%s: shrugged-api dials with password %q; it must be ${%s:?...}, the password postgres holds",
			deployCompose, credential[1], databasePassword)
	}
}
