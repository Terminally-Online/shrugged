package diff

import (
	"fmt"

	"shrugged/internal/parser"
)

type ExtensionChange struct {
	ChangeType ChangeType
	Extension  parser.Extension
}

func (c *ExtensionChange) SQL() string {
	switch c.ChangeType {
	case CreateExtension:
		sql := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", quoteIdent(c.Extension.Name))
		if c.Extension.Schema != "" && c.Extension.Schema != "public" {
			sql += fmt.Sprintf(" SCHEMA %s", quoteIdent(c.Extension.Schema))
		}
		return sql + ";"
	case DropExtension:
		return fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", quoteIdent(c.Extension.Name))
	}
	return ""
}

func (c *ExtensionChange) DownSQL() string {
	switch c.ChangeType {
	case CreateExtension:
		return fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", quoteIdent(c.Extension.Name))
	case DropExtension:
		sql := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", quoteIdent(c.Extension.Name))
		if c.Extension.Schema != "" && c.Extension.Schema != "public" {
			sql += fmt.Sprintf(" SCHEMA %s", quoteIdent(c.Extension.Schema))
		}
		return sql + ";"
	}
	return ""
}

func (c *ExtensionChange) Type() ChangeType   { return c.ChangeType }
func (c *ExtensionChange) ObjectName() string { return c.Extension.Name }
func (c *ExtensionChange) IsReversible() bool { return true }

func compareExtensions(current, desired []parser.Extension) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Extension)
	for _, e := range current {
		currentMap[e.Name] = e
	}

	desiredMap := make(map[string]parser.Extension)
	for _, e := range desired {
		desiredMap[e.Name] = e
	}

	for _, e := range desired {
		if _, exists := currentMap[e.Name]; !exists {
			changes = append(changes, &ExtensionChange{ChangeType: CreateExtension, Extension: e})
		}
	}

	for _, e := range current {
		if _, exists := desiredMap[e.Name]; !exists {
			changes = append(changes, &ExtensionChange{ChangeType: DropExtension, Extension: e})
		}
	}

	return changes
}
