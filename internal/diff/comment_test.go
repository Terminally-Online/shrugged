package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateComment(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Comments: []parser.Comment{
			{ObjectType: "TABLE", Name: "users", Comment: "User accounts table"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateComment && c.ObjectName() == "users" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateComment change for users")
	}
}

func TestCompare_DropComment(t *testing.T) {
	current := &parser.Schema{
		Comments: []parser.Comment{
			{ObjectType: "TABLE", Name: "old_table", Comment: "Old comment"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropComment && c.ObjectName() == "old_table" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropComment change for old_table")
	}
}

func TestCompare_UpdateComment(t *testing.T) {
	current := &parser.Schema{
		Comments: []parser.Comment{
			{ObjectType: "TABLE", Name: "users", Comment: "Old description"},
		},
	}
	desired := &parser.Schema{
		Comments: []parser.Comment{
			{ObjectType: "TABLE", Name: "users", Comment: "New description"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateComment && c.ObjectName() == "users" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateComment (update) change for users")
	}
}

func TestCommentChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *CommentChange
		want   []string
	}{
		{
			name: "create comment on table",
			change: &CommentChange{
				ChangeType: CreateComment,
				Comment: parser.Comment{
					ObjectType: "TABLE",
					Name:       "users",
					Comment:    "User accounts table",
				},
			},
			want: []string{"COMMENT ON", "TABLE", "users", "IS", "'User accounts table'"},
		},
		{
			name: "create comment on column",
			change: &CommentChange{
				ChangeType: CreateComment,
				Comment: parser.Comment{
					ObjectType: "COLUMN",
					Name:       "users",
					Column:     "email",
					Comment:    "User email address",
				},
			},
			want: []string{"COMMENT ON", "COLUMN", "users", "email", "IS", "'User email address'"},
		},
		{
			name: "create comment on function",
			change: &CommentChange{
				ChangeType: CreateComment,
				Comment: parser.Comment{
					ObjectType: "FUNCTION",
					Name:       "my_func",
					Comment:    "Utility function",
				},
			},
			want: []string{"COMMENT ON", "FUNCTION", "my_func", "IS", "'Utility function'"},
		},
		{
			name: "create comment on schema",
			change: &CommentChange{
				ChangeType: CreateComment,
				Comment: parser.Comment{
					ObjectType: "SCHEMA",
					Name:       "myschema",
					Comment:    "Application schema",
				},
			},
			want: []string{"COMMENT ON", "SCHEMA", "myschema", "IS", "'Application schema'"},
		},
		{
			name: "create comment with schema qualifier",
			change: &CommentChange{
				ChangeType: CreateComment,
				Comment: parser.Comment{
					ObjectType: "TABLE",
					Schema:     "myschema",
					Name:       "my_table",
					Comment:    "A table",
				},
			},
			want: []string{"COMMENT ON", "TABLE", "myschema", "my_table", "IS"},
		},
		{
			name: "drop comment",
			change: &CommentChange{
				ChangeType: DropComment,
				Comment: parser.Comment{
					ObjectType: "TABLE",
					Name:       "users",
				},
			},
			want: []string{"COMMENT ON", "TABLE", "users", "IS NULL"},
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

func TestCommentChange_DownSQL(t *testing.T) {
	createChange := &CommentChange{
		ChangeType: CreateComment,
		Comment: parser.Comment{
			ObjectType: "TABLE",
			Name:       "users",
			Comment:    "New comment",
		},
	}
	downSQL := createChange.DownSQL()
	if !strings.Contains(downSQL, "IS NULL") {
		t.Error("DownSQL for CreateComment without OldComment should set IS NULL")
	}

	oldComment := parser.Comment{ObjectType: "TABLE", Name: "users", Comment: "Old comment"}
	createChangeWithOld := &CommentChange{
		ChangeType: CreateComment,
		Comment: parser.Comment{
			ObjectType: "TABLE",
			Name:       "users",
			Comment:    "New comment",
		},
		OldComment: &oldComment,
	}
	downSQLWithOld := createChangeWithOld.DownSQL()
	if !strings.Contains(downSQLWithOld, "'Old comment'") {
		t.Error("DownSQL for CreateComment with OldComment should restore old comment")
	}

	oldCommentDrop := parser.Comment{ObjectType: "TABLE", Name: "users", Comment: "Old comment"}
	dropChange := &CommentChange{
		ChangeType: DropComment,
		Comment: parser.Comment{
			ObjectType: "TABLE",
			Name:       "users",
		},
		OldComment: &oldCommentDrop,
	}
	dropDownSQL := dropChange.DownSQL()
	if !strings.Contains(dropDownSQL, "COMMENT ON") {
		t.Error("DownSQL for DropComment with OldComment should restore comment")
	}
	if !strings.Contains(dropDownSQL, "'Old comment'") {
		t.Error("DownSQL for DropComment should include old comment text")
	}
}

func TestCommentChange_IsReversible(t *testing.T) {
	change := &CommentChange{
		ChangeType: CreateComment,
		Comment: parser.Comment{
			ObjectType: "TABLE",
			Name:       "users",
			Comment:    "Test comment",
		},
	}
	if !change.IsReversible() {
		t.Error("CommentChange should always be reversible")
	}

	dropChange := &CommentChange{
		ChangeType: DropComment,
		Comment: parser.Comment{
			ObjectType: "TABLE",
			Name:       "users",
		},
	}
	if !dropChange.IsReversible() {
		t.Error("DropComment should be reversible")
	}
}

func TestCommentChange_EscapedQuotes(t *testing.T) {
	change := &CommentChange{
		ChangeType: CreateComment,
		Comment: parser.Comment{
			ObjectType: "TABLE",
			Name:       "users",
			Comment:    "User's table with \"quotes\"",
		},
	}
	sql := change.SQL()
	if !strings.Contains(sql, "User''s") {
		t.Error("SQL should escape single quotes")
	}
}
