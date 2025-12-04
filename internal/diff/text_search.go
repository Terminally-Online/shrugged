package diff

import (
	"fmt"

	"shrugged/internal/parser"
)

type TextSearchConfigChange struct {
	ChangeType          ChangeType
	TextSearchConfig    parser.TextSearchConfig
	OldTextSearchConfig *parser.TextSearchConfig
}

func (c *TextSearchConfigChange) SQL() string {
	cfgName := qualifiedName(c.TextSearchConfig.Schema, c.TextSearchConfig.Name)
	switch c.ChangeType {
	case CreateTextSearchConfig:
		return fmt.Sprintf("CREATE TEXT SEARCH CONFIGURATION %s (PARSER = %s);",
			cfgName, c.TextSearchConfig.Parser)
	case DropTextSearchConfig:
		return fmt.Sprintf("DROP TEXT SEARCH CONFIGURATION %s;", cfgName)
	}
	return ""
}

func (c *TextSearchConfigChange) DownSQL() string {
	cfgName := qualifiedName(c.TextSearchConfig.Schema, c.TextSearchConfig.Name)
	switch c.ChangeType {
	case CreateTextSearchConfig:
		return fmt.Sprintf("DROP TEXT SEARCH CONFIGURATION %s;", cfgName)
	case DropTextSearchConfig:
		if c.OldTextSearchConfig != nil {
			return fmt.Sprintf("CREATE TEXT SEARCH CONFIGURATION %s (PARSER = %s);",
				qualifiedName(c.OldTextSearchConfig.Schema, c.OldTextSearchConfig.Name),
				c.OldTextSearchConfig.Parser)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped text search config %s", c.TextSearchConfig.Name)
	}
	return ""
}

func (c *TextSearchConfigChange) Type() ChangeType   { return c.ChangeType }
func (c *TextSearchConfigChange) ObjectName() string { return c.TextSearchConfig.Name }
func (c *TextSearchConfigChange) IsReversible() bool {
	if c.ChangeType == CreateTextSearchConfig {
		return true
	}
	return c.OldTextSearchConfig != nil
}

func compareTextSearchConfigs(current, desired []parser.TextSearchConfig) []Change {
	var changes []Change
	currentMap := make(map[string]parser.TextSearchConfig)
	for _, t := range current {
		currentMap[t.Name] = t
	}
	for _, t := range desired {
		if _, exists := currentMap[t.Name]; !exists {
			changes = append(changes, &TextSearchConfigChange{ChangeType: CreateTextSearchConfig, TextSearchConfig: t})
		}
	}
	desiredMap := make(map[string]bool)
	for _, t := range desired {
		desiredMap[t.Name] = true
	}
	for _, t := range current {
		if !desiredMap[t.Name] {
			oldTS := t
			changes = append(changes, &TextSearchConfigChange{ChangeType: DropTextSearchConfig, TextSearchConfig: t, OldTextSearchConfig: &oldTS})
		}
	}
	return changes
}
