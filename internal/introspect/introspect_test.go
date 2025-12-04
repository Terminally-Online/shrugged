package introspect

import (
	"context"
	"testing"
	"time"

	"shrugged/internal/docker"
	"shrugged/internal/parser"
)

func TestDatabase_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}

	defer func() {
		_ = docker.StopContainer(context.Background(), container.ID)
	}()

	schemaSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			published BOOLEAN NOT NULL DEFAULT FALSE
		);

		CREATE INDEX idx_posts_user_id ON posts(user_id);
		CREATE INDEX idx_posts_published ON posts(published) WHERE published = TRUE;

		CREATE VIEW published_posts AS
		SELECT p.id, p.title, u.name as author
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.published = TRUE;

		CREATE FUNCTION update_timestamp() RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
	`

	if err := docker.ExecuteSQL(ctx, container, schemaSQL); err != nil {
		t.Fatalf("ExecuteSQL() error = %v", err)
	}

	schema, err := Database(ctx, container.ConnectionString())
	if err != nil {
		t.Fatalf("Database() error = %v", err)
	}

	if len(schema.Tables) != 2 {
		t.Errorf("Tables count = %d, want 2", len(schema.Tables))
	}

	usersTable := findTable(schema.Tables, "users")
	if usersTable == nil {
		t.Fatal("users table not found")
	}
	if len(usersTable.Columns) != 4 {
		t.Errorf("users.Columns count = %d, want 4", len(usersTable.Columns))
	}

	postsTable := findTable(schema.Tables, "posts")
	if postsTable == nil {
		t.Fatal("posts table not found")
	}

	hasPK := false
	hasFK := false
	for _, c := range postsTable.Constraints {
		if c.Type == "PRIMARY KEY" {
			hasPK = true
		}
		if c.Type == "FOREIGN KEY" {
			hasFK = true
			if c.OnDelete != "CASCADE" {
				t.Errorf("FK OnDelete = %q, want %q", c.OnDelete, "CASCADE")
			}
		}
	}
	if !hasPK {
		t.Error("posts table missing PRIMARY KEY constraint")
	}
	if !hasFK {
		t.Error("posts table missing FOREIGN KEY constraint")
	}

	if len(schema.Indexes) != 2 {
		t.Errorf("Indexes count = %d, want 2", len(schema.Indexes))
	}

	partialIdx := findIndex(schema.Indexes, "idx_posts_published")
	if partialIdx == nil {
		t.Fatal("idx_posts_published not found")
	}
	if partialIdx.Where == "" {
		t.Error("partial index missing WHERE clause")
	}

	if len(schema.Views) != 1 {
		t.Errorf("Views count = %d, want 1", len(schema.Views))
	}
	if schema.Views[0].Name != "published_posts" {
		t.Errorf("View name = %q, want %q", schema.Views[0].Name, "published_posts")
	}

	if len(schema.Functions) != 1 {
		t.Errorf("Functions count = %d, want 1", len(schema.Functions))
	}
	if schema.Functions[0].Name != "update_timestamp" {
		t.Errorf("Function name = %q, want %q", schema.Functions[0].Name, "update_timestamp")
	}
	if schema.Functions[0].Language != "plpgsql" {
		t.Errorf("Function language = %q, want %q", schema.Functions[0].Language, "plpgsql")
	}
}

func TestDatabase_EmptySchema(t *testing.T) {
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

	defer func() {
		_ = docker.StopContainer(context.Background(), container.ID)
	}()

	schema, err := Database(ctx, container.ConnectionString())
	if err != nil {
		t.Fatalf("Database() error = %v", err)
	}

	if len(schema.Tables) != 0 {
		t.Errorf("Tables count = %d, want 0", len(schema.Tables))
	}
	if len(schema.Views) != 0 {
		t.Errorf("Views count = %d, want 0", len(schema.Views))
	}
	if len(schema.Functions) != 0 {
		t.Errorf("Functions count = %d, want 0", len(schema.Functions))
	}
}

func TestResolveType(t *testing.T) {
	tests := []struct {
		dataType string
		udtName  string
		want     string
	}{
		{"integer", "int4", "integer"},
		{"bigint", "int8", "bigint"},
		{"smallint", "int2", "smallint"},
		{"real", "float4", "real"},
		{"double precision", "float8", "double precision"},
		{"boolean", "bool", "boolean"},
		{"timestamp with time zone", "timestamptz", "timestamp with time zone"},
		{"timestamp without time zone", "timestamp", "timestamp without time zone"},
		{"text", "text", "text"},
		{"USER-DEFINED", "my_enum", "my_enum"},
	}

	for _, tt := range tests {
		t.Run(tt.dataType+"_"+tt.udtName, func(t *testing.T) {
			udtName := tt.udtName
			got := resolveType(tt.dataType, &udtName)
			if got != tt.want {
				t.Errorf("resolveType(%q, %q) = %q, want %q", tt.dataType, tt.udtName, got, tt.want)
			}
		})
	}
}

func findTable(tables []parser.Table, name string) *parser.Table {
	for i := range tables {
		if tables[i].Name == name {
			return &tables[i]
		}
	}
	return nil
}

func findIndex(indexes []parser.Index, name string) *parser.Index {
	for i := range indexes {
		if indexes[i].Name == name {
			return &indexes[i]
		}
	}
	return nil
}
