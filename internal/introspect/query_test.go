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
	if getUserQuery.Parameters[0].Type == "" {
		t.Error("parameter Type should be set")
	}

	for _, col := range getUserQuery.Columns {
		if col.Type == "" {
			t.Errorf("column %s Type should be set", col.Name)
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
	if postsCol.JSONElemType != "posts" {
		t.Errorf("JSONElemType = %q, want %q", postsCol.JSONElemType, "posts")
	}
}

func TestResolveTypeName(t *testing.T) {
	tests := []struct {
		name    string
		oid     uint32
		typeMap map[uint32]string
		want    string
	}{
		{
			name:    "known OID takes precedence over typeMap",
			oid:     2950,
			typeMap: map[uint32]string{2950: "uuid"},
			want:    "uuid",
		},
		{
			name:    "uuid array via known OID",
			oid:     2951,
			typeMap: nil,
			want:    "uuid[]",
		},
		{
			name:    "underscore-prefixed pg_type name converts to array format",
			oid:     50000,
			typeMap: map[uint32]string{50000: "_uuid"},
			want:    "uuid[]",
		},
		{
			name:    "underscore-prefixed int4 converts to int4[]",
			oid:     50001,
			typeMap: map[uint32]string{50001: "_int4"},
			want:    "int4[]",
		},
		{
			name:    "underscore-prefixed text converts to text[]",
			oid:     50002,
			typeMap: map[uint32]string{50002: "_text"},
			want:    "text[]",
		},
		{
			name:    "underscore-prefixed timestamp converts to timestamp[]",
			oid:     50003,
			typeMap: map[uint32]string{50003: "_timestamp"},
			want:    "timestamp[]",
		},
		{
			name:    "non-prefixed custom type passes through",
			oid:     50004,
			typeMap: map[uint32]string{50004: "my_custom_type"},
			want:    "my_custom_type",
		},
		{
			name:    "unknown OID not in typeMap returns unknown",
			oid:     99999,
			typeMap: map[uint32]string{},
			want:    "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveTypeName(tt.oid, tt.typeMap)
			if got != tt.want {
				t.Errorf("resolveTypeName(%d, typeMap) = %q, want %q", tt.oid, got, tt.want)
			}
		})
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
		{2951, "uuid[]"},
		{1115, "timestamp[]"},
		{1182, "date[]"},
		{1185, "timestamp with time zone[]"},
		{1231, "numeric[]"},
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

func TestIntrospectQueries_ArrayTypeParams(t *testing.T) {
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
		CREATE TABLE nodes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL
		);

		CREATE TABLE environments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			entry_nodes UUID[] NOT NULL DEFAULT '{}',
			exit_nodes UUID[] NOT NULL DEFAULT '{}',
			tags TEXT[] NOT NULL DEFAULT '{}',
			scores INTEGER[] NOT NULL DEFAULT '{}'
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
			Name:        "CreateEnvironment",
			SQL:         "INSERT INTO environments (name, entry_nodes, exit_nodes, tags, scores) VALUES (@name, @entry_nodes, @exit_nodes, @tags, @scores)",
			PreparedSQL: "INSERT INTO environments (name, entry_nodes, exit_nodes, tags, scores) VALUES ($1, $2, $3, $4, $5)",
			ResultType:  parser.QueryResultExec,
			Parameters: []parser.QueryParameter{
				{Name: "name", Position: 1},
				{Name: "entry_nodes", Position: 2},
				{Name: "exit_nodes", Position: 3},
				{Name: "tags", Position: 4},
				{Name: "scores", Position: 5},
			},
		},
		{
			Name:        "GetEnvironmentsByEntryNodes",
			SQL:         "SELECT id, name, entry_nodes, exit_nodes, tags, scores FROM environments WHERE entry_nodes && @node_ids",
			PreparedSQL: "SELECT id, name, entry_nodes, exit_nodes, tags, scores FROM environments WHERE entry_nodes && $1",
			ResultType:  parser.QueryResultRows,
			Parameters: []parser.QueryParameter{
				{Name: "node_ids", Position: 1},
			},
		},
	}

	result, err := Queries(ctx, dbURL, queries, schema)
	if err != nil {
		t.Fatalf("Queries() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(result))
	}

	createEnv := result[0]
	expectedParamTypes := map[string]string{
		"name":        "text",
		"entry_nodes": "uuid[]",
		"exit_nodes":  "uuid[]",
		"tags":        "text[]",
		"scores":      "integer[]",
	}
	for _, p := range createEnv.Parameters {
		want, ok := expectedParamTypes[p.Name]
		if !ok {
			continue
		}
		if p.Type != want {
			t.Errorf("CreateEnvironment param %q type = %q, want %q", p.Name, p.Type, want)
		}
	}

	getByEntry := result[1]
	if len(getByEntry.Parameters) != 1 {
		t.Fatalf("expected 1 parameter for GetEnvironmentsByEntryNodes, got %d", len(getByEntry.Parameters))
	}
	if getByEntry.Parameters[0].Type != "uuid[]" {
		t.Errorf("GetEnvironmentsByEntryNodes param type = %q, want %q", getByEntry.Parameters[0].Type, "uuid[]")
	}

	// Verify array columns in the result set are also correctly typed.
	colTypes := map[string]string{
		"entry_nodes": "uuid[]",
		"exit_nodes":  "uuid[]",
		"tags":        "text[]",
		"scores":      "integer[]",
	}
	for _, col := range getByEntry.Columns {
		want, ok := colTypes[col.Name]
		if !ok {
			continue
		}
		if col.Type != want {
			t.Errorf("GetEnvironmentsByEntryNodes column %q type = %q, want %q", col.Name, col.Type, want)
		}
	}
}

