package diff

import (
	"fmt"
	"strings"

	"github.com/terminally-online/shrugged/internal/parser"
)

type EnumChange struct {
	ChangeType ChangeType
	Enum       parser.Enum
	OldEnum    *parser.Enum
	AddValues  []string
}

func (c *EnumChange) SQL() string {
	switch c.ChangeType {
	case CreateEnum:
		return fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);",
			qualifiedName(c.Enum.Schema, c.Enum.Name),
			quoteLiterals(c.Enum.Values))
	case DropEnum:
		return fmt.Sprintf("DROP TYPE %s;", qualifiedName(c.Enum.Schema, c.Enum.Name))
	case AlterEnum:
		var stmts []string
		for _, v := range c.AddValues {
			stmts = append(stmts, fmt.Sprintf("ALTER TYPE %s ADD VALUE %s;",
				qualifiedName(c.Enum.Schema, c.Enum.Name),
				quoteLiteral(v)))
		}
		return strings.Join(stmts, "\n")
	}
	return ""
}

func (c *EnumChange) DownSQL() string {
	switch c.ChangeType {
	case CreateEnum:
		return fmt.Sprintf("DROP TYPE %s;", qualifiedName(c.Enum.Schema, c.Enum.Name))
	case DropEnum:
		if c.OldEnum != nil {
			return fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);",
				qualifiedName(c.OldEnum.Schema, c.OldEnum.Name),
				quoteLiterals(c.OldEnum.Values))
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped enum %s", c.Enum.Name)
	case AlterEnum:
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot remove enum values from %s", c.Enum.Name)
	}
	return ""
}

func (c *EnumChange) Type() ChangeType {
	return c.ChangeType
}

func (c *EnumChange) ObjectName() string {
	return c.Enum.Name
}

func (c *EnumChange) IsReversible() bool {
	switch c.ChangeType {
	case CreateEnum:
		return true
	case DropEnum:
		return c.OldEnum != nil
	case AlterEnum:
		return false
	}
	return false
}

func compareEnums(current, desired []parser.Enum) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Enum)
	for _, e := range current {
		currentMap[e.Name] = e
	}

	desiredMap := make(map[string]parser.Enum)
	for _, e := range desired {
		desiredMap[e.Name] = e
	}

	for _, e := range desired {
		if existing, exists := currentMap[e.Name]; !exists {
			changes = append(changes, &EnumChange{ChangeType: CreateEnum, Enum: e})
		} else {
			existingValues := make(map[string]bool)
			for _, v := range existing.Values {
				existingValues[v] = true
			}

			var newValues []string
			for _, v := range e.Values {
				if !existingValues[v] {
					newValues = append(newValues, v)
				}
			}

			if len(newValues) > 0 {
				changes = append(changes, &EnumChange{ChangeType: AlterEnum, Enum: e, AddValues: newValues})
			}
		}
	}

	for _, e := range current {
		if _, exists := desiredMap[e.Name]; !exists {
			oldEnum := e
			changes = append(changes, &EnumChange{ChangeType: DropEnum, Enum: e, OldEnum: &oldEnum})
		}
	}

	return changes
}

func quoteLiterals(ss []string) string {
	var quoted []string
	for _, s := range ss {
		quoted = append(quoted, quoteLiteral(s))
	}
	return strings.Join(quoted, ", ")
}
