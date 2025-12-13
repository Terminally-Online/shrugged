package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateNamespace(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Namespaces: []parser.Namespace{
			{Name: "myschema", Owner: "admin"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateNamespace && c.ObjectName() == "myschema" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateNamespace change for myschema")
	}
}

func TestCompare_DropNamespace(t *testing.T) {
	current := &parser.Schema{
		Namespaces: []parser.Namespace{
			{Name: "oldschema"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropNamespace && c.ObjectName() == "oldschema" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropNamespace change for oldschema")
	}
}

func TestNamespaceChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *NamespaceChange
		want   []string
	}{
		{
			name: "create namespace",
			change: &NamespaceChange{
				ChangeType: CreateNamespace,
				Namespace:  parser.Namespace{Name: "myschema"},
			},
			want: []string{"CREATE SCHEMA", "myschema"},
		},
		{
			name: "create namespace with owner",
			change: &NamespaceChange{
				ChangeType: CreateNamespace,
				Namespace:  parser.Namespace{Name: "myschema", Owner: "admin"},
			},
			want: []string{"CREATE SCHEMA", "myschema", "AUTHORIZATION", "admin"},
		},
		{
			name: "drop namespace",
			change: &NamespaceChange{
				ChangeType: DropNamespace,
				Namespace:  parser.Namespace{Name: "oldschema"},
			},
			want: []string{"DROP SCHEMA", "oldschema"},
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

func TestNamespaceChange_DownSQL(t *testing.T) {
	createChange := &NamespaceChange{
		ChangeType: CreateNamespace,
		Namespace:  parser.Namespace{Name: "myschema"},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP SCHEMA") {
		t.Error("DownSQL for CreateNamespace should contain DROP SCHEMA")
	}

	dropChange := &NamespaceChange{
		ChangeType: DropNamespace,
		Namespace:  parser.Namespace{Name: "myschema", Owner: "admin"},
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE SCHEMA") {
		t.Error("DownSQL for DropNamespace should contain CREATE SCHEMA")
	}
	if !strings.Contains(downSQL, "AUTHORIZATION") {
		t.Error("DownSQL for DropNamespace should contain AUTHORIZATION for owned schema")
	}
}

func TestNamespaceChange_IsReversible(t *testing.T) {
	change := &NamespaceChange{
		ChangeType: CreateNamespace,
		Namespace:  parser.Namespace{Name: "test"},
	}
	if !change.IsReversible() {
		t.Error("NamespaceChange should be reversible")
	}
}
