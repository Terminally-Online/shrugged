package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateSubscription(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Subscriptions: []parser.Subscription{
			{Name: "my_sub", Publication: "my_pub", ConnInfo: "host=replica dbname=mydb", Enabled: true},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateSubscription && c.ObjectName() == "my_sub" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateSubscription change for my_sub")
	}
}

func TestCompare_DropSubscription(t *testing.T) {
	current := &parser.Schema{
		Subscriptions: []parser.Subscription{
			{Name: "old_sub", Publication: "old_pub", ConnInfo: "host=old", Enabled: true},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropSubscription && c.ObjectName() == "old_sub" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropSubscription change for old_sub")
	}
}

func TestSubscriptionChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *SubscriptionChange
		want   []string
	}{
		{
			name: "create subscription enabled",
			change: &SubscriptionChange{
				ChangeType: CreateSubscription,
				Subscription: parser.Subscription{
					Name:        "my_sub",
					Publication: "my_pub",
					ConnInfo:    "host=replica port=5432 dbname=mydb",
					Enabled:     true,
				},
			},
			want: []string{"CREATE SUBSCRIPTION", "my_sub", "CONNECTION", "host=replica port=5432 dbname=mydb", "PUBLICATION", "my_pub"},
		},
		{
			name: "create subscription disabled",
			change: &SubscriptionChange{
				ChangeType: CreateSubscription,
				Subscription: parser.Subscription{
					Name:        "disabled_sub",
					Publication: "some_pub",
					ConnInfo:    "host=standby",
					Enabled:     false,
				},
			},
			want: []string{"CREATE SUBSCRIPTION", "disabled_sub", "WITH (enabled = false)"},
		},
		{
			name: "drop subscription",
			change: &SubscriptionChange{
				ChangeType:   DropSubscription,
				Subscription: parser.Subscription{Name: "old_sub"},
			},
			want: []string{"DROP SUBSCRIPTION", "old_sub"},
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

func TestSubscriptionChange_DownSQL(t *testing.T) {
	createChange := &SubscriptionChange{
		ChangeType: CreateSubscription,
		Subscription: parser.Subscription{
			Name:        "my_sub",
			Publication: "my_pub",
			ConnInfo:    "host=replica",
			Enabled:     true,
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP SUBSCRIPTION") {
		t.Error("DownSQL for CreateSubscription should contain DROP SUBSCRIPTION")
	}

	oldSub := parser.Subscription{Name: "old_sub", Publication: "old_pub", ConnInfo: "host=old", Enabled: false}
	dropChange := &SubscriptionChange{
		ChangeType:      DropSubscription,
		Subscription:    parser.Subscription{Name: "old_sub"},
		OldSubscription: &oldSub,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE SUBSCRIPTION") {
		t.Error("DownSQL for DropSubscription with OldSubscription should contain CREATE SUBSCRIPTION")
	}
	if !strings.Contains(downSQL, "enabled = false") {
		t.Error("DownSQL for DropSubscription should preserve enabled = false")
	}

	dropChangeNoOld := &SubscriptionChange{
		ChangeType:   DropSubscription,
		Subscription: parser.Subscription{Name: "old_sub"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropSubscription without OldSubscription should indicate IRREVERSIBLE")
	}
}

func TestSubscriptionChange_IsReversible(t *testing.T) {
	createChange := &SubscriptionChange{
		ChangeType: CreateSubscription,
		Subscription: parser.Subscription{
			Name:        "test",
			Publication: "test_pub",
			ConnInfo:    "host=test",
			Enabled:     true,
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateSubscription should be reversible")
	}

	oldSub := parser.Subscription{Name: "test", Publication: "test_pub", ConnInfo: "host=test", Enabled: true}
	dropChangeWithOld := &SubscriptionChange{
		ChangeType:      DropSubscription,
		Subscription:    parser.Subscription{Name: "test"},
		OldSubscription: &oldSub,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropSubscription with OldSubscription should be reversible")
	}

	dropChangeNoOld := &SubscriptionChange{
		ChangeType:   DropSubscription,
		Subscription: parser.Subscription{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropSubscription without OldSubscription should not be reversible")
	}
}
