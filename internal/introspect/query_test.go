package introspect

import (
	"context"
	"testing"
	"time"

	"github.com/terminally-online/shrugged/internal/docker"
	"github.com/terminally-online/shrugged/internal/parser"
)

func TestIntrospectQueries_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	setupSQL := `
		CREATE TABLE users (
			id BIGSERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name TEXT NOT NULL,
			bio TEXT,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE TABLE posts (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id),
			title TEXT NOT NULL,
			content TEXT,
			published BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
	`
	if err := docker.ExecuteSQL(ctx, container, setupSQL); err != nil {
		t.Fatalf("failed to setup schema: %v", err)
	}

	schema, err := Database(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to introspect database: %v", err)
	}

	queries := []parser.Query{
		{
			Name:        "GetUserByID",
			SQL:         "SELECT id, email, name, bio, created_at FROM users WHERE id = @user_id",
			PreparedSQL: "SELECT id, email, name, bio, created_at FROM users WHERE id = $1",
			ResultType:  parser.QueryResultRow,
			Parameters: []parser.QueryParameter{
				{Name: "user_id", Position: 1},
			},
		},
		{
			Name:        "ListUsers",
			SQL:         "SELECT id, email, name FROM users ORDER BY created_at DESC",
			PreparedSQL: "SELECT id, email, name FROM users ORDER BY created_at DESC",
			ResultType:  parser.QueryResultRows,
			Parameters:  []parser.QueryParameter{},
		},
		{
			Name:        "CreateUser",
			SQL:         "INSERT INTO users (email, name) VALUES (@email, @name)",
			PreparedSQL: "INSERT INTO users (email, name) VALUES ($1, $2)",
			ResultType:  parser.QueryResultExec,
			Parameters: []parser.QueryParameter{
				{Name: "email", Position: 1},
				{Name: "name", Position: 2},
			},
		},
		{
			Name:        "DeleteUser",
			SQL:         "DELETE FROM users WHERE id = @user_id",
			PreparedSQL: "DELETE FROM users WHERE id = $1",
			ResultType:  parser.QueryResultExecRows,
			Parameters: []parser.QueryParameter{
				{Name: "user_id", Position: 1},
			},
		},
	}

	result, err := Queries(ctx, dbURL, queries, schema)
	if err != nil {
		t.Fatalf("Queries() error = %v", err)
	}

	if len(result) != len(queries) {
		t.Fatalf("expected %d queries, got %d", len(queries), len(result))
	}

	getUserQuery := result[0]
	if getUserQuery.Name != "GetUserByID" {
		t.Errorf("expected GetUserByID, got %s", getUserQuery.Name)
	}
	if len(getUserQuery.Columns) != 5 {
		t.Errorf("expected 5 columns, got %d", len(getUserQuery.Columns))
	}
	if len(getUserQuery.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(getUserQuery.Parameters))
	}
	if getUserQuery.Parameters[0].GoType == "" {
		t.Error("parameter GoType should be set")
	}

	for _, col := range getUserQuery.Columns {
		if col.GoType == "" {
			t.Errorf("column %s GoType should be set", col.Name)
		}
	}

	listUsersQuery := result[1]
	if len(listUsersQuery.Columns) != 3 {
		t.Errorf("expected 3 columns for ListUsers, got %d", len(listUsersQuery.Columns))
	}

	createUserQuery := result[2]
	if len(createUserQuery.Columns) != 0 {
		t.Errorf("expected 0 columns for exec query, got %d", len(createUserQuery.Columns))
	}
	if len(createUserQuery.Parameters) != 2 {
		t.Errorf("expected 2 parameters for CreateUser, got %d", len(createUserQuery.Parameters))
	}
}

func TestIntrospectQueries_JSONAggregation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	setupSQL := `
		CREATE TABLE users (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE posts (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id),
			title TEXT NOT NULL
		);
	`
	if err := docker.ExecuteSQL(ctx, container, setupSQL); err != nil {
		t.Fatalf("failed to setup schema: %v", err)
	}

	schema, err := Database(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to introspect database: %v", err)
	}

	queries := []parser.Query{
		{
			Name: "GetUserWithPosts",
			SQL: `SELECT u.id, u.name,
				(SELECT json_agg(p.*) FROM posts p WHERE p.user_id = u.id) as posts
				FROM users u WHERE u.id = @user_id`,
			PreparedSQL: `SELECT u.id, u.name,
				(SELECT json_agg(p.*) FROM posts p WHERE p.user_id = u.id) as posts
				FROM users u WHERE u.id = $1`,
			ResultType: parser.QueryResultRow,
			Parameters: []parser.QueryParameter{
				{Name: "user_id", Position: 1},
			},
		},
	}

	result, err := Queries(ctx, dbURL, queries, schema)
	if err != nil {
		t.Fatalf("Queries() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 query, got %d", len(result))
	}

	q := result[0]
	if len(q.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(q.Columns))
	}

	postsCol := q.Columns[2]
	if postsCol.Name != "posts" {
		t.Errorf("expected posts column, got %s", postsCol.Name)
	}
	if !postsCol.IsJSONAgg {
		t.Error("posts column should be marked as IsJSONAgg")
	}
	if postsCol.JSONElemGoType != "Posts" {
		t.Errorf("JSONElemGoType = %q, want %q", postsCol.JSONElemGoType, "Posts")
	}
}

func TestOidToTypeName(t *testing.T) {
	tests := []struct {
		oid  uint32
		want string
	}{
		{16, "boolean"},
		{20, "bigint"},
		{21, "smallint"},
		{23, "integer"},
		{25, "text"},
		{700, "real"},
		{701, "double precision"},
		{1043, "character varying"},
		{1082, "date"},
		{1114, "timestamp"},
		{1184, "timestamp with time zone"},
		{2950, "uuid"},
		{3802, "jsonb"},
		{99999, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := oidToTypeName(tt.oid)
			if got != tt.want {
				t.Errorf("oidToTypeName(%d) = %q, want %q", tt.oid, got, tt.want)
			}
		})
	}
}

func TestPgTypeToGo(t *testing.T) {
	tests := []struct {
		pgType   string
		nullable bool
		wantType string
		wantImp  string
	}{
		{"integer", false, "int32", ""},
		{"integer", true, "*int32", ""},
		{"bigint", false, "int64", ""},
		{"text", false, "string", ""},
		{"text", true, "*string", ""},
		{"boolean", false, "bool", ""},
		{"timestamp with time zone", false, "time.Time", "time"},
		{"timestamp with time zone", true, "*time.Time", "time"},
		{"jsonb", false, "json.RawMessage", "encoding/json"},
		{"uuid", false, "string", ""},
		{"integer[]", false, "[]int32", ""},
	}

	for _, tt := range tests {
		t.Run(tt.pgType, func(t *testing.T) {
			gotType, gotImp := pgTypeToGo(tt.pgType, tt.nullable, nil)
			if gotType != tt.wantType {
				t.Errorf("pgTypeToGo(%q, %v) type = %q, want %q", tt.pgType, tt.nullable, gotType, tt.wantType)
			}
			if gotImp != tt.wantImp {
				t.Errorf("pgTypeToGo(%q, %v) import = %q, want %q", tt.pgType, tt.nullable, gotImp, tt.wantImp)
			}
		})
	}
}
