package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sql")

	content := "CREATE TABLE users (id SERIAL PRIMARY KEY);"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got != content {
		t.Errorf("LoadFile() = %q, want %q", got, content)
	}
}

func TestLoadFile_NotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/path/file.sql")
	if err == nil {
		t.Error("LoadFile() expected error for nonexistent file, got nil")
	}
}

func TestSchema_ObjectCount(t *testing.T) {
	tests := []struct {
		name   string
		schema Schema
		want   int
	}{
		{
			name:   "empty schema",
			schema: Schema{},
			want:   0,
		},
		{
			name: "tables only",
			schema: Schema{
				Tables: []Table{{Name: "users"}, {Name: "posts"}},
			},
			want: 2,
		},
		{
			name: "mixed objects",
			schema: Schema{
				Tables:    []Table{{Name: "users"}},
				Indexes:   []Index{{Name: "idx_users"}},
				Views:     []View{{Name: "v_users"}},
				Functions: []Function{{Name: "fn_test"}},
				Triggers:  []Trigger{{Name: "tr_test"}},
				Sequences: []Sequence{{Name: "seq_test"}},
				Enums:     []Enum{{Name: "status"}},
			},
			want: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.schema.ObjectCount(); got != tt.want {
				t.Errorf("ObjectCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSchema_Lint(t *testing.T) {
	tests := []struct {
		name         string
		schema       Schema
		wantWarnings int
	}{
		{
			name:         "empty schema",
			schema:       Schema{},
			wantWarnings: 0,
		},
		{
			name: "table with PK column",
			schema: Schema{
				Tables: []Table{
					{
						Name: "users",
						Columns: []Column{
							{Name: "id", Type: "integer", PrimaryKey: true},
						},
					},
				},
			},
			wantWarnings: 0,
		},
		{
			name: "table with PK constraint",
			schema: Schema{
				Tables: []Table{
					{
						Name:    "users",
						Columns: []Column{{Name: "id", Type: "integer"}},
						Constraints: []Constraint{
							{Name: "users_pkey", Type: "PRIMARY KEY", Columns: []string{"id"}},
						},
					},
				},
			},
			wantWarnings: 0,
		},
		{
			name: "table without PK",
			schema: Schema{
				Tables: []Table{
					{
						Name:    "users",
						Columns: []Column{{Name: "name", Type: "text"}},
					},
				},
			},
			wantWarnings: 1,
		},
		{
			name: "multiple tables some without PK",
			schema: Schema{
				Tables: []Table{
					{
						Name:    "users",
						Columns: []Column{{Name: "id", Type: "integer", PrimaryKey: true}},
					},
					{
						Name:    "logs",
						Columns: []Column{{Name: "message", Type: "text"}},
					},
				},
			},
			wantWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := tt.schema.Lint()
			if len(warnings) != tt.wantWarnings {
				t.Errorf("Lint() returned %d warnings, want %d: %v", len(warnings), tt.wantWarnings, warnings)
			}
		})
	}
}
