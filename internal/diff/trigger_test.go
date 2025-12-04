package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateTrigger(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Triggers: []parser.Trigger{
			{Name: "audit_trigger", Table: "users", Timing: "AFTER", Events: []string{"INSERT", "UPDATE"}, Function: "audit_func", ForEach: "ROW"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateTrigger && c.ObjectName() == "audit_trigger" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateTrigger change for audit_trigger")
	}
}

func TestCompare_DropTrigger(t *testing.T) {
	current := &parser.Schema{
		Triggers: []parser.Trigger{
			{Name: "old_trigger", Table: "users"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropTrigger && c.ObjectName() == "old_trigger" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropTrigger change for old_trigger")
	}
}

func TestTriggerChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *TriggerChange
		want   []string
	}{
		{
			name: "create trigger basic",
			change: &TriggerChange{
				ChangeType: CreateTrigger,
				Trigger: parser.Trigger{
					Name:     "my_trigger",
					Table:    "users",
					Timing:   "BEFORE",
					Events:   []string{"INSERT"},
					Function: "my_func",
					ForEach:  "ROW",
				},
			},
			want: []string{"CREATE TRIGGER", "my_trigger", "BEFORE", "INSERT", "ON", "users", "FOR EACH ROW", "EXECUTE FUNCTION", "my_func"},
		},
		{
			name: "create trigger with multiple events",
			change: &TriggerChange{
				ChangeType: CreateTrigger,
				Trigger: parser.Trigger{
					Name:     "audit_trigger",
					Table:    "orders",
					Timing:   "AFTER",
					Events:   []string{"INSERT", "UPDATE", "DELETE"},
					Function: "audit_func",
					ForEach:  "ROW",
				},
			},
			want: []string{"INSERT OR UPDATE OR DELETE"},
		},
		{
			name: "create trigger with when clause",
			change: &TriggerChange{
				ChangeType: CreateTrigger,
				Trigger: parser.Trigger{
					Name:     "conditional_trigger",
					Table:    "users",
					Timing:   "AFTER",
					Events:   []string{"UPDATE"},
					Function: "notify_func",
					ForEach:  "ROW",
					When:     "OLD.status IS DISTINCT FROM NEW.status",
				},
			},
			want: []string{"WHEN (OLD.status IS DISTINCT FROM NEW.status)"},
		},
		{
			name: "drop trigger",
			change: &TriggerChange{
				ChangeType: DropTrigger,
				Trigger:    parser.Trigger{Name: "old_trigger", Table: "users"},
			},
			want: []string{"DROP TRIGGER", "old_trigger", "ON", "users"},
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

func TestTriggerChange_DownSQL(t *testing.T) {
	createChange := &TriggerChange{
		ChangeType: CreateTrigger,
		Trigger: parser.Trigger{
			Name:     "my_trigger",
			Table:    "users",
			Timing:   "BEFORE",
			Events:   []string{"INSERT"},
			Function: "my_func",
			ForEach:  "ROW",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP TRIGGER") {
		t.Error("DownSQL for CreateTrigger should contain DROP TRIGGER")
	}

	oldTrigger := parser.Trigger{
		Name:     "old_trigger",
		Table:    "users",
		Timing:   "AFTER",
		Events:   []string{"UPDATE"},
		Function: "audit_func",
		ForEach:  "ROW",
	}
	dropChange := &TriggerChange{
		ChangeType: DropTrigger,
		Trigger:    parser.Trigger{Name: "old_trigger", Table: "users"},
		OldTrigger: &oldTrigger,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE TRIGGER") {
		t.Error("DownSQL for DropTrigger with OldTrigger should contain CREATE TRIGGER")
	}

	dropChangeNoOld := &TriggerChange{
		ChangeType: DropTrigger,
		Trigger:    parser.Trigger{Name: "old_trigger", Table: "users"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropTrigger without OldTrigger should indicate IRREVERSIBLE")
	}
}

func TestTriggerChange_IsReversible(t *testing.T) {
	createChange := &TriggerChange{
		ChangeType: CreateTrigger,
		Trigger: parser.Trigger{
			Name:     "test",
			Table:    "users",
			Timing:   "BEFORE",
			Events:   []string{"INSERT"},
			Function: "my_func",
			ForEach:  "ROW",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateTrigger should be reversible")
	}

	oldTrigger := parser.Trigger{Name: "test", Table: "users"}
	dropChangeWithOld := &TriggerChange{
		ChangeType: DropTrigger,
		Trigger:    parser.Trigger{Name: "test", Table: "users"},
		OldTrigger: &oldTrigger,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropTrigger with OldTrigger should be reversible")
	}

	dropChangeNoOld := &TriggerChange{
		ChangeType: DropTrigger,
		Trigger:    parser.Trigger{Name: "test", Table: "users"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropTrigger without OldTrigger should not be reversible")
	}
}
