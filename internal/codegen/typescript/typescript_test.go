package typescript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/codegen"
	"github.com/terminally-online/shrugged/internal/parser"
)

func TestTypeScriptGenerator_Language(t *testing.T) {
	g := &TypeScriptGenerator{}
	if g.Language() != "typescript" {
		t.Errorf("Language() = %q, want %q", g.Language(), "typescript")
	}
}

func TestPgTypeToTS(t *testing.T) {
	tests := []struct {
		pgType   string
		nullable bool
		want     string
	}{
		{"integer", false, "number"},
		{"integer", true, "number | null"},
		{"int4", false, "number"},
		{"bigint", false, "bigint"},
		{"bigint", true, "bigint | null"},
		{"int8", false, "bigint"},
		{"smallint", false, "number"},
		{"int2", false, "number"},
		{"real", false, "number"},
		{"float4", false, "number"},
		{"double precision", false, "number"},
		{"float8", false, "number"},
		{"boolean", false, "boolean"},
		{"boolean", true, "boolean | null"},
		{"bool", false, "boolean"},
		{"text", false, "string"},
		{"text", true, "string | null"},
		{"varchar", false, "string"},
		{"character varying", false, "string"},
		{"character varying(255)", false, "string"},
		{"char", false, "string"},
		{"bytea", false, "string"},
		{"uuid", false, "string"},
		{"uuid", true, "string | null"},
		{"json", false, "Record<string, unknown>"},
		{"jsonb", false, "Record<string, unknown>"},
		{"jsonb", true, "Record<string, unknown> | null"},
		{"timestamp", false, "Date"},
		{"timestamp", true, "Date | null"},
		{"timestamp without time zone", false, "Date"},
		{"timestamp with time zone", false, "Date"},
		{"timestamptz", false, "Date"},
		{"date", false, "Date"},
		{"time", false, "Date"},
		{"interval", false, "string"},
		{"numeric", false, "string"},
		{"numeric(10,2)", false, "string"},
		{"decimal", false, "string"},
		{"money", false, "string"},
		{"inet", false, "string"},
		{"cidr", false, "string"},
		{"macaddr", false, "string"},
		{"xml", false, "string"},
		{"oid", false, "number"},
		{"text[]", false, "string[]"},
		{"integer[]", false, "number[]"},
		{"bigint[]", false, "bigint[]"},
		{"bigint[]", true, "bigint[] | null"},
	}

	for _, tt := range tests {
		t.Run(tt.pgType, func(t *testing.T) {
			got := pgTypeToTS(tt.pgType, tt.nullable, nil)
			if got != tt.want {
				t.Errorf("pgTypeToTS(%q, %v) = %q, want %q", tt.pgType, tt.nullable, got, tt.want)
			}
		})
	}
}

