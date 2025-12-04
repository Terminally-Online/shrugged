package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateMaterializedView(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		MaterializedViews: []parser.MaterializedView{
			{Name: "sales_summary", Definition: "SELECT date, SUM(amount) FROM sales GROUP BY date", WithData: true},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateMaterializedView && c.ObjectName() == "sales_summary" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateMaterializedView change for sales_summary")
	}
}

func TestCompare_DropMaterializedView(t *testing.T) {
	current := &parser.Schema{
		MaterializedViews: []parser.MaterializedView{
			{Name: "old_mv", Definition: "SELECT 1"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropMaterializedView && c.ObjectName() == "old_mv" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropMaterializedView change for old_mv")
	}
}

func TestCompare_AlterMaterializedView(t *testing.T) {
	current := &parser.Schema{
		MaterializedViews: []parser.MaterializedView{
			{Name: "my_mv", Definition: "SELECT id FROM users"},
		},
	}
	desired := &parser.Schema{
		MaterializedViews: []parser.MaterializedView{
			{Name: "my_mv", Definition: "SELECT id, name FROM users"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterMaterializedView && c.ObjectName() == "my_mv" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected AlterMaterializedView change for my_mv")
	}
}

func TestMaterializedViewChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *MaterializedViewChange
		want   []string
	}{
		{
			name: "create materialized view",
			change: &MaterializedViewChange{
				ChangeType: CreateMaterializedView,
				MaterializedView: parser.MaterializedView{
					Name:       "my_mv",
					Definition: "SELECT * FROM users",
					WithData:   true,
				},
			},
			want: []string{"CREATE MATERIALIZED VIEW", "my_mv", "AS", "SELECT * FROM users"},
		},
		{
			name: "create materialized view without data",
			change: &MaterializedViewChange{
				ChangeType: CreateMaterializedView,
				MaterializedView: parser.MaterializedView{
					Name:       "lazy_mv",
					Definition: "SELECT * FROM big_table",
					WithData:   false,
				},
			},
			want: []string{"CREATE MATERIALIZED VIEW", "WITH NO DATA"},
		},
		{
			name: "create materialized view with schema",
			change: &MaterializedViewChange{
				ChangeType: CreateMaterializedView,
				MaterializedView: parser.MaterializedView{
					Schema:     "analytics",
					Name:       "report_mv",
					Definition: "SELECT 1",
					WithData:   true,
				},
			},
			want: []string{"CREATE MATERIALIZED VIEW", "analytics", "report_mv"},
		},
		{
			name: "drop materialized view",
			change: &MaterializedViewChange{
				ChangeType:       DropMaterializedView,
				MaterializedView: parser.MaterializedView{Name: "old_mv"},
			},
			want: []string{"DROP MATERIALIZED VIEW", "old_mv"},
		},
		{
			name: "alter materialized view",
			change: &MaterializedViewChange{
				ChangeType: AlterMaterializedView,
				MaterializedView: parser.MaterializedView{
					Name:       "my_mv",
					Definition: "SELECT * FROM users WHERE active",
					WithData:   true,
				},
			},
			want: []string{"DROP MATERIALIZED VIEW", "CREATE MATERIALIZED VIEW", "my_mv"},
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

func TestMaterializedViewChange_DownSQL(t *testing.T) {
	createChange := &MaterializedViewChange{
		ChangeType: CreateMaterializedView,
		MaterializedView: parser.MaterializedView{
			Name:       "my_mv",
			Definition: "SELECT 1",
			WithData:   true,
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP MATERIALIZED VIEW") {
		t.Error("DownSQL for CreateMaterializedView should contain DROP MATERIALIZED VIEW")
	}

	oldMV := parser.MaterializedView{Name: "old_mv", Definition: "SELECT 1", WithData: true}
	dropChange := &MaterializedViewChange{
		ChangeType:          DropMaterializedView,
		MaterializedView:    parser.MaterializedView{Name: "old_mv"},
		OldMaterializedView: &oldMV,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE MATERIALIZED VIEW") {
		t.Error("DownSQL for DropMaterializedView with OldMaterializedView should contain CREATE MATERIALIZED VIEW")
	}

	dropChangeNoOld := &MaterializedViewChange{
		ChangeType:       DropMaterializedView,
		MaterializedView: parser.MaterializedView{Name: "old_mv"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropMaterializedView without OldMaterializedView should indicate IRREVERSIBLE")
	}

	oldMVForAlter := parser.MaterializedView{Name: "my_mv", Definition: "SELECT 1", WithData: true}
	alterChange := &MaterializedViewChange{
		ChangeType: AlterMaterializedView,
		MaterializedView: parser.MaterializedView{
			Name:       "my_mv",
			Definition: "SELECT 2",
			WithData:   true,
		},
		OldMaterializedView: &oldMVForAlter,
	}
	alterDownSQL := alterChange.DownSQL()
	if !strings.Contains(alterDownSQL, "DROP MATERIALIZED VIEW") || !strings.Contains(alterDownSQL, "CREATE MATERIALIZED VIEW") {
		t.Error("DownSQL for AlterMaterializedView should contain both DROP and CREATE")
	}
}

func TestMaterializedViewChange_IsReversible(t *testing.T) {
	createChange := &MaterializedViewChange{
		ChangeType: CreateMaterializedView,
		MaterializedView: parser.MaterializedView{
			Name:       "test",
			Definition: "SELECT 1",
			WithData:   true,
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateMaterializedView should be reversible")
	}

	oldMV := parser.MaterializedView{Name: "test", Definition: "SELECT 1", WithData: true}
	dropChangeWithOld := &MaterializedViewChange{
		ChangeType:          DropMaterializedView,
		MaterializedView:    parser.MaterializedView{Name: "test"},
		OldMaterializedView: &oldMV,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropMaterializedView with OldMaterializedView should be reversible")
	}

	dropChangeNoOld := &MaterializedViewChange{
		ChangeType:       DropMaterializedView,
		MaterializedView: parser.MaterializedView{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropMaterializedView without OldMaterializedView should not be reversible")
	}
}
