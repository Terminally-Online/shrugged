package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateRule(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Rules: []parser.Rule{
			{Name: "protect_delete", Table: "users", Event: "DELETE", DoInstead: true, Definition: "CREATE RULE protect_delete AS ON DELETE TO users DO INSTEAD NOTHING"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateRule && c.ObjectName() == "protect_delete" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateRule change for protect_delete")
	}
}

func TestCompare_DropRule(t *testing.T) {
	current := &parser.Schema{
		Rules: []parser.Rule{
			{Name: "old_rule", Table: "users"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropRule && c.ObjectName() == "old_rule" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropRule change for old_rule")
	}
}

func TestRuleChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *RuleChange
		want   []string
	}{
		{
			name: "create rule",
			change: &RuleChange{
				ChangeType: CreateRule,
				Rule: parser.Rule{
					Name:       "my_rule",
					Table:      "users",
					Definition: "CREATE RULE my_rule AS ON INSERT TO users DO ALSO NOTIFY users_changed",
				},
			},
			want: []string{"CREATE RULE", "my_rule", "ON INSERT TO users"},
		},
		{
			name: "drop rule",
			change: &RuleChange{
				ChangeType: DropRule,
				Rule:       parser.Rule{Name: "old_rule", Table: "users"},
			},
			want: []string{"DROP RULE", "old_rule", "ON", "users"},
		},
		{
			name: "drop rule with schema",
			change: &RuleChange{
				ChangeType: DropRule,
				Rule:       parser.Rule{Name: "old_rule", Schema: "myschema", Table: "users"},
			},
			want: []string{"DROP RULE", "old_rule", "ON", "myschema", "users"},
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

func TestRuleChange_DownSQL(t *testing.T) {
	createChange := &RuleChange{
		ChangeType: CreateRule,
		Rule: parser.Rule{
			Name:       "my_rule",
			Table:      "users",
			Definition: "CREATE RULE my_rule AS ON INSERT TO users DO ALSO NOTIFY users_changed",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP RULE") {
		t.Error("DownSQL for CreateRule should contain DROP RULE")
	}

	oldRule := parser.Rule{Name: "old_rule", Table: "users", Definition: "CREATE RULE old_rule AS ON DELETE TO users DO INSTEAD NOTHING"}
	dropChange := &RuleChange{
		ChangeType: DropRule,
		Rule:       parser.Rule{Name: "old_rule", Table: "users"},
		OldRule:    &oldRule,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE RULE") {
		t.Error("DownSQL for DropRule with OldRule should contain CREATE RULE")
	}

	dropChangeNoOld := &RuleChange{
		ChangeType: DropRule,
		Rule:       parser.Rule{Name: "old_rule", Table: "users"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropRule without OldRule should indicate IRREVERSIBLE")
	}
}

func TestRuleChange_IsReversible(t *testing.T) {
	createChange := &RuleChange{
		ChangeType: CreateRule,
		Rule: parser.Rule{
			Name:       "test",
			Table:      "users",
			Definition: "CREATE RULE test AS ON INSERT TO users DO NOTHING",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateRule should be reversible")
	}

	oldRule := parser.Rule{Name: "test", Table: "users", Definition: "CREATE RULE test AS ON INSERT TO users DO NOTHING"}
	dropChangeWithOld := &RuleChange{
		ChangeType: DropRule,
		Rule:       parser.Rule{Name: "test", Table: "users"},
		OldRule:    &oldRule,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropRule with OldRule should be reversible")
	}

	dropChangeNoOld := &RuleChange{
		ChangeType: DropRule,
		Rule:       parser.Rule{Name: "test", Table: "users"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropRule without OldRule should not be reversible")
	}
}
