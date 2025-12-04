package diff

import (
	"fmt"
	"strings"

	"shrugged/internal/parser"
)

type PolicyChange struct {
	ChangeType ChangeType
	Policy     parser.Policy
	OldPolicy  *parser.Policy
}

func (c *PolicyChange) SQL() string {
	tableName := qualifiedName(c.Policy.Schema, c.Policy.Table)
	switch c.ChangeType {
	case CreatePolicy:
		sql := fmt.Sprintf("CREATE POLICY %s ON %s", quoteIdent(c.Policy.Name), tableName)
		if !c.Policy.Permissive {
			sql += " AS RESTRICTIVE"
		}
		if c.Policy.Command != "ALL" {
			sql += fmt.Sprintf(" FOR %s", c.Policy.Command)
		}
		if len(c.Policy.Roles) > 0 {
			sql += fmt.Sprintf(" TO %s", strings.Join(c.Policy.Roles, ", "))
		}
		if c.Policy.Using != "" {
			sql += fmt.Sprintf(" USING (%s)", c.Policy.Using)
		}
		if c.Policy.WithCheck != "" {
			sql += fmt.Sprintf(" WITH CHECK (%s)", c.Policy.WithCheck)
		}
		return sql + ";"
	case DropPolicy:
		return fmt.Sprintf("DROP POLICY %s ON %s;", quoteIdent(c.Policy.Name), tableName)
	}
	return ""
}

func (c *PolicyChange) DownSQL() string {
	tableName := qualifiedName(c.Policy.Schema, c.Policy.Table)
	switch c.ChangeType {
	case CreatePolicy:
		return fmt.Sprintf("DROP POLICY %s ON %s;", quoteIdent(c.Policy.Name), tableName)
	case DropPolicy:
		if c.OldPolicy != nil {
			oldTableName := qualifiedName(c.OldPolicy.Schema, c.OldPolicy.Table)
			sql := fmt.Sprintf("CREATE POLICY %s ON %s", quoteIdent(c.OldPolicy.Name), oldTableName)
			if !c.OldPolicy.Permissive {
				sql += " AS RESTRICTIVE"
			}
			if c.OldPolicy.Command != "ALL" {
				sql += fmt.Sprintf(" FOR %s", c.OldPolicy.Command)
			}
			if len(c.OldPolicy.Roles) > 0 {
				sql += fmt.Sprintf(" TO %s", strings.Join(c.OldPolicy.Roles, ", "))
			}
			if c.OldPolicy.Using != "" {
				sql += fmt.Sprintf(" USING (%s)", c.OldPolicy.Using)
			}
			if c.OldPolicy.WithCheck != "" {
				sql += fmt.Sprintf(" WITH CHECK (%s)", c.OldPolicy.WithCheck)
			}
			return sql + ";"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped policy %s", c.Policy.Name)
	}
	return ""
}

func (c *PolicyChange) Type() ChangeType   { return c.ChangeType }
func (c *PolicyChange) ObjectName() string { return c.Policy.Name }
func (c *PolicyChange) IsReversible() bool {
	if c.ChangeType == CreatePolicy {
		return true
	}
	return c.OldPolicy != nil
}

func comparePolicies(current, desired []parser.Policy) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Policy)
	for _, p := range current {
		currentMap[p.Name] = p
	}

	desiredMap := make(map[string]parser.Policy)
	for _, p := range desired {
		desiredMap[p.Name] = p
	}

	for _, p := range desired {
		if _, exists := currentMap[p.Name]; !exists {
			changes = append(changes, &PolicyChange{ChangeType: CreatePolicy, Policy: p})
		}
	}

	for _, p := range current {
		if _, exists := desiredMap[p.Name]; !exists {
			oldPol := p
			changes = append(changes, &PolicyChange{ChangeType: DropPolicy, Policy: p, OldPolicy: &oldPol})
		}
	}

	return changes
}