func TestPgTypeToTSSchemaAware(t *testing.T) {
	schema := &parser.Schema{
		Enums: []parser.Enum{
			{Name: "user_status", Values: []string{"active", "inactive"}},
		},
		CompositeTypes: []parser.CompositeType{
			{Name: "address", Attributes: []parser.Column{{Name: "street", Type: "text"}}},
		},
	}

	tests := []struct {
		pgType   string
		nullable bool
		want     string
	}{
		{"user_status", false, "UserStatus"},
		{"user_status", true, "UserStatus | null"},
		{"address", false, "Address"},
		{"address", true, "Address | null"},
		{"unknown_type", false, "UnknownType"},
	}

	for _, tt := range tests {
		t.Run(tt.pgType, func(t *testing.T) {
			got := pgTypeToTS(tt.pgType, tt.nullable, schema)
			if got != tt.want {
				t.Errorf("pgTypeToTS(%q, %v, schema) = %q, want %q", tt.pgType, tt.nullable, got, tt.want)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user", "user"},
		{"user_id", "userId"},
		{"user_name", "userName"},
		{"created_at", "createdAt"},
		{"is_active", "isActive"},
		{"id", "id"},
		{"", ""},
		{"with-dashes", "withDashes"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user", "User"},
		{"user_id", "UserId"},
		{"user_name", "UserName"},
		{"created_at", "CreatedAt"},
		{"a", "A"},
		{"", ""},
		{"with-dashes", "WithDashes"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user_status", "user-status"},
		{"user", "user"},
		{"created_at", "created-at"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toKebabCase(tt.input)
			if got != tt.want {
				t.Errorf("toKebabCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTypeScriptGenerator_GenerateTable(t *testing.T) {
	g := &TypeScriptGenerator{}
	tmpDir := t.TempDir()

	table := parser.Table{
		Schema: "public",
		Name:   "users",
		Columns: []parser.Column{
			{Name: "id", Type: "bigint", Nullable: false},
			{Name: "email", Type: "text", Nullable: false},
			{Name: "bio", Type: "text", Nullable: true},
			{Name: "created_at", Type: "timestamp with time zone", Nullable: false},
			{Name: "metadata", Type: "jsonb", Nullable: true},
		},
	}

	if err := g.generateTable(table, nil, tmpDir); err != nil {
		t.Fatalf("generateTable() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "users.ts"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	checks := []string{
		"// Code generated by shrugged. DO NOT EDIT.",
		"export interface Users",
		"id: bigint;",
		"email: string;",
		"bio: string | null;",
		"createdAt: Date;",
		"metadata: Record<string, unknown> | null;",
	}

	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("generated file should contain %q\ngot:\n%s", check, content)
		}
	}
}

func TestTypeScriptGenerator_GenerateEnum(t *testing.T) {
	g := &TypeScriptGenerator{}
	tmpDir := t.TempDir()

	enum := parser.Enum{
		Schema: "public",
		Name:   "user_status",
		Values: []string{"active", "inactive", "pending"},
	}

	if err := g.generateEnum(enum, tmpDir); err != nil {
		t.Fatalf("generateEnum() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "user-status.ts"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	checks := []string{
		"// Code generated by shrugged. DO NOT EDIT.",
		"export type UserStatus =",
		`| "active"`,
		`| "inactive"`,
		`| "pending"`,
	}

	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("generated file should contain %q\ngot:\n%s", check, content)
		}
	}
}

func TestTypeScriptGenerator_GenerateCompositeType(t *testing.T) {
	g := &TypeScriptGenerator{}
	tmpDir := t.TempDir()

	ct := parser.CompositeType{
		Schema: "public",
		Name:   "address",
		Attributes: []parser.Column{
			{Name: "street", Type: "text", Nullable: false},
			{Name: "city", Type: "text", Nullable: false},
			{Name: "zip_code", Type: "text", Nullable: true},
		},
	}

	if err := g.generateCompositeType(ct, tmpDir); err != nil {
		t.Fatalf("generateCompositeType() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "address.ts"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	checks := []string{
		"// Code generated by shrugged. DO NOT EDIT.",
		"export interface Address",
		"street: string;",
		"city: string;",
		"zipCode: string | null;",
	}

	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("generated file should contain %q\ngot:\n%s", check, content)
		}
	}
}

func TestTypeScriptGenerator_SkipsMigrationsTable(t *testing.T) {
	g := &TypeScriptGenerator{}
	tmpDir := t.TempDir()

	schema := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Columns: []parser.Column{{Name: "id", Type: "bigint"}}},
			{Name: "shrugged_migrations", Columns: []parser.Column{{Name: "id", Type: "bigint"}}},
		},
	}

	if err := g.Generate(schema, tmpDir); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "shrugged-migrations.ts")); err == nil {
		t.Error("should not generate shrugged-migrations.ts")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "users.ts")); err != nil {
		t.Error("should generate users.ts")
	}
}

func TestTypeScriptGenerator_GenerateQueries_Unsupported(t *testing.T) {
	g := &TypeScriptGenerator{}
	_, err := g.GenerateQueries(codegen.QueryGenOptions{})
	if err == nil {
		t.Error("GenerateQueries() expected error, got nil")
	}
}

func TestTypeScriptGenerator_RegisteredInCodegen(t *testing.T) {
	g, err := codegen.Get("typescript")
	if err != nil {
		t.Fatalf("codegen.Get(\"typescript\") error = %v", err)
	}
	if g.Language() != "typescript" {
		t.Errorf("Language() = %q, want %q", g.Language(), "typescript")
	}
}