func TestMapParamsToColumns_Insert(t *testing.T) {
	sql := "INSERT INTO story_nodes (id, environment_id, stat_check, generated_by) VALUES ($1, $2, $3, $4)"
	params := []parser.QueryParameter{
		{Name: "id", Position: 1},
		{Name: "environment_id", Position: 2},
		{Name: "stat_check", Position: 3},
		{Name: "generated_by", Position: 4},
	}

	result := mapParamsToColumns(sql, params)

	expected := map[string]string{
		"id":             "id",
		"environment_id": "environment_id",
		"stat_check":     "stat_check",
		"generated_by":   "generated_by",
	}
	for paramName, wantCol := range expected {
		if gotCol, ok := result[paramName]; !ok {
			t.Errorf("param %q not mapped", paramName)
		} else if gotCol != wantCol {
			t.Errorf("param %q mapped to %q, want %q", paramName, gotCol, wantCol)
		}
	}
}

func TestMapParamsToColumns_Update(t *testing.T) {
	sql := "UPDATE choice_edges SET to_node_id = $1 WHERE id = $2"
	params := []parser.QueryParameter{
		{Name: "to_node_id", Position: 1},
		{Name: "id", Position: 2},
	}

	result := mapParamsToColumns(sql, params)

	if got, ok := result["to_node_id"]; !ok || got != "to_node_id" {
		t.Errorf("to_node_id mapping = %q, want %q", got, "to_node_id")
	}
	if got, ok := result["id"]; !ok || got != "id" {
		t.Errorf("id mapping = %q, want %q", got, "id")
	}
}

func TestMarkNullableParameters(t *testing.T) {
	schema := &parser.Schema{
		Tables: []parser.Table{
			{
				Name: "story_nodes",
				Columns: []parser.Column{
					{Name: "id", Type: "uuid", Nullable: false},
					{Name: "environment_id", Type: "uuid", Nullable: false},
					{Name: "content", Type: "text", Nullable: false},
					{Name: "stat_check", Type: "jsonb", Nullable: true},
					{Name: "generation_ctx", Type: "jsonb", Nullable: true},
					{Name: "generated_by", Type: "uuid", Nullable: true},
				},
			},
		},
	}

	sql := "INSERT INTO story_nodes (id, environment_id, content, stat_check, generation_ctx, generated_by) VALUES ($1, $2, $3, $4, $5, $6)"
	params := []parser.QueryParameter{
		{Name: "id", Position: 1},
		{Name: "environment_id", Position: 2},
		{Name: "content", Position: 3},
		{Name: "stat_check", Position: 4},
		{Name: "generation_ctx", Position: 5},
		{Name: "generated_by", Position: 6},
	}

	markNullableParameters(sql, params, schema)

	expectations := map[string]bool{
		"id":             false,
		"environment_id": false,
		"content":        false,
		"stat_check":     true,
		"generation_ctx": true,
		"generated_by":   true,
	}

	for _, p := range params {
		want, ok := expectations[p.Name]
		if !ok {
			continue
		}
		if p.Nullable != want {
			t.Errorf("param %q Nullable = %v, want %v", p.Name, p.Nullable, want)
		}
	}
}

