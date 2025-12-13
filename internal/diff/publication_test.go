package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreatePublication(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Publications: []parser.Publication{
			{Name: "my_pub", AllTables: true},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreatePublication && c.ObjectName() == "my_pub" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreatePublication change for my_pub")
	}
}

func TestCompare_DropPublication(t *testing.T) {
	current := &parser.Schema{
		Publications: []parser.Publication{
			{Name: "old_pub", AllTables: true},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropPublication && c.ObjectName() == "old_pub" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropPublication change for old_pub")
	}
}

func TestPublicationChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *PublicationChange
		want   []string
	}{
		{
			name: "create publication for all tables",
			change: &PublicationChange{
				ChangeType: CreatePublication,
				Publication: parser.Publication{
					Name:      "all_tables_pub",
					AllTables: true,
				},
			},
			want: []string{"CREATE PUBLICATION", "all_tables_pub", "FOR ALL TABLES"},
		},
		{
			name: "create publication for specific tables",
			change: &PublicationChange{
				ChangeType: CreatePublication,
				Publication: parser.Publication{
					Name:   "specific_pub",
					Tables: []string{"users", "orders", "products"},
				},
			},
			want: []string{"CREATE PUBLICATION", "specific_pub", "FOR TABLE", "users", "orders", "products"},
		},
		{
			name: "create publication with no tables",
			change: &PublicationChange{
				ChangeType: CreatePublication,
				Publication: parser.Publication{
					Name: "empty_pub",
				},
			},
			want: []string{"CREATE PUBLICATION", "empty_pub"},
		},
		{
			name: "drop publication",
			change: &PublicationChange{
				ChangeType:  DropPublication,
				Publication: parser.Publication{Name: "old_pub"},
			},
			want: []string{"DROP PUBLICATION", "old_pub"},
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

func TestPublicationChange_DownSQL(t *testing.T) {
	createChange := &PublicationChange{
		ChangeType: CreatePublication,
		Publication: parser.Publication{
			Name:      "my_pub",
			AllTables: true,
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP PUBLICATION") {
		t.Error("DownSQL for CreatePublication should contain DROP PUBLICATION")
	}

	oldPub := parser.Publication{Name: "old_pub", Tables: []string{"users", "orders"}}
	dropChange := &PublicationChange{
		ChangeType:     DropPublication,
		Publication:    parser.Publication{Name: "old_pub"},
		OldPublication: &oldPub,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE PUBLICATION") {
		t.Error("DownSQL for DropPublication with OldPublication should contain CREATE PUBLICATION")
	}
	if !strings.Contains(downSQL, "FOR TABLE") {
		t.Error("DownSQL for DropPublication should preserve FOR TABLE")
	}

	dropChangeNoOld := &PublicationChange{
		ChangeType:  DropPublication,
		Publication: parser.Publication{Name: "old_pub"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropPublication without OldPublication should indicate IRREVERSIBLE")
	}
}

func TestPublicationChange_IsReversible(t *testing.T) {
	createChange := &PublicationChange{
		ChangeType: CreatePublication,
		Publication: parser.Publication{
			Name:      "test",
			AllTables: true,
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreatePublication should be reversible")
	}

	oldPub := parser.Publication{Name: "test", AllTables: true}
	dropChangeWithOld := &PublicationChange{
		ChangeType:     DropPublication,
		Publication:    parser.Publication{Name: "test"},
		OldPublication: &oldPub,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropPublication with OldPublication should be reversible")
	}

	dropChangeNoOld := &PublicationChange{
		ChangeType:  DropPublication,
		Publication: parser.Publication{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropPublication without OldPublication should not be reversible")
	}
}
