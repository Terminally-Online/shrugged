package diff

import (
	"fmt"

	"github.com/terminally-online/shrugged/internal/parser"
)

type RuleChange struct {
	ChangeType ChangeType
	Rule       parser.Rule
	OldRule    *parser.Rule
}

func (c *RuleChange) SQL() string {
	switch c.ChangeType {
	case CreateRule:
		return c.Rule.Definition + ";"
	case DropRule:
		return fmt.Sprintf("DROP RULE %s ON %s;", quoteIdent(c.Rule.Name), qualifiedName(c.Rule.Schema, c.Rule.Table))
	}
	return ""
}

func (c *RuleChange) DownSQL() string {
	switch c.ChangeType {
	case CreateRule:
		return fmt.Sprintf("DROP RULE %s ON %s;", quoteIdent(c.Rule.Name), qualifiedName(c.Rule.Schema, c.Rule.Table))
	case DropRule:
		if c.OldRule != nil {
			return c.OldRule.Definition + ";"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped rule %s", c.Rule.Name)
	}
	return ""
}

func (c *RuleChange) Type() ChangeType   { return c.ChangeType }
func (c *RuleChange) ObjectName() string { return c.Rule.Name }
func (c *RuleChange) IsReversible() bool {
	if c.ChangeType == CreateRule {
		return true
	}
	return c.OldRule != nil
}

func compareRules(current, desired []parser.Rule) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Rule)
	for _, r := range current {
		currentMap[r.Name] = r
	}

	desiredMap := make(map[string]parser.Rule)
	for _, r := range desired {
		desiredMap[r.Name] = r
	}

	for _, r := range desired {
		if _, exists := currentMap[r.Name]; !exists {
			changes = append(changes, &RuleChange{ChangeType: CreateRule, Rule: r})
		}
	}

	for _, r := range current {
		if _, exists := desiredMap[r.Name]; !exists {
			oldRule := r
			changes = append(changes, &RuleChange{ChangeType: DropRule, Rule: r, OldRule: &oldRule})
		}
	}

	return changes
}
