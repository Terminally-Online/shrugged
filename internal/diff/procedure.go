package diff

import (
	"fmt"

	"github.com/terminally-online/shrugged/internal/parser"
)

type ProcedureChange struct {
	ChangeType   ChangeType
	Procedure    parser.Procedure
	OldProcedure *parser.Procedure
}

func (c *ProcedureChange) SQL() string {
	switch c.ChangeType {
	case CreateProcedure, AlterProcedure:
		if c.Procedure.Definition != "" {
			return c.Procedure.Definition + ";"
		}
		return fmt.Sprintf("CREATE OR REPLACE PROCEDURE %s(%s) LANGUAGE %s AS $$%s$$;",
			qualifiedName(c.Procedure.Schema, c.Procedure.Name),
			c.Procedure.Args,
			c.Procedure.Language,
			c.Procedure.Body)
	case DropProcedure:
		return dropProcedureSQL(c.Procedure)
	}
	return ""
}

func (c *ProcedureChange) DownSQL() string {
	switch c.ChangeType {
	case CreateProcedure:
		return dropProcedureSQL(c.Procedure)
	case DropProcedure:
		if c.OldProcedure != nil {
			if c.OldProcedure.Definition != "" {
				return c.OldProcedure.Definition + ";"
			}
			return fmt.Sprintf("CREATE OR REPLACE PROCEDURE %s(%s) LANGUAGE %s AS $$%s$$;",
				qualifiedName(c.OldProcedure.Schema, c.OldProcedure.Name),
				c.OldProcedure.Args,
				c.OldProcedure.Language,
				c.OldProcedure.Body)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped procedure %s", c.Procedure.Name)
	case AlterProcedure:
		if c.OldProcedure != nil {
			if c.OldProcedure.Definition != "" {
				return c.OldProcedure.Definition + ";"
			}
			return fmt.Sprintf("CREATE OR REPLACE PROCEDURE %s(%s) LANGUAGE %s AS $$%s$$;",
				qualifiedName(c.OldProcedure.Schema, c.OldProcedure.Name),
				c.OldProcedure.Args,
				c.OldProcedure.Language,
				c.OldProcedure.Body)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore previous procedure %s", c.Procedure.Name)
	}
	return ""
}

func (c *ProcedureChange) Type() ChangeType   { return c.ChangeType }
func (c *ProcedureChange) ObjectName() string { return c.Procedure.Name }

func (c *ProcedureChange) IsReversible() bool {
	switch c.ChangeType {
	case CreateProcedure:
		return true
	case DropProcedure, AlterProcedure:
		return c.OldProcedure != nil
	}
	return false
}

func compareProcedures(current, desired []parser.Procedure) []Change {
	var changes []Change
	currentMap := make(map[string]parser.Procedure)
	for _, p := range current {
		currentMap[procedureKey(p)] = p
	}
	for _, p := range desired {
		if existing, exists := currentMap[procedureKey(p)]; !exists {
			changes = append(changes, &ProcedureChange{ChangeType: CreateProcedure, Procedure: p})
		} else if normalizeFunctionBody(existing.Body) != normalizeFunctionBody(p.Body) {
			oldProc := existing
			changes = append(changes, &ProcedureChange{ChangeType: AlterProcedure, Procedure: p, OldProcedure: &oldProc})
		}
	}
	desiredMap := make(map[string]bool)
	for _, p := range desired {
		desiredMap[procedureKey(p)] = true
	}
	for _, p := range current {
		if !desiredMap[procedureKey(p)] {
			oldProc := p
			changes = append(changes, &ProcedureChange{ChangeType: DropProcedure, Procedure: p, OldProcedure: &oldProc})
		}
	}
	return changes
}

// procedureKey produces a unique identifier for a procedure overload using the
// schema, name, and normalized argument-type signature. Procedure overloading
// in Postgres works the same way as for functions — only argument types
// participate in dispatch, so parameter names and DEFAULTs must be stripped
// before keying.
func procedureKey(p parser.Procedure) string {
	return objectKey(p.Schema, p.Name) + "(" + normalizeFunctionSignature(p.Args) + ")"
}

// dropProcedureSQL emits a DROP PROCEDURE statement using the normalized
// argument signature so the resulting SQL is unambiguous regardless of how
// many overloads exist.
func dropProcedureSQL(p parser.Procedure) string {
	if p.Args == "" {
		return fmt.Sprintf("DROP PROCEDURE %s;", qualifiedName(p.Schema, p.Name))
	}
	return fmt.Sprintf("DROP PROCEDURE %s(%s);",
		qualifiedName(p.Schema, p.Name),
		normalizeFunctionSignature(p.Args))
}
