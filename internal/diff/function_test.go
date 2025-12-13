package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateFunction(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Functions: []parser.Function{
			{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "SELECT 1"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateFunction && c.ObjectName() == "my_func" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateFunction change for my_func")
	}
}

func TestCompare_DropFunction(t *testing.T) {
	current := &parser.Schema{
		Functions: []parser.Function{
			{Name: "old_func", Args: "", Returns: "void", Language: "sql", Body: "SELECT 1"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropFunction && c.ObjectName() == "old_func" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropFunction change for old_func")
	}
}

func TestCompare_AlterFunction(t *testing.T) {
	current := &parser.Schema{
		Functions: []parser.Function{
			{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "SELECT 1"},
		},
	}
	desired := &parser.Schema{
		Functions: []parser.Function{
			{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "SELECT 2"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterFunction && c.ObjectName() == "my_func" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected AlterFunction change for my_func")
	}
}

func TestFunctionChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *FunctionChange
		want   []string
	}{
		{
			name: "create function",
			change: &FunctionChange{
				ChangeType: CreateFunction,
				Function: parser.Function{
					Name:     "add_numbers",
					Args:     "a integer, b integer",
					Returns:  "integer",
					Language: "sql",
					Body:     "SELECT a + b",
				},
			},
			want: []string{"CREATE OR REPLACE FUNCTION", "add_numbers", "a integer, b integer", "RETURNS", "integer", "LANGUAGE", "sql", "SELECT a + b"},
		},
		{
			name: "create function with definition",
			change: &FunctionChange{
				ChangeType: CreateFunction,
				Function: parser.Function{
					Name:       "complex_func",
					Definition: "CREATE FUNCTION complex_func() RETURNS void AS $$ BEGIN NULL; END; $$ LANGUAGE plpgsql",
				},
			},
			want: []string{"CREATE FUNCTION complex_func"},
		},
		{
			name: "drop function",
			change: &FunctionChange{
				ChangeType: DropFunction,
				Function:   parser.Function{Name: "old_func"},
			},
			want: []string{"DROP FUNCTION", "old_func"},
		},
		{
			name: "drop function with schema",
			change: &FunctionChange{
				ChangeType: DropFunction,
				Function:   parser.Function{Schema: "myschema", Name: "my_func"},
			},
			want: []string{"DROP FUNCTION", "myschema", "my_func"},
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

func TestFunctionChange_DownSQL(t *testing.T) {
	createChange := &FunctionChange{
		ChangeType: CreateFunction,
		Function: parser.Function{
			Name:     "my_func",
			Args:     "",
			Returns:  "integer",
			Language: "sql",
			Body:     "SELECT 1",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP FUNCTION") {
		t.Error("DownSQL for CreateFunction should contain DROP FUNCTION")
	}

	oldFunc := parser.Function{Name: "old_func", Args: "x integer", Returns: "integer", Language: "sql", Body: "SELECT x * 2"}
	dropChange := &FunctionChange{
		ChangeType:  DropFunction,
		Function:    parser.Function{Name: "old_func"},
		OldFunction: &oldFunc,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE OR REPLACE FUNCTION") {
		t.Error("DownSQL for DropFunction with OldFunction should contain CREATE OR REPLACE FUNCTION")
	}
	if !strings.Contains(downSQL, "SELECT x * 2") {
		t.Error("DownSQL for DropFunction should preserve function body")
	}

	dropChangeNoOld := &FunctionChange{
		ChangeType: DropFunction,
		Function:   parser.Function{Name: "old_func"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropFunction without OldFunction should indicate IRREVERSIBLE")
	}

	oldFuncForAlter := parser.Function{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "SELECT 1"}
	alterChange := &FunctionChange{
		ChangeType: AlterFunction,
		Function: parser.Function{
			Name:     "my_func",
			Args:     "",
			Returns:  "integer",
			Language: "sql",
			Body:     "SELECT 2",
		},
		OldFunction: &oldFuncForAlter,
	}
	alterDownSQL := alterChange.DownSQL()
	if !strings.Contains(alterDownSQL, "SELECT 1") {
		t.Error("DownSQL for AlterFunction should restore old body")
	}
}

func TestFunctionChange_IsReversible(t *testing.T) {
	createChange := &FunctionChange{
		ChangeType: CreateFunction,
		Function: parser.Function{
			Name:     "test",
			Args:     "",
			Returns:  "void",
			Language: "sql",
			Body:     "SELECT 1",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateFunction should be reversible")
	}

	oldFunc := parser.Function{Name: "test", Args: "", Returns: "void", Language: "sql", Body: "SELECT 1"}
	dropChangeWithOld := &FunctionChange{
		ChangeType:  DropFunction,
		Function:    parser.Function{Name: "test"},
		OldFunction: &oldFunc,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropFunction with OldFunction should be reversible")
	}

	dropChangeNoOld := &FunctionChange{
		ChangeType: DropFunction,
		Function:   parser.Function{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropFunction without OldFunction should not be reversible")
	}

	alterChangeWithOld := &FunctionChange{
		ChangeType:  AlterFunction,
		Function:    parser.Function{Name: "test", Body: "SELECT 2"},
		OldFunction: &oldFunc,
	}
	if !alterChangeWithOld.IsReversible() {
		t.Error("AlterFunction with OldFunction should be reversible")
	}

	alterChangeNoOld := &FunctionChange{
		ChangeType: AlterFunction,
		Function:   parser.Function{Name: "test", Body: "SELECT 2"},
	}
	if alterChangeNoOld.IsReversible() {
		t.Error("AlterFunction without OldFunction should not be reversible")
	}
}

func TestGenerateCreateFunction(t *testing.T) {
	f := parser.Function{
		Name:     "multiply",
		Args:     "a integer, b integer",
		Returns:  "integer",
		Language: "sql",
		Body:     "SELECT a * b",
	}

	sql := generateCreateFunction(f)

	if !strings.Contains(sql, "CREATE OR REPLACE FUNCTION") {
		t.Error("should contain CREATE OR REPLACE FUNCTION")
	}
	if !strings.Contains(sql, "multiply") {
		t.Error("should contain function name")
	}
	if !strings.Contains(sql, "RETURNS integer") {
		t.Error("should contain RETURNS clause")
	}
	if !strings.Contains(sql, "LANGUAGE sql") {
		t.Error("should contain LANGUAGE clause")
	}
	if !strings.Contains(sql, "$$SELECT a * b$$") {
		t.Error("should contain function body")
	}
}
