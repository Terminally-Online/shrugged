package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateCompositeType(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		CompositeTypes: []parser.CompositeType{
			{
				Name: "address",
				Attributes: []parser.Column{
					{Name: "street", Type: "text"},
					{Name: "city", Type: "text"},
					{Name: "zip", Type: "text"},
				},
			},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateCompositeType && c.ObjectName() == "address" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateCompositeType change for address")
	}
}

func TestCompare_DropCompositeType(t *testing.T) {
	current := &parser.Schema{
		CompositeTypes: []parser.CompositeType{
			{Name: "old_type", Attributes: []parser.Column{{Name: "field", Type: "text"}}},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropCompositeType && c.ObjectName() == "old_type" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropCompositeType change for old_type")
	}
}

func TestCompositeTypeChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *CompositeTypeChange
		want   []string
	}{
		{
			name: "create composite type",
			change: &CompositeTypeChange{
				ChangeType: CreateCompositeType,
				CompositeType: parser.CompositeType{
					Name: "person",
					Attributes: []parser.Column{
						{Name: "first_name", Type: "text"},
						{Name: "last_name", Type: "text"},
						{Name: "age", Type: "integer"},
					},
				},
			},
			want: []string{"CREATE TYPE", "person", "AS", "first_name", "last_name", "age", "text", "integer"},
		},
		{
			name: "create composite type with schema",
			change: &CompositeTypeChange{
				ChangeType: CreateCompositeType,
				CompositeType: parser.CompositeType{
					Schema: "myschema",
					Name:   "my_type",
					Attributes: []parser.Column{
						{Name: "field", Type: "text"},
					},
				},
			},
			want: []string{"CREATE TYPE", "myschema", "my_type"},
		},
		{
			name: "drop composite type",
			change: &CompositeTypeChange{
				ChangeType:    DropCompositeType,
				CompositeType: parser.CompositeType{Name: "old_type"},
			},
			want: []string{"DROP TYPE", "old_type"},
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

func TestCompositeTypeChange_DownSQL(t *testing.T) {
	createChange := &CompositeTypeChange{
		ChangeType: CreateCompositeType,
		CompositeType: parser.CompositeType{
			Name:       "my_type",
			Attributes: []parser.Column{{Name: "field", Type: "text"}},
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP TYPE") {
		t.Error("DownSQL for CreateCompositeType should contain DROP TYPE")
	}

	oldType := parser.CompositeType{
		Name: "old_type",
		Attributes: []parser.Column{
			{Name: "field1", Type: "text"},
			{Name: "field2", Type: "integer"},
		},
	}
	dropChange := &CompositeTypeChange{
		ChangeType:       DropCompositeType,
		CompositeType:    parser.CompositeType{Name: "old_type"},
		OldCompositeType: &oldType,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE TYPE") {
		t.Error("DownSQL for DropCompositeType with OldCompositeType should contain CREATE TYPE")
	}
	if !strings.Contains(downSQL, "field1") || !strings.Contains(downSQL, "field2") {
		t.Error("DownSQL for DropCompositeType should preserve all attributes")
	}

	dropChangeNoOld := &CompositeTypeChange{
		ChangeType:    DropCompositeType,
		CompositeType: parser.CompositeType{Name: "old_type"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropCompositeType without OldCompositeType should indicate IRREVERSIBLE")
	}
}

func TestCompositeTypeChange_IsReversible(t *testing.T) {
	createChange := &CompositeTypeChange{
		ChangeType: CreateCompositeType,
		CompositeType: parser.CompositeType{
			Name:       "test",
			Attributes: []parser.Column{{Name: "field", Type: "text"}},
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateCompositeType should be reversible")
	}

	oldType := parser.CompositeType{Name: "test", Attributes: []parser.Column{{Name: "field", Type: "text"}}}
	dropChangeWithOld := &CompositeTypeChange{
		ChangeType:       DropCompositeType,
		CompositeType:    parser.CompositeType{Name: "test"},
		OldCompositeType: &oldType,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropCompositeType with OldCompositeType should be reversible")
	}

	dropChangeNoOld := &CompositeTypeChange{
		ChangeType:    DropCompositeType,
		CompositeType: parser.CompositeType{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropCompositeType without OldCompositeType should not be reversible")
	}
}
