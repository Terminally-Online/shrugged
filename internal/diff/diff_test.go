package diff

import (
	"testing"

	"git.ca.plug.to/terminally-online/shrugged/internal/parser"
)

func TestCompare_NoChanges(t *testing.T) {
	schema := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Columns: []parser.Column{{Name: "id", Type: "integer"}}},
		},
	}

	changes := Compare(schema, schema)

	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestCompare_MultipleTypes(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Namespaces: []parser.Namespace{{Name: "myschema"}},
		Extensions: []parser.Extension{{Name: "uuid-ossp"}},
		Enums:      []parser.Enum{{Name: "status", Values: []string{"active"}}},
		Tables:     []parser.Table{{Name: "users", Columns: []parser.Column{{Name: "id", Type: "int"}}}},
		Indexes:    []parser.Index{{Name: "idx_users", Table: "users", Columns: []string{"id"}}},
		Views:      []parser.View{{Name: "v_users", Definition: "SELECT 1"}},
		Functions:  []parser.Function{{Name: "my_func", Returns: "int", Language: "sql", Body: "SELECT 1"}},
	}

	changes := Compare(current, desired)

	expectedTypes := map[ChangeType]bool{
		CreateNamespace: false,
		CreateExtension: false,
		CreateEnum:      false,
		CreateTable:     false,
		CreateIndex:     false,
		CreateView:      false,
		CreateFunction:  false,
	}

	for _, c := range changes {
		if _, ok := expectedTypes[c.Type()]; ok {
			expectedTypes[c.Type()] = true
		}
	}

	for ct, found := range expectedTypes {
		if !found {
			t.Errorf("expected change type %v not found", ct)
		}
	}
}

