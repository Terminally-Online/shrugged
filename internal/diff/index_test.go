package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateIndex(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Indexes: []parser.Index{
			{Name: "idx_users_email", Table: "users", Columns: []string{"email"}},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateIndex && c.ObjectName() == "idx_users_email" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateIndex change for idx_users_email")
	}
}

func TestCompare_DropIndex(t *testing.T) {
	current := &parser.Schema{
		Indexes: []parser.Index{
			{Name: "old_idx", Table: "users", Columns: []string{"id"}},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropIndex && c.ObjectName() == "old_idx" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropIndex change for old_idx")
	}
}

func TestIndexChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *IndexChange
		want   []string
	}{
		{
			name: "create simple index",
			change: &IndexChange{
				ChangeType: CreateIndex,
				Index: parser.Index{
					Name:    "idx_users_email",
					Table:   "users",
					Columns: []string{"email"},
				},
			},
			want: []string{"CREATE INDEX", "idx_users_email", "ON", "users", "(email)"},
		},
		{
			name: "create unique index",
			change: &IndexChange{
				ChangeType: CreateIndex,
				Index: parser.Index{
					Name:    "idx_users_email_unique",
					Table:   "users",
					Columns: []string{"email"},
					Unique:  true,
				},
			},
			want: []string{"CREATE UNIQUE INDEX", "idx_users_email_unique"},
		},
		{
			name: "create index with multiple columns",
			change: &IndexChange{
				ChangeType: CreateIndex,
				Index: parser.Index{
					Name:    "idx_orders_composite",
					Table:   "orders",
					Columns: []string{"user_id", "created_at"},
				},
			},
			want: []string{"CREATE INDEX", "(user_id, created_at)"},
		},
		{
			name: "create partial index",
			change: &IndexChange{
				ChangeType: CreateIndex,
				Index: parser.Index{
					Name:    "idx_users_active",
					Table:   "users",
					Columns: []string{"email"},
					Where:   "active = true",
				},
			},
			want: []string{"CREATE INDEX", "WHERE active = true"},
		},
		{
			name: "create index with using clause",
			change: &IndexChange{
				ChangeType: CreateIndex,
				Index: parser.Index{
					Name:    "idx_users_hash",
					Table:   "users",
					Columns: []string{"id"},
					Using:   "hash",
				},
			},
			want: []string{"CREATE INDEX", "USING hash"},
		},
		{
			name: "create index with schema",
			change: &IndexChange{
				ChangeType: CreateIndex,
				Index: parser.Index{
					Schema:  "myschema",
					Name:    "idx_data",
					Table:   "data",
					Columns: []string{"value"},
				},
			},
			want: []string{"CREATE INDEX", "idx_data", "ON", "myschema", "data"},
		},
		{
			name: "drop index",
			change: &IndexChange{
				ChangeType: DropIndex,
				Index:      parser.Index{Name: "old_idx"},
			},
			want: []string{"DROP INDEX", "old_idx"},
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

func TestIndexChange_DownSQL(t *testing.T) {
	createChange := &IndexChange{
		ChangeType: CreateIndex,
		Index: parser.Index{
			Name:    "idx_test",
			Table:   "test",
			Columns: []string{"col"},
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP INDEX") {
		t.Error("DownSQL for CreateIndex should contain DROP INDEX")
	}

	oldIdx := parser.Index{Name: "old_idx", Table: "users", Columns: []string{"email"}, Unique: true}
	dropChange := &IndexChange{
		ChangeType: DropIndex,
		Index:      parser.Index{Name: "old_idx"},
		OldIndex:   &oldIdx,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE UNIQUE INDEX") {
		t.Error("DownSQL for DropIndex with OldIndex should contain CREATE UNIQUE INDEX")
	}
	if !strings.Contains(downSQL, "users") {
		t.Error("DownSQL for DropIndex should preserve table name")
	}

	dropChangeNoOld := &IndexChange{
		ChangeType: DropIndex,
		Index:      parser.Index{Name: "old_idx"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropIndex without OldIndex should indicate IRREVERSIBLE")
	}
}

func TestIndexChange_IsReversible(t *testing.T) {
	createChange := &IndexChange{
		ChangeType: CreateIndex,
		Index: parser.Index{
			Name:    "test",
			Table:   "test",
			Columns: []string{"col"},
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateIndex should be reversible")
	}

	oldIdx := parser.Index{Name: "test", Table: "test", Columns: []string{"col"}}
	dropChangeWithOld := &IndexChange{
		ChangeType: DropIndex,
		Index:      parser.Index{Name: "test"},
		OldIndex:   &oldIdx,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropIndex with OldIndex should be reversible")
	}

	dropChangeNoOld := &IndexChange{
		ChangeType: DropIndex,
		Index:      parser.Index{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropIndex without OldIndex should not be reversible")
	}
}

func TestGenerateCreateIndex(t *testing.T) {
	tests := []struct {
		name  string
		index parser.Index
		want  []string
	}{
		{
			name: "basic index",
			index: parser.Index{
				Name:    "idx_test",
				Table:   "test",
				Columns: []string{"col"},
			},
			want: []string{"CREATE INDEX", "idx_test", "ON", "test", "(col)"},
		},
		{
			name: "btree index does not show USING",
			index: parser.Index{
				Name:    "idx_test",
				Table:   "test",
				Columns: []string{"col"},
				Using:   "btree",
			},
			want: []string{"CREATE INDEX"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := generateCreateIndex(tt.index)
			for _, want := range tt.want {
				if !strings.Contains(sql, want) {
					t.Errorf("SQL = %q, should contain %q", sql, want)
				}
			}
		})
	}

	btreeIdx := parser.Index{Name: "idx", Table: "t", Columns: []string{"c"}, Using: "btree"}
	sql := generateCreateIndex(btreeIdx)
	if strings.Contains(sql, "USING btree") {
		t.Error("btree index should not include USING btree (it's the default)")
	}
}
