package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseQueryContent_SingleQuery(t *testing.T) {
	content := `-- name: GetUserByID :row
SELECT id, email, name FROM users WHERE id = @user_id;`

	queries, err := parseQueryContent(content, "test.sql")
	if err != nil {
		t.Fatalf("parseQueryContent() error = %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if q.Name != "GetUserByID" {
		t.Errorf("query name = %q, want %q", q.Name, "GetUserByID")
	}
	if q.ResultType != QueryResultRow {
		t.Errorf("result type = %q, want %q", q.ResultType, QueryResultRow)
	}
	if len(q.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(q.Parameters))
	}
	if q.Parameters[0].Name != "user_id" {
		t.Errorf("parameter name = %q, want %q", q.Parameters[0].Name, "user_id")
	}
	if q.Parameters[0].Position != 1 {
		t.Errorf("parameter position = %d, want %d", q.Parameters[0].Position, 1)
	}
	if q.PreparedSQL != "SELECT id, email, name FROM users WHERE id = $1;" {
		t.Errorf("prepared SQL = %q, want @user_id replaced with $1", q.PreparedSQL)
	}
}

func TestParseQueryContent_MultipleQueries(t *testing.T) {
	content := `-- name: GetUserByID :row
SELECT * FROM users WHERE id = @user_id;

-- name: ListUsers :rows
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateUser :exec
INSERT INTO users (email, name) VALUES (@email, @name);

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = @user_id;`

	queries, err := parseQueryContent(content, "test.sql")
	if err != nil {
		t.Fatalf("parseQueryContent() error = %v", err)
	}

	if len(queries) != 4 {
		t.Fatalf("expected 4 queries, got %d", len(queries))
	}

	expected := []struct {
		name       string
		resultType QueryResultType
		paramCount int
	}{
		{"GetUserByID", QueryResultRow, 1},
		{"ListUsers", QueryResultRows, 0},
		{"CreateUser", QueryResultExec, 2},
		{"DeleteUser", QueryResultExecRows, 1},
	}

	for i, exp := range expected {
		if queries[i].Name != exp.name {
			t.Errorf("query %d name = %q, want %q", i, queries[i].Name, exp.name)
		}
		if queries[i].ResultType != exp.resultType {
			t.Errorf("query %d result type = %q, want %q", i, queries[i].ResultType, exp.resultType)
		}
		if len(queries[i].Parameters) != exp.paramCount {
			t.Errorf("query %d param count = %d, want %d", i, len(queries[i].Parameters), exp.paramCount)
		}
	}
}

