package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateEventTrigger(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		EventTriggers: []parser.EventTrigger{
			{Name: "ddl_audit", Event: "ddl_command_end", Function: "audit_ddl"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateEventTrigger && c.ObjectName() == "ddl_audit" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateEventTrigger change for ddl_audit")
	}
}

func TestCompare_DropEventTrigger(t *testing.T) {
	current := &parser.Schema{
		EventTriggers: []parser.EventTrigger{
			{Name: "old_trigger", Event: "ddl_command_start", Function: "old_func"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropEventTrigger && c.ObjectName() == "old_trigger" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropEventTrigger change for old_trigger")
	}
}

func TestEventTriggerChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *EventTriggerChange
		want   []string
	}{
		{
			name: "create event trigger basic",
			change: &EventTriggerChange{
				ChangeType: CreateEventTrigger,
				EventTrigger: parser.EventTrigger{
					Name:     "my_trigger",
					Event:    "ddl_command_end",
					Function: "my_func",
				},
			},
			want: []string{"CREATE EVENT TRIGGER", "my_trigger", "ON", "ddl_command_end", "EXECUTE FUNCTION", "my_func"},
		},
		{
			name: "create event trigger with tags",
			change: &EventTriggerChange{
				ChangeType: CreateEventTrigger,
				EventTrigger: parser.EventTrigger{
					Name:     "table_audit",
					Event:    "ddl_command_end",
					Function: "audit_func",
					Tags:     []string{"CREATE TABLE", "DROP TABLE", "ALTER TABLE"},
				},
			},
			want: []string{"CREATE EVENT TRIGGER", "table_audit", "WHEN TAG IN", "'CREATE TABLE'", "'DROP TABLE'", "'ALTER TABLE'"},
		},
		{
			name: "drop event trigger",
			change: &EventTriggerChange{
				ChangeType:   DropEventTrigger,
				EventTrigger: parser.EventTrigger{Name: "old_trigger"},
			},
			want: []string{"DROP EVENT TRIGGER", "old_trigger"},
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

func TestEventTriggerChange_DownSQL(t *testing.T) {
	createChange := &EventTriggerChange{
		ChangeType: CreateEventTrigger,
		EventTrigger: parser.EventTrigger{
			Name:     "my_trigger",
			Event:    "ddl_command_end",
			Function: "my_func",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP EVENT TRIGGER") {
		t.Error("DownSQL for CreateEventTrigger should contain DROP EVENT TRIGGER")
	}

	oldET := parser.EventTrigger{Name: "old_trigger", Event: "ddl_command_end", Function: "audit_func", Tags: []string{"CREATE TABLE"}}
	dropChange := &EventTriggerChange{
		ChangeType:      DropEventTrigger,
		EventTrigger:    parser.EventTrigger{Name: "old_trigger"},
		OldEventTrigger: &oldET,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE EVENT TRIGGER") {
		t.Error("DownSQL for DropEventTrigger with OldEventTrigger should contain CREATE EVENT TRIGGER")
	}
	if !strings.Contains(downSQL, "WHEN TAG IN") {
		t.Error("DownSQL for DropEventTrigger should preserve tags")
	}

	dropChangeNoOld := &EventTriggerChange{
		ChangeType:   DropEventTrigger,
		EventTrigger: parser.EventTrigger{Name: "old_trigger"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropEventTrigger without OldEventTrigger should indicate IRREVERSIBLE")
	}
}

func TestEventTriggerChange_IsReversible(t *testing.T) {
	createChange := &EventTriggerChange{
		ChangeType: CreateEventTrigger,
		EventTrigger: parser.EventTrigger{
			Name:     "test",
			Event:    "ddl_command_end",
			Function: "test_func",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateEventTrigger should be reversible")
	}

	oldET := parser.EventTrigger{Name: "test", Event: "ddl_command_end", Function: "test_func"}
	dropChangeWithOld := &EventTriggerChange{
		ChangeType:      DropEventTrigger,
		EventTrigger:    parser.EventTrigger{Name: "test"},
		OldEventTrigger: &oldET,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropEventTrigger with OldEventTrigger should be reversible")
	}

	dropChangeNoOld := &EventTriggerChange{
		ChangeType:   DropEventTrigger,
		EventTrigger: parser.EventTrigger{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropEventTrigger without OldEventTrigger should not be reversible")
	}
}
