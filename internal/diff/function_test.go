package diff

import (
	"strings"
	"testing"

	"git.ca.plug.to/terminally-online/shrugged/internal/parser"
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
func TestCompare_FunctionBodyWhitespaceOnly(t *testing.T) {
	current := &parser.Schema{
		Functions: []parser.Function{
			{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "SELECT 1"},
		},
	}
	desired := &parser.Schema{
		Functions: []parser.Function{
			{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "  SELECT\n\t1\n"},
		},
	}

	changes := Compare(current, desired)

	for _, c := range changes {
		if c.Type() == AlterFunction && c.ObjectName() == "my_func" {
			t.Error("whitespace-only body difference should not produce AlterFunction")
		}
	}
}

func TestCompare_FunctionBodyCommentOnly(t *testing.T) {
	current := &parser.Schema{
		Functions: []parser.Function{
			{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "SELECT 1"},
		},
	}
	desired := &parser.Schema{
		Functions: []parser.Function{
			{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "-- compute one\nSELECT 1 /* trailing */"},
		},
	}

	changes := Compare(current, desired)

	for _, c := range changes {
		if c.Type() == AlterFunction && c.ObjectName() == "my_func" {
			t.Error("comment-only body difference should not produce AlterFunction")
		}
	}
}

func TestCompare_FunctionBodySemanticChange(t *testing.T) {
	current := &parser.Schema{
		Functions: []parser.Function{
			{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "SELECT 1"},
		},
	}
	desired := &parser.Schema{
		Functions: []parser.Function{
			{Name: "my_func", Args: "", Returns: "integer", Language: "sql", Body: "-- now returns two\nSELECT  2"},
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
		t.Error("genuine body change should still produce AlterFunction")
	}
}

func TestNormalizeFunctionBody(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "trim and collapse whitespace",
			in:   "  SELECT\n\t1\n",
			want: "SELECT 1",
		},
		{
			name: "strip line comment",
			in:   "SELECT 1 -- trailing comment\n",
			want: "SELECT 1",
		},
		{
			name: "strip block comment",
			in:   "SELECT /* inline */ 1",
			want: "SELECT 1",
		},
		{
			name: "preserve -- inside single-quoted string",
			in:   "SELECT 'foo -- bar'",
			want: "SELECT 'foo -- bar'",
		},
		{
			name: "preserve doubled-quote escape inside string",
			in:   "SELECT 'it''s -- fine'",
			want: "SELECT 'it''s -- fine'",
		},
		{
			name: "preserve dollar-quoted body with comment-like text",
			in:   "BEGIN RETURN $$line one -- not a comment\nline two$$; END",
			want: "BEGIN RETURN $$line one -- not a comment line two$$; END",
		},
		{
			name: "preserve tagged dollar quote",
			in:   "SELECT $tag$ -- still text $tag$",
			want: "SELECT $tag$ -- still text $tag$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeFunctionBody(tt.in)
			if got != tt.want {
				t.Errorf("normalizeFunctionBody(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
func TestCompare_FunctionOverloads_DistinctSignatures(t *testing.T) {
	current := &parser.Schema{
		Functions: []parser.Function{
			{Name: "foo", Args: "x integer", Returns: "integer", Language: "sql", Body: "SELECT x"},
			{Name: "foo", Args: "x integer, y text", Returns: "integer", Language: "sql", Body: "SELECT x"},
		},
	}
	desired := &parser.Schema{
		Functions: []parser.Function{
			{Name: "foo", Args: "x integer", Returns: "integer", Language: "sql", Body: "SELECT x"},
			{Name: "foo", Args: "x integer, y text", Returns: "integer", Language: "sql", Body: "SELECT x"},
		},
	}

	changes := Compare(current, desired)

	for _, c := range changes {
		if c.ObjectName() == "foo" {
			t.Errorf("expected no changes for matched overloads, got %v", c.Type())
		}
	}
}

func TestCompare_FunctionOverloads_AddSignature(t *testing.T) {
	current := &parser.Schema{
		Functions: []parser.Function{
			{Name: "foo", Args: "x integer", Returns: "integer", Language: "sql", Body: "SELECT x"},
		},
	}
	desired := &parser.Schema{
		Functions: []parser.Function{
			{Name: "foo", Args: "x integer", Returns: "integer", Language: "sql", Body: "SELECT x"},
			{Name: "foo", Args: "x integer, y text", Returns: "integer", Language: "sql", Body: "SELECT x"},
		},
	}

	changes := Compare(current, desired)

	var creates, alters, drops int
	for _, c := range changes {
		if c.ObjectName() != "foo" {
			continue
		}
		switch c.Type() {
		case CreateFunction:
			creates++
		case AlterFunction:
			alters++
		case DropFunction:
			drops++
		}
	}

	if creates != 1 {
		t.Errorf("expected 1 CreateFunction for new overload, got %d", creates)
	}
	if alters != 0 {
		t.Errorf("expected 0 AlterFunction (existing overload unchanged), got %d", alters)
	}
	if drops != 0 {
		t.Errorf("expected 0 DropFunction (existing overload unchanged), got %d", drops)
	}
}

func TestCompare_FunctionOverloads_SignatureChange(t *testing.T) {
	current := &parser.Schema{
		Functions: []parser.Function{
			{Name: "foo", Args: "x integer", Returns: "integer", Language: "sql", Body: "SELECT x"},
		},
	}
	desired := &parser.Schema{
		Functions: []parser.Function{
			{Name: "foo", Args: "x integer, y text", Returns: "integer", Language: "sql", Body: "SELECT x"},
		},
	}

	changes := Compare(current, desired)

	var dropSQL, createSQL string
	for _, c := range changes {
		if c.ObjectName() != "foo" {
			continue
		}
		switch c.Type() {
		case DropFunction:
			dropSQL = c.(*FunctionChange).SQL()
		case CreateFunction:
			createSQL = c.(*FunctionChange).SQL()
		case AlterFunction:
			t.Errorf("expected DROP+CREATE for signature change, got AlterFunction")
		}
	}

	if dropSQL == "" {
		t.Fatal("expected DropFunction change for old signature")
	}
	if createSQL == "" {
		t.Fatal("expected CreateFunction change for new signature")
	}
	if !strings.Contains(dropSQL, "DROP FUNCTION") || !strings.Contains(dropSQL, "(integer)") {
		t.Errorf("DropFunction SQL should drop the old signature exactly: got %q", dropSQL)
	}
	if !strings.Contains(createSQL, "CREATE OR REPLACE FUNCTION") || !strings.Contains(createSQL, "x integer, y text") {
		t.Errorf("CreateFunction SQL should create the new signature: got %q", createSQL)
	}
}

func TestFunctionChange_DropIncludesSignature(t *testing.T) {
	change := &FunctionChange{
		ChangeType: DropFunction,
		Function: parser.Function{
			Schema: "public",
			Name:   "foo",
			Args:   "p_x integer, p_y text DEFAULT ''",
		},
	}

	sql := change.SQL()
	want := "DROP FUNCTION foo(integer, text);"
	if sql != want {
		t.Errorf("DROP FUNCTION SQL = %q, want %q", sql, want)
	}
}

func TestFunctionChange_DropQualifiedSchemaSignature(t *testing.T) {
	change := &FunctionChange{
		ChangeType: DropFunction,
		Function: parser.Function{
			Schema: "myschema",
			Name:   "foo",
			Args:   "VARIADIC arr integer[]",
		},
	}

	sql := change.SQL()
	want := `DROP FUNCTION myschema.foo(VARIADIC integer[]);`
	if sql != want {
		t.Errorf("DROP FUNCTION SQL = %q, want %q", sql, want)
	}
}

func TestNormalizeFunctionSignature(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{name: "empty", args: "", want: ""},
		{name: "single typed", args: "x integer", want: "integer"},
		{name: "type-only", args: "integer", want: "integer"},
		{name: "two typed", args: "x integer, y text", want: "integer, text"},
		{name: "default value", args: "x integer DEFAULT 0", want: "integer"},
		{name: "equals default", args: "x integer = 0", want: "integer"},
		{name: "default literal text", args: "p_y text DEFAULT ''", want: "text"},
		{name: "parenthesized type", args: "amount numeric(10, 2)", want: "numeric(10, 2)"},
		{name: "in mode stripped", args: "IN x integer", want: "IN integer"},
		{name: "out mode stripped", args: "OUT x integer", want: "OUT integer"},
		{name: "variadic kept", args: "VARIADIC arr integer[]", want: "VARIADIC integer[]"},
		{name: "mixed", args: "p_x int, p_y text DEFAULT 'foo', p_z numeric(10,2)",
			want: "int, text, numeric(10,2)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeFunctionSignature(tt.args)
			if got != tt.want {
				t.Errorf("normalizeFunctionSignature(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
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
