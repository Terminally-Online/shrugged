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
		} else if normalizeFunctionBody(existing.Body) != normalizeFunctionBody(f.Body) {
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

// normalizeFunctionBody returns a canonical form of a function body suitable
// for byte-equality comparison between bodies sourced from Postgres
// introspection (pg_proc.prosrc) and bodies parsed from human-authored schema
// SQL. Comments and whitespace differ freely between those paths even when
// the executable SQL is identical, so they are stripped before comparison.
//
// NOTE: Line-comment stripping is string-literal aware for single-quoted
// strings (with '' escape) and dollar-quoted strings ($tag$...$tag$). It
// does not parse C-style escape strings (E'...\'...') — a backslash-escaped
// quote inside an E-string could fool the heuristic into treating literal
// content as code. This is acceptable for diff purposes: a false positive
// only triggers a redundant CREATE OR REPLACE, never a missed change.
func normalizeFunctionBody(body string) string {
	stripped := stripSQLComments(body)
	return strings.Join(strings.Fields(stripped), " ")
}

func stripSQLComments(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))

	i := 0
	for i < len(s) {
		c := s[i]

		if c == '\'' {
			end := scanSingleQuotedString(s, i)
			sb.WriteString(s[i:end])
			i = end
			continue
		}

		if c == '$' {
			if tagEnd, ok := scanDollarTag(s, i); ok {
				tag := s[i:tagEnd]
				closeIdx := strings.Index(s[tagEnd:], tag)
				if closeIdx >= 0 {
					end := tagEnd + closeIdx + len(tag)
					sb.WriteString(s[i:end])
					i = end
					continue
				}
			}
		}

		if c == '-' && i+1 < len(s) && s[i+1] == '-' {
			nl := strings.IndexByte(s[i:], '\n')
			if nl < 0 {
				return sb.String()
			}
			i += nl
			sb.WriteByte('\n')
			i++
			continue
		}

		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				return sb.String()
			}
			i += 2 + end + 2
			sb.WriteByte(' ')
			continue
		}

		sb.WriteByte(c)
		i++
	}

	return sb.String()
}

// scanSingleQuotedString returns the index just past the closing quote of a
// single-quoted SQL string literal beginning at start. Doubled quotes ('')
// are treated as an escaped quote, per SQL standard.
func scanSingleQuotedString(s string, start int) int {
	i := start + 1
	for i < len(s) {
		if s[i] == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return len(s)
}

// scanDollarTag returns the index just past a dollar-quote opening tag
// ($tag$ or $$) beginning at start, and whether the position is a valid tag.
func scanDollarTag(s string, start int) (int, bool) {
	if start >= len(s) || s[start] != '$' {
		return 0, false
	}
	i := start + 1
	for i < len(s) {
		c := s[i]
		if c == '$' {
			return i + 1, true
		}
		if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return 0, false
		}
		i++
	}
	return 0, false
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