func TestParseQueryContent_ReusedParameter(t *testing.T) {
	content := `-- name: SearchUsers :rows
SELECT * FROM users WHERE name LIKE @search OR email LIKE @search;`

	queries, err := parseQueryContent(content, "test.sql")
	if err != nil {
		t.Fatalf("parseQueryContent() error = %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if len(q.Parameters) != 1 {
		t.Errorf("expected 1 unique parameter, got %d", len(q.Parameters))
	}
	if q.Parameters[0].Name != "search" {
		t.Errorf("parameter name = %q, want %q", q.Parameters[0].Name, "search")
	}
	if q.PreparedSQL != "SELECT * FROM users WHERE name LIKE $1 OR email LIKE $1;" {
		t.Errorf("prepared SQL should reuse $1 for @search, got %q", q.PreparedSQL)
	}
}

func TestParseQueryContent_NestAnnotation(t *testing.T) {
	content := `-- name: GetUserWithPosts :row
-- nest: User(u.*), Posts(p.*)
SELECT u.id, u.name, p.id as post_id, p.title
FROM users u
LEFT JOIN posts p ON p.user_id = u.id
WHERE u.id = @user_id;`

	queries, err := parseQueryContent(content, "test.sql")
	if err != nil {
		t.Fatalf("parseQueryContent() error = %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if len(q.NestMappings) != 2 {
		t.Fatalf("expected 2 nest mappings, got %d", len(q.NestMappings))
	}

	if q.NestMappings[0].StructName != "User" {
		t.Errorf("first nest struct = %q, want %q", q.NestMappings[0].StructName, "User")
	}
	if q.NestMappings[0].Prefix != "u" {
		t.Errorf("first nest prefix = %q, want %q", q.NestMappings[0].Prefix, "u")
	}

	if q.NestMappings[1].StructName != "Posts" {
		t.Errorf("second nest struct = %q, want %q", q.NestMappings[1].StructName, "Posts")
	}
	if q.NestMappings[1].Prefix != "p" {
		t.Errorf("second nest prefix = %q, want %q", q.NestMappings[1].Prefix, "p")
	}
}

func TestParseQueryContent_NestAnnotationWithColumns(t *testing.T) {
	content := `-- name: GetUserSummary :row
-- nest: User(id, name, email)
SELECT id, name, email, COUNT(posts.id) as post_count
FROM users
LEFT JOIN posts ON posts.user_id = users.id
WHERE users.id = @user_id
GROUP BY users.id;`

	queries, err := parseQueryContent(content, "test.sql")
	if err != nil {
		t.Fatalf("parseQueryContent() error = %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if len(q.NestMappings) != 1 {
		t.Fatalf("expected 1 nest mapping, got %d", len(q.NestMappings))
	}

	if q.NestMappings[0].StructName != "User" {
		t.Errorf("nest struct = %q, want %q", q.NestMappings[0].StructName, "User")
	}
	if len(q.NestMappings[0].Columns) != 3 {
		t.Errorf("nest columns count = %d, want 3", len(q.NestMappings[0].Columns))
	}
}

func TestParseQueryContent_MultilineQuery(t *testing.T) {
	content := `-- name: ComplexQuery :rows
SELECT
    u.id,
    u.email,
    u.name,
    (SELECT json_agg(p.*) FROM posts p WHERE p.user_id = u.id) as posts
FROM users u
WHERE u.status = @status
    AND u.created_at > @created_after
ORDER BY u.created_at DESC
LIMIT @limit;`

	queries, err := parseQueryContent(content, "test.sql")
	if err != nil {
		t.Fatalf("parseQueryContent() error = %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if len(q.Parameters) != 3 {
		t.Errorf("expected 3 parameters, got %d", len(q.Parameters))
	}

	paramNames := make(map[string]bool)
	for _, p := range q.Parameters {
		paramNames[p.Name] = true
	}

	for _, expected := range []string{"status", "created_after", "limit"} {
		if !paramNames[expected] {
			t.Errorf("missing parameter %q", expected)
		}
	}
}

func TestParseQueryContent_IgnoresComments(t *testing.T) {
	content := `-- name: GetUser :row
-- This is a comment that should be ignored
-- Another comment
SELECT * FROM users WHERE id = @id;`

	queries, err := parseQueryContent(content, "test.sql")
	if err != nil {
		t.Fatalf("parseQueryContent() error = %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if q.SQL != "SELECT * FROM users WHERE id = @id;" {
		t.Errorf("SQL should not contain comments, got %q", q.SQL)
	}
}

func TestExtractParameters(t *testing.T) {
	tests := []struct {
		input        string
		wantSQL      string
		wantParamLen int
	}{
		{
			"SELECT * FROM users WHERE id = @id",
			"SELECT * FROM users WHERE id = $1",
			1,
		},
		{
			"INSERT INTO users (email, name) VALUES (@email, @name)",
			"INSERT INTO users (email, name) VALUES ($1, $2)",
			2,
		},
		{
			"SELECT * FROM users WHERE name = @name OR email = @name",
			"SELECT * FROM users WHERE name = $1 OR email = $1",
			1,
		},
		{
			"SELECT * FROM users",
			"SELECT * FROM users",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotSQL, gotParams := extractParameters(tt.input)
			if gotSQL != tt.wantSQL {
				t.Errorf("extractParameters() SQL = %q, want %q", gotSQL, tt.wantSQL)
			}
			if len(gotParams) != tt.wantParamLen {
				t.Errorf("extractParameters() param count = %d, want %d", len(gotParams), tt.wantParamLen)
			}
		})
	}
}

func TestDetectJSONAggregation(t *testing.T) {
	tests := []struct {
		sql  string
		want bool
	}{
		{"SELECT json_agg(p.*) FROM posts p", true},
		{"SELECT jsonb_agg(p.*) FROM posts p", true},
		{"SELECT JSON_AGG(p.*) FROM posts p", true},
		{"SELECT * FROM users", false},
		{"SELECT id, (SELECT json_agg(p) FROM posts p) as posts FROM users", true},
	}

	for _, tt := range tests {
		t.Run(tt.sql, func(t *testing.T) {
			got := DetectJSONAggregation(tt.sql)
			if got != tt.want {
				t.Errorf("DetectJSONAggregation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseQueryFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "queries.sql")

	content := `-- name: GetUser :row
SELECT * FROM users WHERE id = @id;

-- name: ListUsers :rows
SELECT * FROM users;`

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	qf, err := ParseQueryFile(filePath)
	if err != nil {
		t.Fatalf("ParseQueryFile() error = %v", err)
	}

	if qf.Path != filePath {
		t.Errorf("path = %q, want %q", qf.Path, filePath)
	}
	if len(qf.Queries) != 2 {
		t.Errorf("query count = %d, want 2", len(qf.Queries))
	}
}

func TestParseQueryDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	files := []struct {
		name    string
		content string
	}{
		{"users.sql", "-- name: GetUser :row\nSELECT * FROM users WHERE id = @id;"},
		{"posts.sql", "-- name: GetPost :row\nSELECT * FROM posts WHERE id = @id;"},
		{"readme.md", "This is not a SQL file"},
	}

	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f.name), []byte(f.content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", f.name, err)
		}
	}

	queryFiles, err := ParseQueryDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ParseQueryDirectory() error = %v", err)
	}

	if len(queryFiles) != 2 {
		t.Errorf("query file count = %d, want 2 (should skip readme.md)", len(queryFiles))
	}
}

func TestParseQueries_AutoDetect(t *testing.T) {
	tmpDir := t.TempDir()

	singleFile := filepath.Join(tmpDir, "queries.sql")
	if err := os.WriteFile(singleFile, []byte("-- name: Test :row\nSELECT 1;"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := ParseQueries(singleFile)
	if err != nil {
		t.Fatalf("ParseQueries(file) error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}

	queryDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queryDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queryDir, "test.sql"), []byte("-- name: Test2 :row\nSELECT 2;"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err = ParseQueries(queryDir)
	if err != nil {
		t.Fatalf("ParseQueries(dir) error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file from directory, got %d", len(files))
	}
}
