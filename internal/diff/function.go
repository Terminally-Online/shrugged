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
		return fmt.Sprintf("DROP FUNCTION %s(%s);",
			qualifiedName(c.Function.Schema, c.Function.Name),
			normalizeFunctionSignature(c.Function.Args))
	}
	return ""
}

func (c *FunctionChange) DownSQL() string {
	switch c.ChangeType {
	case CreateFunction:
		return fmt.Sprintf("DROP FUNCTION %s(%s);",
			qualifiedName(c.Function.Schema, c.Function.Name),
			normalizeFunctionSignature(c.Function.Args))
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
		currentMap[functionKey(f)] = f
	}

	desiredMap := make(map[string]parser.Function)
	for _, f := range desired {
		desiredMap[functionKey(f)] = f
	}

	for _, f := range desired {
		if existing, exists := currentMap[functionKey(f)]; !exists {
			changes = append(changes, &FunctionChange{ChangeType: CreateFunction, Function: f})
		} else if existing.Body != f.Body {
			oldFunc := existing
			changes = append(changes, &FunctionChange{ChangeType: AlterFunction, Function: f, OldFunction: &oldFunc})
		}
	}

	for _, f := range current {
		if _, exists := desiredMap[functionKey(f)]; !exists {
			oldFunc := f
			changes = append(changes, &FunctionChange{ChangeType: DropFunction, Function: f, OldFunction: &oldFunc})
		}
	}

	return changes
}

// functionKey produces a unique identifier for a function overload using the
// schema, name, and normalized argument-type signature. Postgres identifies
// overloads by argument types only, so two functions with the same name but
// different signatures must be tracked as distinct entries.
func functionKey(f parser.Function) string {
	return objectKey(f.Schema, f.Name) + "(" + normalizeFunctionSignature(f.Args) + ")"
}

// normalizeFunctionSignature reduces an argument list to the comma-separated
// list of argument types that Postgres uses to identify an overload. Parameter
// names and DEFAULT clauses are stripped; argument-mode keywords (IN, OUT,
// INOUT, VARIADIC) are preserved as-is since they participate in dispatch
// semantics for VARIADIC and are otherwise inert.
func normalizeFunctionSignature(args string) string {
	args = strings.TrimSpace(args)
	if args == "" {
		return ""
	}

	parts := splitTopLevelCommas(args)
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		if t := normalizeArgType(part); t != "" {
			normalized = append(normalized, t)
		}
	}
	return strings.Join(normalized, ", ")
}

// splitTopLevelCommas splits a string on commas that are not nested inside
// parentheses (e.g. so that "numeric(10,2)" is not torn apart).
func splitTopLevelCommas(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// normalizeArgType extracts the type portion of a single argument declaration,
// dropping any leading parameter name and trailing DEFAULT clause. Argument
// modes (IN/OUT/INOUT/VARIADIC) are preserved at the front of the result.
func normalizeArgType(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return ""
	}

	// Strip DEFAULT clause: "foo int DEFAULT 7" or "foo int = 7".
	if idx := indexOfTopLevelDefault(arg); idx >= 0 {
		arg = strings.TrimSpace(arg[:idx])
	}

	tokens := tokenizeArg(arg)
	if len(tokens) == 0 {
		return ""
	}

	var mode string
	switch strings.ToUpper(tokens[0]) {
	case "IN", "OUT", "INOUT", "VARIADIC":
		mode = strings.ToUpper(tokens[0])
		tokens = tokens[1:]
	}

	if len(tokens) == 0 {
		return mode
	}

	// If there is more than one token remaining, the first one is a parameter
	// name and the rest form the type. Otherwise the single token IS the type.
	var typePart string
	if len(tokens) == 1 {
		typePart = tokens[0]
	} else {
		typePart = strings.Join(tokens[1:], " ")
	}

	typePart = strings.TrimSpace(typePart)
	if mode != "" {
		return mode + " " + typePart
	}
	return typePart
}

// indexOfTopLevelDefault returns the index in arg where a DEFAULT clause
// begins, or -1 if none is present. Recognizes both the DEFAULT keyword
// (case-insensitive) and the "=" shorthand. Anything inside parentheses or a
// string literal is ignored.
func indexOfTopLevelDefault(arg string) int {
	depth := 0
	inSingle := false
	upper := strings.ToUpper(arg)
	for i := 0; i < len(arg); i++ {
		c := arg[i]
		if inSingle {
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		switch c {
		case '\'':
			inSingle = true
		case '(':
			depth++
		case ')':
			depth--
		case '=':
			if depth == 0 {
				return i
			}
		}
		if depth == 0 && c == ' ' && i+8 <= len(arg) && upper[i+1:i+8] == "DEFAULT" {
			// Require word boundary after DEFAULT.
			if i+8 == len(arg) || upper[i+8] == ' ' {
				return i
			}
		}
	}
	return -1
}

// tokenizeArg splits an argument declaration into whitespace-separated tokens
// while keeping parenthesized groups (e.g. "numeric(10, 2)") attached to the
// preceding token.
func tokenizeArg(arg string) []string {
	var tokens []string
	var current strings.Builder
	depth := 0
	for _, r := range arg {
		switch {
		case r == '(':
			depth++
			current.WriteRune(r)
		case r == ')':
			depth--
			current.WriteRune(r)
		case (r == ' ' || r == '\t' || r == '\n') && depth == 0:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
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