func TestCompare_OrderPreservation(t *testing.T) {
	current := &parser.Schema{
		Tables: []parser.Table{
			{Name: "to_drop", Columns: []parser.Column{{Name: "id", Type: "int"}}},
		},
	}
	desired := &parser.Schema{
		Namespaces: []parser.Namespace{{Name: "first"}},
		Tables:     []parser.Table{{Name: "second", Columns: []parser.Column{{Name: "id", Type: "int"}}}},
	}

	changes := Compare(current, desired)

	if len(changes) < 2 {
		t.Fatalf("expected at least 2 changes, got %d", len(changes))
	}

	namespaceIdx := -1
	tableIdx := -1
	for i, c := range changes {
		if c.Type() == CreateNamespace {
			namespaceIdx = i
		}
		if c.Type() == CreateTable {
			tableIdx = i
		}
	}

	if namespaceIdx == -1 || tableIdx == -1 {
		t.Fatal("expected both namespace and table changes")
	}

	if namespaceIdx > tableIdx {
		t.Error("namespaces should be created before tables")
	}
}

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"integer", "int4"},
		{"INTEGER", "int4"},
		{"bigint", "int8"},
		{"smallint", "int2"},
		{"boolean", "bool"},
		{"double precision", "float8"},
		{"real", "float4"},
		{"character varying", "varchar"},
		{"text", "text"},
		{"varchar", "varchar"},
		{"timestamp without time zone", "timestamp"},
		{"timestamp with time zone", "timestamptz"},
		{"time without time zone", "time"},
		{"time with time zone", "timetz"},
		{"  INTEGER  ", "int4"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeType(tt.input); got != tt.want {
				t.Errorf("normalizeType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeSQL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trims whitespace",
			input: "  SELECT 1  ",
			want:  "select 1",
		},
		{
			name:  "removes semicolon",
			input: "SELECT 1;",
			want:  "select 1",
		},
		{
			name:  "normalizes newlines",
			input: "SELECT\n  *\n  FROM\n  users",
			want:  "select * from users",
		},
		{
			name:  "collapses multiple spaces",
			input: "SELECT   id,    name   FROM   users",
			want:  "select id, name from users",
		},
		{
			name:  "lowercases",
			input: "SELECT ID FROM USERS",
			want:  "select id from users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSQL(tt.input); got != tt.want {
				t.Errorf("normalizeSQL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "users"},
		{"my_table", "my_table"},
		{"select", `"select"`},
		{"table", `"table"`},
		{"User", `"User"`},
		{"my-table", `"my-table"`},
		{"123abc", `"123abc"`},
		{"", ""},
		{"with_underscore_123", "with_underscore_123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := quoteIdent(tt.input); got != tt.want {
				t.Errorf("quoteIdent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestQuoteIdents(t *testing.T) {
	input := []string{"users", "select", "my_col"}
	want := []string{"users", `"select"`, "my_col"}

	got := quoteIdents(input)

	if len(got) != len(want) {
		t.Fatalf("quoteIdents() returned %d items, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("quoteIdents()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestQuoteLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "'hello'"},
		{"it's", "'it''s'"},
		{"", "''"},
		{"don't stop", "'don''t stop'"},
		{"normal text", "'normal text'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := quoteLiteral(tt.input); got != tt.want {
				t.Errorf("quoteLiteral(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestQualifiedName(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		object string
		want   string
	}{
		{
			name:   "no schema",
			schema: "",
			object: "users",
			want:   "users",
		},
		{
			name:   "public schema",
			schema: "public",
			object: "users",
			want:   "users",
		},
		{
			name:   "custom schema",
			schema: "myschema",
			object: "users",
			want:   "myschema.users",
		},
		{
			name:   "reserved word object",
			schema: "myschema",
			object: "select",
			want:   `myschema."select"`,
		},
		{
			name:   "reserved word schema",
			schema: "table",
			object: "users",
			want:   `"table".users`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := qualifiedName(tt.schema, tt.object); got != tt.want {
				t.Errorf("qualifiedName(%q, %q) = %q, want %q", tt.schema, tt.object, got, tt.want)
			}
		})
	}
}

func TestDropTableFiltersTableGrants(t *testing.T) {
	current := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Schema: "public", Columns: []parser.Column{{Name: "id", Type: "int"}}},
		},
		RoleGrants: []parser.RoleGrant{
			{Privilege: "SELECT", ObjectType: "TABLE", Schema: "public", ObjectName: "users", Grantee: "reader"},
			{Privilege: "USAGE", ObjectType: "TYPE", Schema: "public", ObjectName: "status_enum", Grantee: "reader"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	for _, c := range changes {
		if c.Type() == DropRoleGrant {
			gc := c.(*RoleGrantChange)
			if gc.RoleGrant.ObjectType == "TABLE" && gc.RoleGrant.ObjectName == "users" {
				t.Errorf("REVOKE on TABLE users should be filtered — table is being dropped")
			}
		}
	}

	foundTypeRevoke := false
	for _, c := range changes {
		if c.Type() == DropRoleGrant {
			gc := c.(*RoleGrantChange)
			if gc.RoleGrant.ObjectType == "TYPE" {
				foundTypeRevoke = true
			}
		}
	}
	if !foundTypeRevoke {
		t.Error("REVOKE USAGE ON TYPE should be preserved — types are not cascade-dropped with tables")
	}
}

func TestDropTableOwnedObjectsUseIfExists(t *testing.T) {
	current := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Schema: "public", Columns: []parser.Column{{Name: "id", Type: "int"}}},
		},
		Indexes: []parser.Index{
			{Name: "idx_users_email", Schema: "public", Table: "users", Columns: []string{"email"}},
		},
		Triggers: []parser.Trigger{
			{Name: "trg_users_audit", Schema: "public", Table: "users", Timing: "AFTER", Events: []string{"INSERT"}, Function: "audit_fn", ForEach: "ROW"},
		},
		Rules: []parser.Rule{
			{Name: "rule_users", Schema: "public", Table: "users", Definition: "CREATE RULE rule_users AS ON INSERT TO users DO NOTHING"},
		},
		Policies: []parser.Policy{
			{Name: "pol_users", Schema: "public", Table: "users", Permissive: true, Command: "ALL"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	for _, c := range changes {
		sql := c.SQL()
		switch c.Type() {
		case DropIndex:
			if expected := "DROP INDEX IF EXISTS idx_users_email;"; sql != expected {
				t.Errorf("got %q, want %q", sql, expected)
			}
		case DropTrigger:
			if expected := "DROP TRIGGER IF EXISTS trg_users_audit ON users;"; sql != expected {
				t.Errorf("got %q, want %q", sql, expected)
			}
		case DropRule:
			if expected := "DROP RULE IF EXISTS rule_users ON users;"; sql != expected {
				t.Errorf("got %q, want %q", sql, expected)
			}
		case DropPolicy:
			if expected := "DROP POLICY IF EXISTS pol_users ON users;"; sql != expected {
				t.Errorf("got %q, want %q", sql, expected)
			}
		}
	}

	// Verify down migrations preserve CREATE statements for rollback
	for _, c := range changes {
		downSQL := c.DownSQL()
		switch c.Type() {
		case DropIndex:
			if downSQL == "" || downSQL[0] == '-' {
				t.Errorf("DropIndex DownSQL should produce CREATE INDEX, got %q", downSQL)
			}
		case DropTrigger:
			if downSQL == "" || downSQL[0] == '-' {
				t.Errorf("DropTrigger DownSQL should produce CREATE TRIGGER, got %q", downSQL)
			}
		case DropRule:
			if downSQL == "" || downSQL[0] == '-' {
				t.Errorf("DropRule DownSQL should produce CREATE RULE, got %q", downSQL)
			}
		case DropPolicy:
			if downSQL == "" || downSQL[0] == '-' {
				t.Errorf("DropPolicy DownSQL should produce CREATE POLICY, got %q", downSQL)
			}
		}
	}
}

func TestObjectKey(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		object string
		want   string
	}{
		{
			name:   "no schema defaults to public",
			schema: "",
			object: "users",
			want:   "public.users",
		},
		{
			name:   "with schema",
			schema: "myschema",
			object: "users",
			want:   "myschema.users",
		},
		{
			name:   "public schema",
			schema: "public",
			object: "users",
			want:   "public.users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := objectKey(tt.schema, tt.object); got != tt.want {
				t.Errorf("objectKey(%q, %q) = %q, want %q", tt.schema, tt.object, got, tt.want)
			}
		})
	}
}
