package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateView(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Views: []parser.View{
			{Name: "active_users", Definition: "SELECT * FROM users WHERE active = true"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateView && c.ObjectName() == "active_users" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateView change for active_users")
	}
}

func TestCompare_DropView(t *testing.T) {
	current := &parser.Schema{
		Views: []parser.View{
			{Name: "old_view", Definition: "SELECT 1"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropView && c.ObjectName() == "old_view" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropView change for old_view")
	}
}

func TestCompare_AlterView(t *testing.T) {
	current := &parser.Schema{
		Views: []parser.View{
			{Name: "user_summary", Definition: "SELECT id, name FROM users"},
		},
	}
	desired := &parser.Schema{
		Views: []parser.View{
			{Name: "user_summary", Definition: "SELECT id, name, email FROM users"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterView && c.ObjectName() == "user_summary" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected AlterView change for user_summary")
	}
}

func TestViewChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *ViewChange
		want   []string
	}{
		{
			name: "create view",
			change: &ViewChange{
				ChangeType: CreateView,
				View: parser.View{
					Name:       "active_users",
					Definition: "SELECT * FROM users WHERE active = true",
				},
			},
			want: []string{"CREATE OR REPLACE VIEW", "active_users", "AS", "SELECT * FROM users WHERE active = true"},
		},
		{
			name: "create view with schema",
			change: &ViewChange{
				ChangeType: CreateView,
				View: parser.View{
					Schema:     "myschema",
					Name:       "my_view",
					Definition: "SELECT 1",
				},
			},
			want: []string{"CREATE OR REPLACE VIEW", "myschema", "my_view", "AS"},
		},
		{
			name: "alter view",
			change: &ViewChange{
				ChangeType: AlterView,
				View: parser.View{
					Name:       "user_summary",
					Definition: "SELECT id, name, email FROM users",
				},
			},
			want: []string{"CREATE OR REPLACE VIEW", "user_summary", "AS", "SELECT id, name, email FROM users"},
		},
		{
			name: "drop view",
			change: &ViewChange{
				ChangeType: DropView,
				View:       parser.View{Name: "old_view"},
			},
			want: []string{"DROP VIEW", "old_view"},
		},
		{
			name: "drop view with schema",
			change: &ViewChange{
				ChangeType: DropView,
				View:       parser.View{Schema: "myschema", Name: "my_view"},
			},
			want: []string{"DROP VIEW", "myschema", "my_view"},
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

func TestViewChange_DownSQL(t *testing.T) {
	createChange := &ViewChange{
		ChangeType: CreateView,
		View: parser.View{
			Name:       "my_view",
			Definition: "SELECT 1",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP VIEW") {
		t.Error("DownSQL for CreateView should contain DROP VIEW")
	}

	oldView := parser.View{Name: "old_view", Definition: "SELECT * FROM users"}
	dropChange := &ViewChange{
		ChangeType: DropView,
		View:       parser.View{Name: "old_view"},
		OldView:    &oldView,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE OR REPLACE VIEW") {
		t.Error("DownSQL for DropView with OldView should contain CREATE OR REPLACE VIEW")
	}
	if !strings.Contains(downSQL, "SELECT * FROM users") {
		t.Error("DownSQL for DropView should preserve view definition")
	}

	dropChangeNoOld := &ViewChange{
		ChangeType: DropView,
		View:       parser.View{Name: "old_view"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropView without OldView should indicate IRREVERSIBLE")
	}

	oldViewForAlter := parser.View{Name: "my_view", Definition: "SELECT 1"}
	alterChange := &ViewChange{
		ChangeType: AlterView,
		View: parser.View{
			Name:       "my_view",
			Definition: "SELECT 2",
		},
		OldView: &oldViewForAlter,
	}
	alterDownSQL := alterChange.DownSQL()
	if !strings.Contains(alterDownSQL, "SELECT 1") {
		t.Error("DownSQL for AlterView should restore old definition")
	}

	alterChangeNoOld := &ViewChange{
		ChangeType: AlterView,
		View: parser.View{
			Name:       "my_view",
			Definition: "SELECT 2",
		},
	}
	alterDownSQLNoOld := alterChangeNoOld.DownSQL()
	if !strings.Contains(alterDownSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for AlterView without OldView should indicate IRREVERSIBLE")
	}
}

func TestViewChange_IsReversible(t *testing.T) {
	createChange := &ViewChange{
		ChangeType: CreateView,
		View: parser.View{
			Name:       "test",
			Definition: "SELECT 1",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateView should be reversible")
	}

	oldView := parser.View{Name: "test", Definition: "SELECT 1"}
	dropChangeWithOld := &ViewChange{
		ChangeType: DropView,
		View:       parser.View{Name: "test"},
		OldView:    &oldView,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropView with OldView should be reversible")
	}

	dropChangeNoOld := &ViewChange{
		ChangeType: DropView,
		View:       parser.View{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropView without OldView should not be reversible")
	}

	alterChangeWithOld := &ViewChange{
		ChangeType: AlterView,
		View:       parser.View{Name: "test", Definition: "SELECT 2"},
		OldView:    &oldView,
	}
	if !alterChangeWithOld.IsReversible() {
		t.Error("AlterView with OldView should be reversible")
	}

	alterChangeNoOld := &ViewChange{
		ChangeType: AlterView,
		View:       parser.View{Name: "test", Definition: "SELECT 2"},
	}
	if alterChangeNoOld.IsReversible() {
		t.Error("AlterView without OldView should not be reversible")
	}
}

func TestCompareViews(t *testing.T) {
	current := []parser.View{
		{Name: "view_to_drop", Definition: "SELECT 1"},
		{Name: "view_to_alter", Definition: "SELECT old"},
		{Name: "unchanged_view", Definition: "SELECT same"},
	}
	desired := []parser.View{
		{Name: "view_to_create", Definition: "SELECT new"},
		{Name: "view_to_alter", Definition: "SELECT modified"},
		{Name: "unchanged_view", Definition: "SELECT same"},
	}

	changes := compareViews(current, desired)

	foundCreate := false
	foundDrop := false
	foundAlter := false

	for _, c := range changes {
		vc := c.(*ViewChange)
		switch {
		case vc.ChangeType == CreateView && vc.View.Name == "view_to_create":
			foundCreate = true
		case vc.ChangeType == DropView && vc.View.Name == "view_to_drop":
			foundDrop = true
		case vc.ChangeType == AlterView && vc.View.Name == "view_to_alter":
			foundAlter = true
			if vc.OldView == nil {
				t.Error("AlterView should have OldView set")
			}
		}
	}

	if !foundCreate {
		t.Error("expected CreateView for view_to_create")
	}
	if !foundDrop {
		t.Error("expected DropView for view_to_drop")
	}
	if !foundAlter {
		t.Error("expected AlterView for view_to_alter")
	}

	for _, c := range changes {
		if c.ObjectName() == "unchanged_view" {
			t.Error("unchanged_view should not appear in changes")
		}
	}
}

func TestCompareViews_Normalization(t *testing.T) {
	current := []parser.View{
		{Name: "spacy_view", Definition: "SELECT   id,    name   FROM   users"},
	}
	desired := []parser.View{
		{Name: "spacy_view", Definition: "SELECT id, name FROM users"},
	}

	changes := compareViews(current, desired)

	if len(changes) > 0 {
		t.Error("views with equivalent definitions (differing only in whitespace) should not produce changes")
	}
}
