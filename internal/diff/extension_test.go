package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateExtension(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Extensions: []parser.Extension{
			{Name: "uuid-ossp", Schema: "public"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateExtension && c.ObjectName() == "uuid-ossp" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateExtension change for uuid-ossp")
	}
}

func TestCompare_DropExtension(t *testing.T) {
	current := &parser.Schema{
		Extensions: []parser.Extension{
			{Name: "hstore"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropExtension && c.ObjectName() == "hstore" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropExtension change for hstore")
	}
}

func TestExtensionChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *ExtensionChange
		want   []string
	}{
		{
			name: "create extension",
			change: &ExtensionChange{
				ChangeType: CreateExtension,
				Extension:  parser.Extension{Name: "uuid-ossp"},
			},
			want: []string{"CREATE EXTENSION IF NOT EXISTS", "uuid-ossp"},
		},
		{
			name: "create extension with schema",
			change: &ExtensionChange{
				ChangeType: CreateExtension,
				Extension:  parser.Extension{Name: "postgis", Schema: "geo"},
			},
			want: []string{"CREATE EXTENSION IF NOT EXISTS", "postgis", "SCHEMA", "geo"},
		},
		{
			name: "drop extension",
			change: &ExtensionChange{
				ChangeType: DropExtension,
				Extension:  parser.Extension{Name: "hstore"},
			},
			want: []string{"DROP EXTENSION IF EXISTS", "hstore"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.change.SQL()
			for _, want := range tt.want {
				if !strings.Contains(sql, want) {
					t.Errorf("SQL = %q, should contain %q", sql, want)
				}
			}
		})
	}
}

func TestExtensionChange_DownSQL(t *testing.T) {
	createChange := &ExtensionChange{
		ChangeType: CreateExtension,
		Extension:  parser.Extension{Name: "uuid-ossp"},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP EXTENSION IF EXISTS") {
		t.Error("DownSQL for CreateExtension should contain DROP EXTENSION IF EXISTS")
	}

	dropChange := &ExtensionChange{
		ChangeType: DropExtension,
		Extension:  parser.Extension{Name: "postgis", Schema: "geo"},
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE EXTENSION IF NOT EXISTS") {
		t.Error("DownSQL for DropExtension should contain CREATE EXTENSION IF NOT EXISTS")
	}
	if !strings.Contains(downSQL, "SCHEMA") {
		t.Error("DownSQL for DropExtension should contain SCHEMA for non-public schema")
	}
}

func TestExtensionChange_IsReversible(t *testing.T) {
	change := &ExtensionChange{
		ChangeType: CreateExtension,
		Extension:  parser.Extension{Name: "test"},
	}
	if !change.IsReversible() {
		t.Error("ExtensionChange should be reversible")
	}
}
