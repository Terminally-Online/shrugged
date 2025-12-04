package diff

import (
	"fmt"

	"shrugged/internal/parser"
)

type NamespaceChange struct {
	ChangeType ChangeType
	Namespace  parser.Namespace
}

func (c *NamespaceChange) SQL() string {
	switch c.ChangeType {
	case CreateNamespace:
		sql := fmt.Sprintf("CREATE SCHEMA %s", quoteIdent(c.Namespace.Name))
		if c.Namespace.Owner != "" {
			sql += fmt.Sprintf(" AUTHORIZATION %s", quoteIdent(c.Namespace.Owner))
		}
		return sql + ";"
	case DropNamespace:
		return fmt.Sprintf("DROP SCHEMA %s;", quoteIdent(c.Namespace.Name))
	}
	return ""
}

func (c *NamespaceChange) DownSQL() string {
	switch c.ChangeType {
	case CreateNamespace:
		return fmt.Sprintf("DROP SCHEMA %s;", quoteIdent(c.Namespace.Name))
	case DropNamespace:
		sql := fmt.Sprintf("CREATE SCHEMA %s", quoteIdent(c.Namespace.Name))
		if c.Namespace.Owner != "" {
			sql += fmt.Sprintf(" AUTHORIZATION %s", quoteIdent(c.Namespace.Owner))
		}
		return sql + ";"
	}
	return ""
}

func (c *NamespaceChange) Type() ChangeType   { return c.ChangeType }
func (c *NamespaceChange) ObjectName() string { return c.Namespace.Name }
func (c *NamespaceChange) IsReversible() bool { return true }

func compareNamespaces(current, desired []parser.Namespace) []Change {
	var changes []Change
	currentMap := make(map[string]parser.Namespace)
	for _, n := range current {
		currentMap[n.Name] = n
	}
	for _, n := range desired {
		if _, exists := currentMap[n.Name]; !exists {
			changes = append(changes, &NamespaceChange{ChangeType: CreateNamespace, Namespace: n})
		}
	}
	desiredMap := make(map[string]bool)
	for _, n := range desired {
		desiredMap[n.Name] = true
	}
	for _, n := range current {
		if !desiredMap[n.Name] {
			changes = append(changes, &NamespaceChange{ChangeType: DropNamespace, Namespace: n})
		}
	}
	return changes
}
