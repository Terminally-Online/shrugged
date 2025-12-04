package diff

import (
	"fmt"
	"strings"

	"shrugged/internal/parser"
)

type CompositeTypeChange struct {
	ChangeType       ChangeType
	CompositeType    parser.CompositeType
	OldCompositeType *parser.CompositeType
}

func (c *CompositeTypeChange) SQL() string {
	typeName := qualifiedName(c.CompositeType.Schema, c.CompositeType.Name)
	switch c.ChangeType {
	case CreateCompositeType:
		var attrs []string
		for _, a := range c.CompositeType.Attributes {
			attrs = append(attrs, fmt.Sprintf("%s %s", quoteIdent(a.Name), a.Type))
		}
		return fmt.Sprintf("CREATE TYPE %s AS (%s);", typeName, strings.Join(attrs, ", "))
	case DropCompositeType:
		return fmt.Sprintf("DROP TYPE %s;", typeName)
	}
	return ""
}

func (c *CompositeTypeChange) DownSQL() string {
	typeName := qualifiedName(c.CompositeType.Schema, c.CompositeType.Name)
	switch c.ChangeType {
	case CreateCompositeType:
		return fmt.Sprintf("DROP TYPE %s;", typeName)
	case DropCompositeType:
		if c.OldCompositeType != nil {
			var attrs []string
			for _, a := range c.OldCompositeType.Attributes {
				attrs = append(attrs, fmt.Sprintf("%s %s", quoteIdent(a.Name), a.Type))
			}
			return fmt.Sprintf("CREATE TYPE %s AS (%s);",
				qualifiedName(c.OldCompositeType.Schema, c.OldCompositeType.Name),
				strings.Join(attrs, ", "))
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped type %s", c.CompositeType.Name)
	}
	return ""
}

func (c *CompositeTypeChange) Type() ChangeType   { return c.ChangeType }
func (c *CompositeTypeChange) ObjectName() string { return c.CompositeType.Name }
func (c *CompositeTypeChange) IsReversible() bool {
	if c.ChangeType == CreateCompositeType {
		return true
	}
	return c.OldCompositeType != nil
}

func compareCompositeTypes(current, desired []parser.CompositeType) []Change {
	var changes []Change

	currentMap := make(map[string]parser.CompositeType)
	for _, t := range current {
		currentMap[objectKey(t.Schema, t.Name)] = t
	}

	desiredMap := make(map[string]parser.CompositeType)
	for _, t := range desired {
		desiredMap[objectKey(t.Schema, t.Name)] = t
	}

	for _, t := range desired {
		if _, exists := currentMap[objectKey(t.Schema, t.Name)]; !exists {
			changes = append(changes, &CompositeTypeChange{ChangeType: CreateCompositeType, CompositeType: t})
		}
	}

	for _, t := range current {
		if _, exists := desiredMap[objectKey(t.Schema, t.Name)]; !exists {
			oldType := t
			changes = append(changes, &CompositeTypeChange{ChangeType: DropCompositeType, CompositeType: t, OldCompositeType: &oldType})
		}
	}

	return changes
}
