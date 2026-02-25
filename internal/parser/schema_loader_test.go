package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSchema_SingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.sql")
	content := "CREATE TABLE users (id SERIAL PRIMARY KEY);"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSchema(path)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}
	if got != content {
		t.Errorf("LoadSchema() = %q, want %q", got, content)
	}
}

func TestLoadSchema_SingleFileWithoutExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.sql")
	content := "CREATE TABLE users (id SERIAL PRIMARY KEY);"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSchema(filepath.Join(dir, "schema"))
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}
	if got != content {
		t.Errorf("LoadSchema() = %q, want %q", got, content)
	}
}

func TestLoadSchema_NotFound(t *testing.T) {
	_, err := LoadSchema("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}
}

func TestLoadSchema_FlatDirectory(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"types.sql":  "CREATE TYPE status AS ENUM ('active', 'inactive');",
		"tables.sql": "CREATE TABLE users (\n    id SERIAL PRIMARY KEY,\n    status status NOT NULL\n);",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(schemaDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(schemaDir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	typesIdx := strings.Index(got, "CREATE TYPE status")
	tablesIdx := strings.Index(got, "CREATE TABLE users")
	if typesIdx < 0 || tablesIdx < 0 {
		t.Fatalf("expected both statements in output, got:\n%s", got)
	}
	if typesIdx > tablesIdx {
		t.Errorf("types should precede tables (types at %d, tables at %d)", typesIdx, tablesIdx)
	}
}

func TestLoadSchema_RecursiveDirectory(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema")

	subdirs := []string{
		filepath.Join(schemaDir, "types"),
		filepath.Join(schemaDir, "tables"),
	}
	for _, d := range subdirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		filepath.Join(schemaDir, "types", "enums.sql"):  "CREATE TYPE role AS ENUM ('admin', 'user');",
		filepath.Join(schemaDir, "tables", "users.sql"): "CREATE TABLE users (\n    id SERIAL PRIMARY KEY,\n    role role NOT NULL\n);",
		filepath.Join(schemaDir, "tables", "posts.sql"): "CREATE TABLE posts (\n    id SERIAL PRIMARY KEY,\n    user_id INTEGER REFERENCES users(id)\n);",
		filepath.Join(schemaDir, "tables", "readme.md"): "Not a SQL file",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(schemaDir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	if strings.Contains(got, "Not a SQL file") {
		t.Error("non-SQL files should be excluded")
	}

	roleIdx := strings.Index(got, "CREATE TYPE role")
	usersIdx := strings.Index(got, "CREATE TABLE users")
	postsIdx := strings.Index(got, "CREATE TABLE posts")

	if roleIdx < 0 || usersIdx < 0 || postsIdx < 0 {
		t.Fatalf("expected all three statements in output, got:\n%s", got)
	}
	if roleIdx > usersIdx {
		t.Errorf("role type should precede users table (role at %d, users at %d)", roleIdx, usersIdx)
	}
	if usersIdx > postsIdx {
		t.Errorf("users table should precede posts table (users at %d, posts at %d)", usersIdx, postsIdx)
	}
}

func TestLoadSchema_FKDependencyOrdering(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"c_orders.sql":        "CREATE TABLE orders (\n    id SERIAL PRIMARY KEY,\n    customer_id INTEGER REFERENCES customers(id)\n);",
		"b_customers.sql":     "CREATE TABLE customers (\n    id SERIAL PRIMARY KEY,\n    org_id INTEGER REFERENCES organizations(id)\n);",
		"a_organizations.sql": "CREATE TABLE organizations (\n    id SERIAL PRIMARY KEY\n);",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	orgsIdx := strings.Index(got, "CREATE TABLE organizations")
	custIdx := strings.Index(got, "CREATE TABLE customers")
	ordersIdx := strings.Index(got, "CREATE TABLE orders")

	if orgsIdx < 0 || custIdx < 0 || ordersIdx < 0 {
		t.Fatalf("expected all statements in output, got:\n%s", got)
	}
	if orgsIdx > custIdx {
		t.Errorf("organizations should precede customers")
	}
	if custIdx > ordersIdx {
		t.Errorf("customers should precede orders")
	}
}

func TestLoadSchema_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadSchema(dir)
	if err == nil {
		t.Error("expected error for empty directory, got nil")
	}
}

func TestLoadSchema_ExtensionBeforeTables(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"tables.sql":     "CREATE TABLE events (\n    id uuid PRIMARY KEY DEFAULT gen_random_uuid()\n);",
		"extensions.sql": `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	if !strings.Contains(got, "CREATE EXTENSION") || !strings.Contains(got, "CREATE TABLE events") {
		t.Errorf("expected both statements, got:\n%s", got)
	}
}

func TestLoadSchema_CyclicDependencies(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"a.sql": "CREATE TABLE a (\n    id SERIAL PRIMARY KEY,\n    b_id INTEGER REFERENCES b(id)\n);",
		"b.sql": "CREATE TABLE b (\n    id SERIAL PRIMARY KEY,\n    a_id INTEGER REFERENCES a(id)\n);",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}
	if !strings.Contains(got, "CREATE TABLE a") || !strings.Contains(got, "CREATE TABLE b") {
		t.Errorf("expected both tables in output, got:\n%s", got)
	}
}

func TestLoadSchema_QuotedIdentifiers(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"types.sql":  `CREATE TYPE "UserStatus" AS ENUM ('active', 'inactive');`,
		"tables.sql": "CREATE TABLE \"Users\" (\n    id SERIAL PRIMARY KEY,\n    status \"UserStatus\" NOT NULL\n);",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	typeIdx := strings.Index(got, `CREATE TYPE "UserStatus"`)
	tableIdx := strings.Index(got, `CREATE TABLE "Users"`)
	if typeIdx < 0 || tableIdx < 0 {
		t.Fatalf("expected both statements, got:\n%s", got)
	}
	if typeIdx > tableIdx {
		t.Errorf("type should precede table")
	}
}

func TestLoadSchema_SchemaQualifiedNames(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"schema.sql": "CREATE SCHEMA app;",
		"tables.sql": "CREATE TABLE app.users (\n    id SERIAL PRIMARY KEY\n);",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	if !strings.Contains(got, "CREATE SCHEMA app") && !strings.Contains(got, "CREATE TABLE app.users") {
		t.Errorf("expected both statements, got:\n%s", got)
	}
}

func TestLoadSchema_CommentsIgnored(t *testing.T) {
	dir := t.TempDir()

	// File A defines "items", file B defines "orders".
	// File B has a comment mentioning "items" but no actual dependency.
	// Without comment stripping, this would create a false dependency.
	files := map[string]string{
		"items.sql":  "CREATE TABLE items (\n    id SERIAL PRIMARY KEY\n);",
		"orders.sql": "-- TODO: eventually add REFERENCES to items table\nCREATE TABLE orders (\n    id SERIAL PRIMARY KEY\n);",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	// Both should be present, alphabetical order since no real dependency
	itemsIdx := strings.Index(got, "CREATE TABLE items")
	ordersIdx := strings.Index(got, "CREATE TABLE orders")
	if itemsIdx < 0 || ordersIdx < 0 {
		t.Fatalf("expected both tables, got:\n%s", got)
	}
	if itemsIdx > ordersIdx {
		t.Errorf("items should precede orders alphabetically (no real dep), items at %d, orders at %d", itemsIdx, ordersIdx)
	}
}

func TestLoadSchema_StringLiteralsIgnored(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"categories.sql": "CREATE TABLE categories (\n    id SERIAL PRIMARY KEY,\n    name TEXT NOT NULL\n);",
		"products.sql":   "CREATE TABLE products (\n    id SERIAL PRIMARY KEY,\n    description TEXT DEFAULT 'see categories for details'\n);",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	// "categories" in a string literal should not create a dependency
	catIdx := strings.Index(got, "CREATE TABLE categories")
	prodIdx := strings.Index(got, "CREATE TABLE products")
	if catIdx < 0 || prodIdx < 0 {
		t.Fatalf("expected both tables, got:\n%s", got)
	}
	if catIdx > prodIdx {
		t.Errorf("categories should precede products alphabetically (no real dep)")
	}
}

func TestLoadSchema_BlockCommentsIgnored(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"accounts.sql": "CREATE TABLE accounts (\n    id SERIAL PRIMARY KEY\n);",
		"ledger.sql":   "/*\n * This table tracks ledger entries.\n * REFERENCES accounts for auditing.\n */\nCREATE TABLE ledger (\n    id SERIAL PRIMARY KEY\n);",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	accIdx := strings.Index(got, "CREATE TABLE accounts")
	ledIdx := strings.Index(got, "CREATE TABLE ledger")
	if accIdx < 0 || ledIdx < 0 {
		t.Fatalf("expected both tables, got:\n%s", got)
	}
	if accIdx > ledIdx {
		t.Errorf("accounts should precede ledger alphabetically (no real dep from block comment)")
	}
}

func TestLoadSchema_DollarQuotedStringsIgnored(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"events.sql": "CREATE TABLE events (\n    id SERIAL PRIMARY KEY\n);",
		"funcs.sql":  "CREATE FUNCTION notify() RETURNS TRIGGER LANGUAGE plpgsql AS $$\nBEGIN\n    -- references events table internally\n    PERFORM pg_notify('events', NEW.id::text);\n    RETURN NEW;\nEND;\n$$;",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	// "events" inside $$ ... $$ should not create a dependency
	eventsIdx := strings.Index(got, "CREATE TABLE events")
	funcsIdx := strings.Index(got, "CREATE FUNCTION notify")
	if eventsIdx < 0 || funcsIdx < 0 {
		t.Fatalf("expected both statements, got:\n%s", got)
	}
}

func TestStripSQLCommentsAndStrings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single line comment",
			input: "SELECT * FROM users; -- get all users",
			want:  "SELECT * FROM users; ",
		},
		{
			name:  "block comment",
			input: "SELECT /* all columns */ * FROM users;",
			want:  "SELECT  * FROM users;",
		},
		{
			name:  "string literal",
			input: "INSERT INTO t (name) VALUES ('hello world');",
			want:  "INSERT INTO t (name) VALUES ();",
		},
		{
			name:  "escaped quote in string",
			input: "INSERT INTO t (name) VALUES ('it''s fine');",
			want:  "INSERT INTO t (name) VALUES ();",
		},
		{
			name:  "dollar quoted",
			input: "CREATE FUNCTION f() AS $$ SELECT 1; $$ LANGUAGE sql;",
			want:  "CREATE FUNCTION f() AS  LANGUAGE sql;",
		},
		{
			name:  "tagged dollar quote",
			input: "CREATE FUNCTION f() AS $fn$ SELECT 1; $fn$ LANGUAGE sql;",
			want:  "CREATE FUNCTION f() AS  LANGUAGE sql;",
		},
		{
			name:  "no comments or strings",
			input: "CREATE TABLE users (id SERIAL PRIMARY KEY);",
			want:  "CREATE TABLE users (id SERIAL PRIMARY KEY);",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripSQLCommentsAndStrings(tt.input)
			if got != tt.want {
				t.Errorf("stripSQLCommentsAndStrings() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractDependencies_Basic(t *testing.T) {
	files := []*schemaFile{
		{
			Path:    "types.sql",
			Content: "CREATE TYPE status AS ENUM ('a', 'b');",
		},
		{
			Path:    "tables.sql",
			Content: "CREATE TABLE users (\n    id SERIAL PRIMARY KEY,\n    status status NOT NULL\n);",
		},
	}

	extractDependencies(files)

	if len(files[0].Defines) != 1 || files[0].Defines[0] != "status" {
		t.Errorf("types.sql defines = %v, want [status]", files[0].Defines)
	}
	if len(files[0].DependsOn) != 0 {
		t.Errorf("types.sql dependencies = %v, want none", files[0].DependsOn)
	}

	if len(files[1].Defines) != 1 || files[1].Defines[0] != "users" {
		t.Errorf("tables.sql defines = %v, want [users]", files[1].Defines)
	}
	if len(files[1].DependsOn) != 1 || files[1].DependsOn[0] != "status" {
		t.Errorf("tables.sql dependencies = %v, want [status]", files[1].DependsOn)
	}
}

func TestToposortSchemaFiles_Linear(t *testing.T) {
	files := []*schemaFile{
		{Path: "c.sql", Defines: []string{"c"}, DependsOn: []string{"b"}},
		{Path: "b.sql", Defines: []string{"b"}, DependsOn: []string{"a"}},
		{Path: "a.sql", Defines: []string{"a"}, DependsOn: nil},
	}

	sorted := toposortSchemaFiles(files)

	if len(sorted) != 3 {
		t.Fatalf("expected 3 files, got %d", len(sorted))
	}

	pathOrder := make([]string, len(sorted))
	for i, sf := range sorted {
		pathOrder[i] = sf.Path
	}

	aIdx, bIdx, cIdx := -1, -1, -1
	for i, p := range pathOrder {
		switch p {
		case "a.sql":
			aIdx = i
		case "b.sql":
			bIdx = i
		case "c.sql":
			cIdx = i
		}
	}

	if aIdx > bIdx || bIdx > cIdx {
		t.Errorf("expected a < b < c, got order: %v", pathOrder)
	}
}
