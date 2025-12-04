package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateSequence(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Sequences: []parser.Sequence{
			{Name: "user_id_seq", Start: 1, Increment: 1},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateSequence && c.ObjectName() == "user_id_seq" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateSequence change for user_id_seq")
	}
}

func TestCompare_DropSequence(t *testing.T) {
	current := &parser.Schema{
		Sequences: []parser.Sequence{
			{Name: "old_seq"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropSequence && c.ObjectName() == "old_seq" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropSequence change for old_seq")
	}
}

func TestSequenceChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *SequenceChange
		want   []string
	}{
		{
			name: "create sequence basic",
			change: &SequenceChange{
				ChangeType: CreateSequence,
				Sequence:   parser.Sequence{Name: "my_seq"},
			},
			want: []string{"CREATE SEQUENCE", "my_seq"},
		},
		{
			name: "create sequence with options",
			change: &SequenceChange{
				ChangeType: CreateSequence,
				Sequence: parser.Sequence{
					Name:      "counter_seq",
					Start:     100,
					Increment: 5,
					MinValue:  1,
					MaxValue:  10000,
					Cache:     10,
					Cycle:     true,
				},
			},
			want: []string{"CREATE SEQUENCE", "counter_seq", "START 100", "INCREMENT 5", "MINVALUE 1", "MAXVALUE 10000", "CACHE 10", "CYCLE"},
		},
		{
			name: "create sequence with schema",
			change: &SequenceChange{
				ChangeType: CreateSequence,
				Sequence:   parser.Sequence{Schema: "myschema", Name: "my_seq"},
			},
			want: []string{"CREATE SEQUENCE", "myschema", "my_seq"},
		},
		{
			name: "drop sequence",
			change: &SequenceChange{
				ChangeType: DropSequence,
				Sequence:   parser.Sequence{Name: "old_seq"},
			},
			want: []string{"DROP SEQUENCE", "old_seq"},
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

func TestSequenceChange_DownSQL(t *testing.T) {
	createChange := &SequenceChange{
		ChangeType: CreateSequence,
		Sequence:   parser.Sequence{Name: "my_seq"},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP SEQUENCE") {
		t.Error("DownSQL for CreateSequence should contain DROP SEQUENCE")
	}

	oldSeq := parser.Sequence{Name: "old_seq", Start: 1, Increment: 1}
	dropChange := &SequenceChange{
		ChangeType:  DropSequence,
		Sequence:    parser.Sequence{Name: "old_seq"},
		OldSequence: &oldSeq,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE SEQUENCE") {
		t.Error("DownSQL for DropSequence with OldSequence should contain CREATE SEQUENCE")
	}

	dropChangeNoOld := &SequenceChange{
		ChangeType: DropSequence,
		Sequence:   parser.Sequence{Name: "old_seq"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropSequence without OldSequence should indicate IRREVERSIBLE")
	}
}

func TestSequenceChange_IsReversible(t *testing.T) {
	createChange := &SequenceChange{
		ChangeType: CreateSequence,
		Sequence:   parser.Sequence{Name: "test"},
	}
	if !createChange.IsReversible() {
		t.Error("CreateSequence should be reversible")
	}

	oldSeq := parser.Sequence{Name: "test"}
	dropChangeWithOld := &SequenceChange{
		ChangeType:  DropSequence,
		Sequence:    parser.Sequence{Name: "test"},
		OldSequence: &oldSeq,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropSequence with OldSequence should be reversible")
	}

	dropChangeNoOld := &SequenceChange{
		ChangeType: DropSequence,
		Sequence:   parser.Sequence{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropSequence without OldSequence should not be reversible")
	}
}
