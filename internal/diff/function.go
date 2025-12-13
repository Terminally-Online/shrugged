package diff

import (
	"fmt"
	"strings"

	"github.com/terminally-online/shrugged/internal/parser"
)

type FunctionChange struct {
	ChangeType  ChangeType
	Function    parser.Function
	OldFunction *parser.Function
}

func (c *FunctionChange) SQL() string {
	switch c.ChangeType {
	case CreateFunction, AlterFunction:
		return generateCreateFunction(c.Function)
	case DropFunction:
		return fmt.Sprintf("DROP FUNCTION %s;", qualifiedName(c.Function.Schema, c.Function.Name))
	}
	return ""
}

func (c *FunctionChange) DownSQL() string {
	switch c.ChangeType {
	case CreateFunction:
		return fmt.Sprintf("DROP FUNCTION %s;", qualifiedName(c.Function.Schema, c.Function.Name))
	case DropFunction:
		if c.OldFunction != nil {
			return generateCreateFunction(*c.OldFunction)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped function %s", c.Function.Name)
	case AlterFunction:
		if c.OldFunction != nil {
			return generateCreateFunction(*c.OldFunction)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore previous function %s", c.Function.Name)
	}
	return ""
}

func (c *FunctionChange) Type() ChangeType {
	return c.ChangeType
}

func (c *FunctionChange) ObjectName() string {
	return c.Function.Name
}

func (c *FunctionChange) IsReversible() bool {
	switch c.ChangeType {
	case CreateFunction:
		return true
	case DropFunction, AlterFunction:
		return c.OldFunction != nil
	}
	return false
}

func compareFunctions(current, desired []parser.Function) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Function)
	for _, f := range current {
		currentMap[f.Name] = f
	}

	desiredMap := make(map[string]parser.Function)
	for _, f := range desired {
		desiredMap[f.Name] = f
	}

	for _, f := range desired {
		if existing, exists := currentMap[f.Name]; !exists {
			changes = append(changes, &FunctionChange{ChangeType: CreateFunction, Function: f})
		} else if existing.Body != f.Body {
			oldFunc := existing
			changes = append(changes, &FunctionChange{ChangeType: AlterFunction, Function: f, OldFunction: &oldFunc})
		}
	}

	for _, f := range current {
		if _, exists := desiredMap[f.Name]; !exists {
			oldFunc := f
			changes = append(changes, &FunctionChange{ChangeType: DropFunction, Function: f, OldFunction: &oldFunc})
		}
	}

	return changes
}

func generateCreateFunction(f parser.Function) string {
	if f.Definition != "" {
		return f.Definition + ";"
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s(%s)", quoteIdent(f.Name), f.Args))
	sb.WriteString(fmt.Sprintf(" RETURNS %s", f.Returns))
	sb.WriteString(fmt.Sprintf(" LANGUAGE %s", f.Language))
	sb.WriteString(fmt.Sprintf(" AS $$%s$$;", f.Body))

	return sb.String()
}
