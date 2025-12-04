package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateEnum(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Enums: []parser.Enum{
			{Name: "status", Values: []string{"pending", "active", "done"}},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateEnum && c.ObjectName() == "status" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateEnum change for status")
	}
}

func TestCompare_DropEnum(t *testing.T) {
	current := &parser.Schema{
		Enums: []parser.Enum{
			{Name: "old_enum", Values: []string{"a", "b"}},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropEnum && c.ObjectName() == "old_enum" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropEnum change for old_enum")
	}
}

func TestCompare_AlterEnum_AddValues(t *testing.T) {
	current := &parser.Schema{
		Enums: []parser.Enum{
			{Name: "status", Values: []string{"pending", "active"}},
		},
	}
	desired := &parser.Schema{
		Enums: []parser.Enum{
			{Name: "status", Values: []string{"pending", "active", "done", "cancelled"}},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterEnum && c.ObjectName() == "status" {
			found = true
			enumChange := c.(*EnumChange)
			if len(enumChange.AddValues) != 2 {
				t.Errorf("expected 2 new values, got %d", len(enumChange.AddValues))
			}
			break
		}
	}

	if !found {
		t.Error("expected AlterEnum change for status")
	}
}

func TestEnumChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *EnumChange
		want   []string
	}{
		{
			name: "create enum",
			change: &EnumChange{
				ChangeType: CreateEnum,
				Enum:       parser.Enum{Name: "status", Values: []string{"pending", "active", "done"}},
			},
			want: []string{"CREATE TYPE", "status", "AS ENUM", "'pending'", "'active'", "'done'"},
		},
		{
			name: "create enum with schema",
			change: &EnumChange{
				ChangeType: CreateEnum,
				Enum:       parser.Enum{Schema: "myschema", Name: "priority", Values: []string{"low", "high"}},
			},
			want: []string{"CREATE TYPE", "myschema", "priority", "AS ENUM"},
		},
		{
			name: "drop enum",
			change: &EnumChange{
				ChangeType: DropEnum,
				Enum:       parser.Enum{Name: "old_enum"},
			},
			want: []string{"DROP TYPE", "old_enum"},
		},
		{
			name: "alter enum add values",
			change: &EnumChange{
				ChangeType: AlterEnum,
				Enum:       parser.Enum{Name: "status"},
				AddValues:  []string{"cancelled", "archived"},
			},
			want: []string{"ALTER TYPE", "status", "ADD VALUE", "'cancelled'", "'archived'"},
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

func TestEnumChange_DownSQL(t *testing.T) {
	createChange := &EnumChange{
		ChangeType: CreateEnum,
		Enum:       parser.Enum{Name: "status", Values: []string{"a", "b"}},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP TYPE") {
		t.Error("DownSQL for CreateEnum should contain DROP TYPE")
	}

	oldEnum := parser.Enum{Name: "old_enum", Values: []string{"x", "y", "z"}}
	dropChange := &EnumChange{
		ChangeType: DropEnum,
		Enum:       parser.Enum{Name: "old_enum"},
		OldEnum:    &oldEnum,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE TYPE") {
		t.Error("DownSQL for DropEnum with OldEnum should contain CREATE TYPE")
	}
	if !strings.Contains(downSQL, "'x'") || !strings.Contains(downSQL, "'y'") {
		t.Error("DownSQL for DropEnum should preserve enum values")
	}

	dropChangeNoOld := &EnumChange{
		ChangeType: DropEnum,
		Enum:       parser.Enum{Name: "old_enum"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropEnum without OldEnum should indicate IRREVERSIBLE")
	}

	alterChange := &EnumChange{
		ChangeType: AlterEnum,
		Enum:       parser.Enum{Name: "status"},
		AddValues:  []string{"new_value"},
	}
	alterDownSQL := alterChange.DownSQL()
	if !strings.Contains(alterDownSQL, "IRREVERSIBLE") {
		t.Error("DownSQL for AlterEnum should indicate IRREVERSIBLE (can't remove enum values)")
	}
}

func TestEnumChange_IsReversible(t *testing.T) {
	createChange := &EnumChange{
		ChangeType: CreateEnum,
		Enum:       parser.Enum{Name: "test", Values: []string{"a"}},
	}
	if !createChange.IsReversible() {
		t.Error("CreateEnum should be reversible")
	}

	oldEnum := parser.Enum{Name: "test", Values: []string{"a"}}
	dropChangeWithOld := &EnumChange{
		ChangeType: DropEnum,
		Enum:       parser.Enum{Name: "test"},
		OldEnum:    &oldEnum,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropEnum with OldEnum should be reversible")
	}

	dropChangeNoOld := &EnumChange{
		ChangeType: DropEnum,
		Enum:       parser.Enum{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropEnum without OldEnum should not be reversible")
	}

	alterChange := &EnumChange{
		ChangeType: AlterEnum,
		Enum:       parser.Enum{Name: "test"},
		AddValues:  []string{"new"},
	}
	if alterChange.IsReversible() {
		t.Error("AlterEnum (adding values) should not be reversible")
	}
}

func TestQuoteLiterals(t *testing.T) {
	result := quoteLiterals([]string{"a", "b", "c"})
	if result != "'a', 'b', 'c'" {
		t.Errorf("quoteLiterals = %q, want %q", result, "'a', 'b', 'c'")
	}

	resultWithQuote := quoteLiterals([]string{"it's", "test"})
	if !strings.Contains(resultWithQuote, "it''s") {
		t.Error("quoteLiterals should escape single quotes")
	}
}
