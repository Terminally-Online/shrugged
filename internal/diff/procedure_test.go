package diff

import (
	"strings"
	"testing"

	"git.ca.plug.to/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateProcedure(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "p_id integer", Language: "plpgsql", Body: "BEGIN RAISE NOTICE 'Hello'; END;"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateProcedure && c.ObjectName() == "my_proc" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateProcedure change for my_proc")
	}
}

func TestCompare_DropProcedure(t *testing.T) {
	current := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "old_proc", Args: "", Language: "sql", Body: "SELECT 1"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropProcedure && c.ObjectName() == "old_proc" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropProcedure change for old_proc")
	}
}

func TestCompare_AlterProcedure(t *testing.T) {
	current := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "p_id integer", Language: "plpgsql", Body: "BEGIN SELECT 1; END;"},
		},
	}
	desired := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "p_id integer", Language: "plpgsql", Body: "BEGIN SELECT 2; END;"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterProcedure && c.ObjectName() == "my_proc" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected AlterProcedure change for my_proc")
	}
}

func TestProcedureChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *ProcedureChange
		want   []string
	}{
		{
			name: "create procedure",
			change: &ProcedureChange{
				ChangeType: CreateProcedure,
				Procedure: parser.Procedure{
					Name:     "my_proc",
					Args:     "p_id integer",
					Language: "plpgsql",
					Body:     "BEGIN RAISE NOTICE 'Hello'; END;",
				},
			},
			want: []string{"CREATE OR REPLACE PROCEDURE", "my_proc", "p_id integer", "LANGUAGE", "plpgsql"},
		},
		{
			name: "create procedure with schema",
			change: &ProcedureChange{
				ChangeType: CreateProcedure,
				Procedure: parser.Procedure{
					Schema:   "myschema",
					Name:     "my_proc",
					Args:     "",
					Language: "sql",
					Body:     "SELECT 1",
				},
			},
			want: []string{"CREATE OR REPLACE PROCEDURE", "myschema", "my_proc"},
		},
		{
			name: "drop procedure with args",
			change: &ProcedureChange{
				ChangeType: DropProcedure,
				Procedure: parser.Procedure{
					Name: "old_proc",
					Args: "p_id integer",
				},
			},
			want: []string{"DROP PROCEDURE", "old_proc", "(integer)"},
		},
		{
			name: "drop procedure without args",
			change: &ProcedureChange{
				ChangeType: DropProcedure,
				Procedure:  parser.Procedure{Name: "simple_proc"},
			},
			want: []string{"DROP PROCEDURE", "simple_proc"},
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

func TestProcedureChange_DownSQL(t *testing.T) {
	createChange := &ProcedureChange{
		ChangeType: CreateProcedure,
		Procedure: parser.Procedure{
			Name:     "my_proc",
			Args:     "p_id integer",
			Language: "plpgsql",
			Body:     "BEGIN SELECT 1; END;",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP PROCEDURE") {
		t.Error("DownSQL for CreateProcedure should contain DROP PROCEDURE")
	}

	oldProc := parser.Procedure{Name: "old_proc", Args: "p_id integer", Language: "plpgsql", Body: "BEGIN SELECT 1; END;"}
	dropChange := &ProcedureChange{
		ChangeType:   DropProcedure,
		Procedure:    parser.Procedure{Name: "old_proc", Args: "p_id integer"},
		OldProcedure: &oldProc,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE OR REPLACE PROCEDURE") {
		t.Error("DownSQL for DropProcedure with OldProcedure should contain CREATE OR REPLACE PROCEDURE")
	}

	dropChangeNoOld := &ProcedureChange{
		ChangeType: DropProcedure,
		Procedure:  parser.Procedure{Name: "old_proc"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropProcedure without OldProcedure should indicate IRREVERSIBLE")
	}
}

func TestProcedureChange_IsReversible(t *testing.T) {
	createChange := &ProcedureChange{
		ChangeType: CreateProcedure,
		Procedure: parser.Procedure{
			Name:     "test",
			Args:     "",
			Language: "sql",
			Body:     "SELECT 1",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateProcedure should be reversible")
	}

	oldProc := parser.Procedure{Name: "test", Args: "", Language: "sql", Body: "SELECT 1"}
	dropChangeWithOld := &ProcedureChange{
		ChangeType:   DropProcedure,
		Procedure:    parser.Procedure{Name: "test"},
		OldProcedure: &oldProc,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropProcedure with OldProcedure should be reversible")
	}

	dropChangeNoOld := &ProcedureChange{
		ChangeType: DropProcedure,
		Procedure:  parser.Procedure{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropProcedure without OldProcedure should not be reversible")
	}
}

func TestCompare_ProcedureParameterRenameNoChange(t *testing.T) {
	current := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "p_x integer", Language: "plpgsql", Body: "BEGIN NULL; END;"},
		},
	}
	desired := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "x integer", Language: "plpgsql", Body: "BEGIN NULL; END;"},
		},
	}

	changes := Compare(current, desired)

	for _, c := range changes {
		if c.ObjectName() == "my_proc" {
			t.Errorf("parameter rename should not produce a change, got %v", c.Type())
		}
	}
}

