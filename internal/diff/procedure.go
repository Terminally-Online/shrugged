package diff

import (
	"fmt"

	"shrugged/internal/parser"
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
		if c.Procedure.Args != "" {
			return fmt.Sprintf("DROP PROCEDURE %s(%s);",
				qualifiedName(c.Procedure.Schema, c.Procedure.Name),
				c.Procedure.Args)
		}
		return fmt.Sprintf("DROP PROCEDURE %s;", qualifiedName(c.Procedure.Schema, c.Procedure.Name))
	}
	return ""
}

func (c *ProcedureChange) DownSQL() string {
	switch c.ChangeType {
	case CreateProcedure:
		if c.Procedure.Args != "" {
			return fmt.Sprintf("DROP PROCEDURE %s(%s);",
				qualifiedName(c.Procedure.Schema, c.Procedure.Name),
				c.Procedure.Args)
		}
		return fmt.Sprintf("DROP PROCEDURE %s;", qualifiedName(c.Procedure.Schema, c.Procedure.Name))
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
		key := objectKey(p.Schema, p.Name) + "(" + p.Args + ")"
		currentMap[key] = p
	}
	for _, p := range desired {
		key := objectKey(p.Schema, p.Name) + "(" + p.Args + ")"
		if existing, exists := currentMap[key]; !exists {
			changes = append(changes, &ProcedureChange{ChangeType: CreateProcedure, Procedure: p})
		} else if existing.Body != p.Body {
			oldProc := existing
			changes = append(changes, &ProcedureChange{ChangeType: AlterProcedure, Procedure: p, OldProcedure: &oldProc})
		}
	}
	desiredMap := make(map[string]bool)
	for _, p := range desired {
		key := objectKey(p.Schema, p.Name) + "(" + p.Args + ")"
		desiredMap[key] = true
	}
	for _, p := range current {
		key := objectKey(p.Schema, p.Name) + "(" + p.Args + ")"
		if !desiredMap[key] {
			oldProc := p
			changes = append(changes, &ProcedureChange{ChangeType: DropProcedure, Procedure: p, OldProcedure: &oldProc})
		}
	}
	return changes
}
