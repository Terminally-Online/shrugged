package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestPgTypeToGo(t *testing.T) {
	tests := []struct {
		pgType   string
		nullable bool
		wantType string
		wantImp  string
	}{
		{"integer", false, "int32", ""},
		{"integer", true, "*int32", ""},
		{"int4", false, "int32", ""},
		{"bigint", false, "int64", ""},
		{"bigint", true, "*int64", ""},
		{"int8", false, "int64", ""},
		{"smallint", false, "int16", ""},
		{"int2", false, "int16", ""},
		{"real", false, "float32", ""},
		{"float4", false, "float32", ""},
		{"double precision", false, "float64", ""},
		{"float8", false, "float64", ""},
		{"boolean", false, "bool", ""},
		{"boolean", true, "*bool", ""},
		{"bool", false, "bool", ""},
		{"text", false, "string", ""},
		{"text", true, "*string", ""},
		{"varchar", false, "string", ""},
		{"character varying", false, "string", ""},
		{"character varying(255)", false, "string", ""},
		{"char", false, "string", ""},
		{"bytea", false, "[]byte", ""},
		{"bytea", true, "[]byte", ""},
		{"uuid", false, "string", ""},
		{"uuid", true, "*string", ""},
		{"json", false, "json.RawMessage", "encoding/json"},
		{"jsonb", false, "json.RawMessage", "encoding/json"},
		{"jsonb", true, "json.RawMessage", "encoding/json"},
		{"timestamp", false, "time.Time", "time"},
		{"timestamp", true, "*time.Time", "time"},
		{"timestamp without time zone", false, "time.Time", "time"},
		{"timestamp with time zone", false, "time.Time", "time"},
		{"timestamptz", false, "time.Time", "time"},
		{"date", false, "time.Time", "time"},
		{"time", false, "time.Time", "time"},
		{"interval", false, "string", ""},
		{"numeric", false, "string", ""},
		{"numeric(10,2)", false, "string", ""},
		{"decimal", false, "string", ""},
		{"money", false, "string", ""},
		{"inet", false, "string", ""},
		{"cidr", false, "string", ""},
		{"macaddr", false, "string", ""},
		{"xml", false, "string", ""},
		{"oid", false, "uint32", ""},
		{"text[]", false, "[]string", ""},
		{"integer[]", false, "[]int32", ""},
		{"bigint[]", true, "[]int64", ""},
		{"user_status", false, "UserStatus", ""},
		{"my_custom_type", true, "*MyCustomType", ""},
	}

	for _, tt := range tests {
		t.Run(tt.pgType, func(t *testing.T) {
			gotType, gotImp := pgTypeToGo(tt.pgType, tt.nullable)
			if gotType != tt.wantType {
				t.Errorf("pgTypeToGo(%q, %v) type = %q, want %q", tt.pgType, tt.nullable, gotType, tt.wantType)
			}
			if gotImp != tt.wantImp {
				t.Errorf("pgTypeToGo(%q, %v) import = %q, want %q", tt.pgType, tt.nullable, gotImp, tt.wantImp)
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
		{"user_id", "UserID"},
		{"user_name", "UserName"},
		{"created_at", "CreatedAt"},
		{"http_url", "HTTPURL"},
		{"api_key", "APIKey"},
		{"json_data", "JSONData"},
		{"xml_content", "XMLContent"},
		{"user_uuid", "UserUUID"},
		{"sql_query", "SQLQuery"},
		{"tcp_port", "TCPPort"},
		{"is_active", "IsActive"},
		{"has_bio", "HasBio"},
		{"some_thing", "SomeThing"},
		{"a", "A"},
		{"", ""},
		{"already_pascal", "AlreadyPascal"},
		{"with-dashes", "WithDashes"},
		{"multiple__underscores", "MultipleUnderscores"},
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

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user", "user"},
		{"UserID", "user_i_d"},
		{"userName", "user_name"},
		{"CreatedAt", "created_at"},
		{"already_snake", "already_snake"},
		{"A", "a"},
		{"", ""},
		{"ABC", "a_b_c"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsCommonInitialism(t *testing.T) {
	trueCases := []string{"ID", "URL", "API", "HTTP", "JSON", "XML", "UUID", "SQL", "TCP", "IP"}
	for _, s := range trueCases {
		if !isCommonInitialism(s) {
			t.Errorf("isCommonInitialism(%q) = false, want true", s)
		}
	}

	falseCases := []string{"id", "User", "Name", "FOO", "BAR"}
	for _, s := range falseCases {
		if isCommonInitialism(s) {
			t.Errorf("isCommonInitialism(%q) = true, want false", s)
		}
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if !fileExists(existingFile) {
		t.Error("fileExists() = false for existing file")
	}

	if fileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("fileExists() = true for non-existent file")
	}
}

func TestGoGenerator_Language(t *testing.T) {
	g := &GoGenerator{}
	if g.Language() != "go" {
		t.Errorf("Language() = %q, want %q", g.Language(), "go")
	}
}

func TestGoGenerator_GenerateTable(t *testing.T) {
	g := &GoGenerator{}
	tmpDir := t.TempDir()

	table := parser.Table{
		Schema: "public",
		Name:   "users",
		Columns: []parser.Column{
			{Name: "id", Type: "bigint", Nullable: false},
			{Name: "email", Type: "text", Nullable: false},
			{Name: "bio", Type: "text", Nullable: true},
			{Name: "created_at", Type: "timestamp with time zone", Nullable: false},
		},
	}

	if err := g.generateTable(table, tmpDir); err != nil {
		t.Fatalf("generateTable() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "users.go"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	checks := []string{
		"package models",
		"type Users struct",
		"ID int64",
		"Email string",
		"Bio *string",
		"CreatedAt time.Time",
		`json:"id"`,
		`json:"email"`,
		`json:"bio,omitempty"`,
		`json:"created_at"`,
		`"time"`,
	}

	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("generated file should contain %q", check)
		}
	}
}

func TestGoGenerator_GenerateEnum(t *testing.T) {
	g := &GoGenerator{}
	tmpDir := t.TempDir()

	enum := parser.Enum{
		Schema: "public",
		Name:   "user_status",
		Values: []string{"active", "inactive", "pending"},
	}

	if err := g.generateEnum(enum, tmpDir); err != nil {
		t.Fatalf("generateEnum() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "user_status.go"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	checks := []string{
		"package models",
		"type UserStatus string",
		"const (",
		"UserStatusActive UserStatus",
		`"active"`,
		"UserStatusInactive UserStatus",
		`"inactive"`,
		"UserStatusPending UserStatus",
		`"pending"`,
	}

	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("generated file should contain %q", check)
		}
	}
}

func TestGoGenerator_GenerateCompositeType(t *testing.T) {
	g := &GoGenerator{}
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

	content, err := os.ReadFile(filepath.Join(tmpDir, "address.go"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	checks := []string{
		"package models",
		"type Address struct",
		"Street string",
		"City string",
		"ZipCode *string",
	}

	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("generated file should contain %q", check)
		}
	}
}

func TestGoGenerator_SkipsMigrationsTable(t *testing.T) {
	g := &GoGenerator{}
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

	if fileExists(filepath.Join(tmpDir, "shrugged_migrations.go")) {
		t.Error("should not generate shrugged_migrations.go")
	}

	if !fileExists(filepath.Join(tmpDir, "users.go")) {
		t.Error("should generate users.go")
	}
}
