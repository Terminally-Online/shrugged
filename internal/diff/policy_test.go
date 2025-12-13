package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreatePolicy(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Policies: []parser.Policy{
			{Name: "user_isolation", Table: "data", Command: "ALL", Permissive: true, Using: "user_id = current_user_id()"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreatePolicy && c.ObjectName() == "user_isolation" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreatePolicy change for user_isolation")
	}
}

func TestCompare_DropPolicy(t *testing.T) {
	current := &parser.Schema{
		Policies: []parser.Policy{
			{Name: "old_policy", Table: "users"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropPolicy && c.ObjectName() == "old_policy" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropPolicy change for old_policy")
	}
}

func TestPolicyChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *PolicyChange
		want   []string
	}{
		{
			name: "create policy basic",
			change: &PolicyChange{
				ChangeType: CreatePolicy,
				Policy: parser.Policy{
					Name:       "my_policy",
					Table:      "users",
					Command:    "ALL",
					Permissive: true,
				},
			},
			want: []string{"CREATE POLICY", "my_policy", "ON", "users"},
		},
		{
			name: "create policy with all options",
			change: &PolicyChange{
				ChangeType: CreatePolicy,
				Policy: parser.Policy{
					Name:       "rls_policy",
					Table:      "data",
					Command:    "SELECT",
					Permissive: false,
					Roles:      []string{"app_user", "admin"},
					Using:      "tenant_id = current_tenant()",
					WithCheck:  "tenant_id = current_tenant()",
				},
			},
			want: []string{"CREATE POLICY", "rls_policy", "AS RESTRICTIVE", "FOR SELECT", "TO app_user, admin", "USING", "WITH CHECK"},
		},
		{
			name: "create policy with schema",
			change: &PolicyChange{
				ChangeType: CreatePolicy,
				Policy: parser.Policy{
					Schema:     "myschema",
					Name:       "my_policy",
					Table:      "data",
					Command:    "ALL",
					Permissive: true,
				},
			},
			want: []string{"CREATE POLICY", "my_policy", "myschema", "data"},
		},
		{
			name: "drop policy",
			change: &PolicyChange{
				ChangeType: DropPolicy,
				Policy:     parser.Policy{Name: "old_policy", Table: "users"},
			},
			want: []string{"DROP POLICY", "old_policy", "ON", "users"},
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

func TestPolicyChange_DownSQL(t *testing.T) {
	createChange := &PolicyChange{
		ChangeType: CreatePolicy,
		Policy: parser.Policy{
			Name:       "my_policy",
			Table:      "users",
			Command:    "ALL",
			Permissive: true,
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP POLICY") {
		t.Error("DownSQL for CreatePolicy should contain DROP POLICY")
	}

	oldPolicy := parser.Policy{
		Name:       "old_policy",
		Table:      "users",
		Command:    "SELECT",
		Permissive: false,
		Roles:      []string{"app_user"},
		Using:      "user_id = current_user()",
	}
	dropChange := &PolicyChange{
		ChangeType: DropPolicy,
		Policy:     parser.Policy{Name: "old_policy", Table: "users"},
		OldPolicy:  &oldPolicy,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE POLICY") {
		t.Error("DownSQL for DropPolicy with OldPolicy should contain CREATE POLICY")
	}
	if !strings.Contains(downSQL, "AS RESTRICTIVE") {
		t.Error("DownSQL for DropPolicy should preserve RESTRICTIVE")
	}

	dropChangeNoOld := &PolicyChange{
		ChangeType: DropPolicy,
		Policy:     parser.Policy{Name: "old_policy", Table: "users"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropPolicy without OldPolicy should indicate IRREVERSIBLE")
	}
}

func TestPolicyChange_IsReversible(t *testing.T) {
	createChange := &PolicyChange{
		ChangeType: CreatePolicy,
		Policy: parser.Policy{
			Name:       "test",
			Table:      "users",
			Command:    "ALL",
			Permissive: true,
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreatePolicy should be reversible")
	}

	oldPolicy := parser.Policy{Name: "test", Table: "users", Command: "ALL", Permissive: true}
	dropChangeWithOld := &PolicyChange{
		ChangeType: DropPolicy,
		Policy:     parser.Policy{Name: "test", Table: "users"},
		OldPolicy:  &oldPolicy,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropPolicy with OldPolicy should be reversible")
	}

	dropChangeNoOld := &PolicyChange{
		ChangeType: DropPolicy,
		Policy:     parser.Policy{Name: "test", Table: "users"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropPolicy without OldPolicy should not be reversible")
	}
}
