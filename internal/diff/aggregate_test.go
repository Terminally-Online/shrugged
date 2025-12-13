package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateAggregate(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Aggregates: []parser.Aggregate{
			{Name: "my_sum", Args: "integer", SFunc: "int4pl", SType: "integer"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateAggregate && c.ObjectName() == "my_sum" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateAggregate change for my_sum")
	}
}

func TestCompare_DropAggregate(t *testing.T) {
	current := &parser.Schema{
		Aggregates: []parser.Aggregate{
			{Name: "old_agg", Args: "integer", SFunc: "int4pl", SType: "integer"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropAggregate && c.ObjectName() == "old_agg" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropAggregate change for old_agg")
	}
}

func TestAggregateChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *AggregateChange
		want   []string
	}{
		{
			name: "create aggregate basic",
			change: &AggregateChange{
				ChangeType: CreateAggregate,
				Aggregate: parser.Aggregate{
					Name:  "my_sum",
					Args:  "integer",
					SFunc: "int4pl",
					SType: "integer",
				},
			},
			want: []string{"CREATE AGGREGATE", "my_sum", "integer", "SFUNC = int4pl", "STYPE = integer"},
		},
		{
			name: "create aggregate with all options",
			change: &AggregateChange{
				ChangeType: CreateAggregate,
				Aggregate: parser.Aggregate{
					Name:      "complex_agg",
					Args:      "integer",
					SFunc:     "my_sfunc",
					SType:     "integer[]",
					FinalFunc: "my_finalfunc",
					InitCond:  "{}",
					SortOp:    "<",
				},
			},
			want: []string{"CREATE AGGREGATE", "complex_agg", "SFUNC = my_sfunc", "STYPE = integer[]", "FINALFUNC = my_finalfunc", "INITCOND = '{}'", "SORTOP = <"},
		},
		{
			name: "drop aggregate",
			change: &AggregateChange{
				ChangeType: DropAggregate,
				Aggregate:  parser.Aggregate{Name: "old_agg", Args: "integer"},
			},
			want: []string{"DROP AGGREGATE", "old_agg", "integer"},
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

func TestAggregateChange_DownSQL(t *testing.T) {
	createChange := &AggregateChange{
		ChangeType: CreateAggregate,
		Aggregate: parser.Aggregate{
			Name:  "my_agg",
			Args:  "integer",
			SFunc: "int4pl",
			SType: "integer",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP AGGREGATE") {
		t.Error("DownSQL for CreateAggregate should contain DROP AGGREGATE")
	}

	oldAgg := parser.Aggregate{Name: "old_agg", Args: "integer", SFunc: "int4pl", SType: "integer", FinalFunc: "my_final"}
	dropChange := &AggregateChange{
		ChangeType:   DropAggregate,
		Aggregate:    parser.Aggregate{Name: "old_agg", Args: "integer"},
		OldAggregate: &oldAgg,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE AGGREGATE") {
		t.Error("DownSQL for DropAggregate with OldAggregate should contain CREATE AGGREGATE")
	}
	if !strings.Contains(downSQL, "FINALFUNC") {
		t.Error("DownSQL for DropAggregate should preserve FINALFUNC")
	}

	dropChangeNoOld := &AggregateChange{
		ChangeType: DropAggregate,
		Aggregate:  parser.Aggregate{Name: "old_agg", Args: "integer"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropAggregate without OldAggregate should indicate IRREVERSIBLE")
	}
}

func TestAggregateChange_IsReversible(t *testing.T) {
	createChange := &AggregateChange{
		ChangeType: CreateAggregate,
		Aggregate: parser.Aggregate{
			Name:  "test",
			Args:  "integer",
			SFunc: "int4pl",
			SType: "integer",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateAggregate should be reversible")
	}

	oldAgg := parser.Aggregate{Name: "test", Args: "integer", SFunc: "int4pl", SType: "integer"}
	dropChangeWithOld := &AggregateChange{
		ChangeType:   DropAggregate,
		Aggregate:    parser.Aggregate{Name: "test", Args: "integer"},
		OldAggregate: &oldAgg,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropAggregate with OldAggregate should be reversible")
	}

	dropChangeNoOld := &AggregateChange{
		ChangeType: DropAggregate,
		Aggregate:  parser.Aggregate{Name: "test", Args: "integer"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropAggregate without OldAggregate should not be reversible")
	}
}