func TestCompare_ProcedureDefaultAddNoChange(t *testing.T) {
	current := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "x integer", Language: "plpgsql", Body: "BEGIN NULL; END;"},
		},
	}
	desired := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "x integer DEFAULT 0", Language: "plpgsql", Body: "BEGIN NULL; END;"},
		},
	}

	changes := Compare(current, desired)

	for _, c := range changes {
		if c.ObjectName() == "my_proc" {
			t.Errorf("DEFAULT-only change should not be treated as overload change, got %v", c.Type())
		}
	}
}

func TestCompare_ProcedureBodyWhitespaceOnly(t *testing.T) {
	current := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "", Language: "plpgsql", Body: "BEGIN NULL; END;"},
		},
	}
	desired := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "", Language: "plpgsql", Body: "BEGIN\n  NULL;\nEND;\n"},
		},
	}

	changes := Compare(current, desired)

	for _, c := range changes {
		if c.Type() == AlterProcedure && c.ObjectName() == "my_proc" {
			t.Error("whitespace-only body difference should not produce AlterProcedure")
		}
	}
}

func TestCompare_ProcedureBodyCommentOnly(t *testing.T) {
	current := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "", Language: "plpgsql", Body: "BEGIN NULL; END;"},
		},
	}
	desired := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "", Language: "plpgsql", Body: "BEGIN -- noop\nNULL; /* still nothing */ END;"},
		},
	}

	changes := Compare(current, desired)

	for _, c := range changes {
		if c.Type() == AlterProcedure && c.ObjectName() == "my_proc" {
			t.Error("comment-only body difference should not produce AlterProcedure")
		}
	}
}

func TestCompare_ProcedureBodySemanticChange(t *testing.T) {
	current := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "", Language: "plpgsql", Body: "BEGIN INSERT INTO t VALUES (1); END;"},
		},
	}
	desired := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "my_proc", Args: "", Language: "plpgsql", Body: "BEGIN INSERT INTO t VALUES (2); END;"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterProcedure && c.ObjectName() == "my_proc" {
			found = true
		}
	}
	if !found {
		t.Error("genuine body change should still produce AlterProcedure")
	}
}

func TestCompare_ProcedureOverloads_DistinctSignatures(t *testing.T) {
	current := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "p", Args: "x integer", Language: "plpgsql", Body: "BEGIN NULL; END;"},
			{Name: "p", Args: "x integer, y text", Language: "plpgsql", Body: "BEGIN NULL; END;"},
		},
	}
	desired := &parser.Schema{
		Procedures: []parser.Procedure{
			{Name: "p", Args: "x integer", Language: "plpgsql", Body: "BEGIN NULL; END;"},
			{Name: "p", Args: "x integer, y text", Language: "plpgsql", Body: "BEGIN NULL; END;"},
		},
	}

	changes := Compare(current, desired)

	for _, c := range changes {
		if c.ObjectName() == "p" {
			t.Errorf("matched overloads should produce no change, got %v", c.Type())
		}
	}
}

func TestProcedureChange_DropIncludesNormalizedSignature(t *testing.T) {
	change := &ProcedureChange{
		ChangeType: DropProcedure,
		Procedure: parser.Procedure{
			Schema: "public",
			Name:   "p",
			Args:   "p_x integer, p_y text DEFAULT ''",
		},
	}

	sql := change.SQL()
	want := "DROP PROCEDURE p(integer, text);"
	if sql != want {
		t.Errorf("DROP PROCEDURE SQL = %q, want %q", sql, want)
	}
}
