package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateOperator(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Operators: []parser.Operator{
			{Name: "===", LeftType: "integer", RightType: "integer", Procedure: "int4eq"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateOperator && c.ObjectName() == "===" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateOperator change for ===")
	}
}

func TestCompare_DropOperator(t *testing.T) {
	current := &parser.Schema{
		Operators: []parser.Operator{
			{Name: "###", LeftType: "text", RightType: "text", Procedure: "text_cmp"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropOperator && c.ObjectName() == "###" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropOperator change for ###")
	}
}

func TestOperatorChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *OperatorChange
		want   []string
	}{
		{
			name: "create operator basic",
			change: &OperatorChange{
				ChangeType: CreateOperator,
				Operator: parser.Operator{
					Name:      "===",
					LeftType:  "integer",
					RightType: "integer",
					Procedure: "int4eq",
				},
			},
			want: []string{"CREATE OPERATOR", "===", "FUNCTION = int4eq", "LEFTARG = integer", "RIGHTARG = integer"},
		},
		{
			name: "create operator with commutator and negator",
			change: &OperatorChange{
				ChangeType: CreateOperator,
				Operator: parser.Operator{
					Name:       "<=>",
					LeftType:   "text",
					RightType:  "text",
					Procedure:  "text_cmp",
					Commutator: "<=>",
					Negator:    "<!=>",
				},
			},
			want: []string{"CREATE OPERATOR", "<=>", "COMMUTATOR = <=>", "NEGATOR = <!=>"},
		},
		{
			name: "create prefix operator",
			change: &OperatorChange{
				ChangeType: CreateOperator,
				Operator: parser.Operator{
					Name:      "!!",
					LeftType:  "NONE",
					RightType: "integer",
					Procedure: "factorial",
				},
			},
			want: []string{"CREATE OPERATOR", "!!", "RIGHTARG = integer"},
		},
		{
			name: "drop operator",
			change: &OperatorChange{
				ChangeType: DropOperator,
				Operator: parser.Operator{
					Name:      "===",
					LeftType:  "integer",
					RightType: "integer",
				},
			},
			want: []string{"DROP OPERATOR", "===", "integer", "integer"},
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

func TestOperatorChange_DownSQL(t *testing.T) {
	createChange := &OperatorChange{
		ChangeType: CreateOperator,
		Operator: parser.Operator{
			Name:      "===",
			LeftType:  "integer",
			RightType: "integer",
			Procedure: "int4eq",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP OPERATOR") {
		t.Error("DownSQL for CreateOperator should contain DROP OPERATOR")
	}

	oldOp := parser.Operator{Name: "###", LeftType: "text", RightType: "text", Procedure: "text_cmp", Commutator: "###"}
	dropChange := &OperatorChange{
		ChangeType:  DropOperator,
		Operator:    parser.Operator{Name: "###", LeftType: "text", RightType: "text"},
		OldOperator: &oldOp,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE OPERATOR") {
		t.Error("DownSQL for DropOperator with OldOperator should contain CREATE OPERATOR")
	}
	if !strings.Contains(downSQL, "COMMUTATOR") {
		t.Error("DownSQL for DropOperator should preserve COMMUTATOR")
	}

	dropChangeNoOld := &OperatorChange{
		ChangeType: DropOperator,
		Operator:   parser.Operator{Name: "###", LeftType: "text", RightType: "text"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropOperator without OldOperator should indicate IRREVERSIBLE")
	}
}

func TestOperatorChange_IsReversible(t *testing.T) {
	createChange := &OperatorChange{
		ChangeType: CreateOperator,
		Operator: parser.Operator{
			Name:      "===",
			LeftType:  "integer",
			RightType: "integer",
			Procedure: "int4eq",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateOperator should be reversible")
	}

	oldOp := parser.Operator{Name: "===", LeftType: "integer", RightType: "integer", Procedure: "int4eq"}
	dropChangeWithOld := &OperatorChange{
		ChangeType:  DropOperator,
		Operator:    parser.Operator{Name: "===", LeftType: "integer", RightType: "integer"},
		OldOperator: &oldOp,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropOperator with OldOperator should be reversible")
	}

	dropChangeNoOld := &OperatorChange{
		ChangeType: DropOperator,
		Operator:   parser.Operator{Name: "===", LeftType: "integer", RightType: "integer"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropOperator without OldOperator should not be reversible")
	}
}