func TestMarkNullableParameters_Update(t *testing.T) {
	schema := &parser.Schema{
		Tables: []parser.Table{
			{
				Name: "choice_edges",
				Columns: []parser.Column{
					{Name: "id", Type: "uuid", Nullable: false},
					{Name: "to_node_id", Type: "uuid", Nullable: true},
				},
			},
		},
	}

	sql := "UPDATE choice_edges SET to_node_id = $1 WHERE id = $2"
	params := []parser.QueryParameter{
		{Name: "to_node_id", Position: 1},
		{Name: "id", Position: 2},
	}

	markNullableParameters(sql, params, schema)

	if params[0].Nullable != true {
		t.Errorf("to_node_id Nullable = %v, want true", params[0].Nullable)
	}
	if params[1].Nullable != false {
		t.Errorf("id Nullable = %v, want false", params[1].Nullable)
	}
}

func TestMarkNullableParameters_NilSchema(t *testing.T) {
	params := []parser.QueryParameter{
		{Name: "id", Position: 1},
	}

	markNullableParameters("INSERT INTO t (id) VALUES ($1)", params, nil)

	if params[0].Nullable {
		t.Error("param should not be nullable with nil schema")
	}
}

func TestMarkNullableParameters_SelectQuery(t *testing.T) {
	schema := &parser.Schema{
		Tables: []parser.Table{
			{
				Name: "story_nodes",
				Columns: []parser.Column{
					{Name: "id", Type: "uuid", Nullable: false},
				},
			},
		},
	}

	params := []parser.QueryParameter{
		{Name: "id", Position: 1},
	}

	markNullableParameters("SELECT * FROM story_nodes WHERE id = $1", params, schema)

	if params[0].Nullable {
		t.Error("SELECT param should not be marked nullable (no INSERT/UPDATE match)")
	}
}

func TestIntrospectQueries_NullableParams(t *testing.T) {
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
		CREATE TABLE items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			description TEXT,
			metadata JSONB,
			parent_id UUID REFERENCES items(id)
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
			Name:        "CreateItem",
			SQL:         "INSERT INTO items (id, name, description, metadata, parent_id) VALUES (@id, @name, @description, @metadata, @parent_id) RETURNING *",
			PreparedSQL: "INSERT INTO items (id, name, description, metadata, parent_id) VALUES ($1, $2, $3, $4, $5) RETURNING *",
			ResultType:  parser.QueryResultRow,
			Parameters: []parser.QueryParameter{
				{Name: "id", Position: 1},
				{Name: "name", Position: 2},
				{Name: "description", Position: 3},
				{Name: "metadata", Position: 4},
				{Name: "parent_id", Position: 5},
			},
		},
		{
			Name:        "UpdateItemParent",
			SQL:         "UPDATE items SET parent_id = @parent_id WHERE id = @id",
			PreparedSQL: "UPDATE items SET parent_id = $1 WHERE id = $2",
			ResultType:  parser.QueryResultExec,
			Parameters: []parser.QueryParameter{
				{Name: "parent_id", Position: 1},
				{Name: "id", Position: 2},
			},
		},
	}

	result, err := Queries(ctx, dbURL, queries, schema)
	if err != nil {
		t.Fatalf("Queries() error = %v", err)
	}

	createItem := result[0]
	expectedNullable := map[string]bool{
		"id":          false,
		"name":        false,
		"description": true,
		"metadata":    true,
		"parent_id":   true,
	}
	for _, p := range createItem.Parameters {
		want, ok := expectedNullable[p.Name]
		if !ok {
			continue
		}
		if p.Nullable != want {
			t.Errorf("CreateItem param %q Nullable = %v, want %v", p.Name, p.Nullable, want)
		}
	}

	updateItem := result[1]
	for _, p := range updateItem.Parameters {
		switch p.Name {
		case "parent_id":
			if !p.Nullable {
				t.Error("UpdateItemParent param parent_id should be nullable")
			}
		case "id":
			if p.Nullable {
				t.Error("UpdateItemParent param id should not be nullable")
			}
		}
	}
}
