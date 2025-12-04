package diff

import (
	"fmt"
	"strings"

	"shrugged/internal/parser"
)

type CollationChange struct {
	ChangeType   ChangeType
	Collation    parser.Collation
	OldCollation *parser.Collation
}

func (c *CollationChange) SQL() string {
	collName := qualifiedName(c.Collation.Schema, c.Collation.Name)
	switch c.ChangeType {
	case CreateCollation:
		sql := fmt.Sprintf("CREATE COLLATION %s (", collName)
		var opts []string
		if c.Collation.Provider != "" {
			opts = append(opts, fmt.Sprintf("PROVIDER = %s", c.Collation.Provider))
		}
		if c.Collation.Locale != "" {
			opts = append(opts, fmt.Sprintf("LOCALE = '%s'", c.Collation.Locale))
		}
		if c.Collation.LcCollate != "" {
			opts = append(opts, fmt.Sprintf("LC_COLLATE = '%s'", c.Collation.LcCollate))
		}
		if c.Collation.LcCtype != "" {
			opts = append(opts, fmt.Sprintf("LC_CTYPE = '%s'", c.Collation.LcCtype))
		}
		return sql + strings.Join(opts, ", ") + ");"
	case DropCollation:
		return fmt.Sprintf("DROP COLLATION %s;", collName)
	}
	return ""
}

func (c *CollationChange) DownSQL() string {
	collName := qualifiedName(c.Collation.Schema, c.Collation.Name)
	switch c.ChangeType {
	case CreateCollation:
		return fmt.Sprintf("DROP COLLATION %s;", collName)
	case DropCollation:
		if c.OldCollation != nil {
			oldName := qualifiedName(c.OldCollation.Schema, c.OldCollation.Name)
			sql := fmt.Sprintf("CREATE COLLATION %s (", oldName)
			var opts []string
			if c.OldCollation.Provider != "" {
				opts = append(opts, fmt.Sprintf("PROVIDER = %s", c.OldCollation.Provider))
			}
			if c.OldCollation.Locale != "" {
				opts = append(opts, fmt.Sprintf("LOCALE = '%s'", c.OldCollation.Locale))
			}
			if c.OldCollation.LcCollate != "" {
				opts = append(opts, fmt.Sprintf("LC_COLLATE = '%s'", c.OldCollation.LcCollate))
			}
			if c.OldCollation.LcCtype != "" {
				opts = append(opts, fmt.Sprintf("LC_CTYPE = '%s'", c.OldCollation.LcCtype))
			}
			return sql + strings.Join(opts, ", ") + ");"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped collation %s", c.Collation.Name)
	}
	return ""
}

func (c *CollationChange) Type() ChangeType   { return c.ChangeType }
func (c *CollationChange) ObjectName() string { return c.Collation.Name }
func (c *CollationChange) IsReversible() bool {
	if c.ChangeType == CreateCollation {
		return true
	}
	return c.OldCollation != nil
}

func compareCollations(current, desired []parser.Collation) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Collation)
	for _, c := range current {
		currentMap[objectKey(c.Schema, c.Name)] = c
	}

	desiredMap := make(map[string]parser.Collation)
	for _, c := range desired {
		desiredMap[objectKey(c.Schema, c.Name)] = c
	}

	for _, c := range desired {
		if _, exists := currentMap[objectKey(c.Schema, c.Name)]; !exists {
			changes = append(changes, &CollationChange{ChangeType: CreateCollation, Collation: c})
		}
	}

	for _, c := range current {
		if _, exists := desiredMap[objectKey(c.Schema, c.Name)]; !exists {
			oldColl := c
			changes = append(changes, &CollationChange{ChangeType: DropCollation, Collation: c, OldCollation: &oldColl})
		}
	}

	return changes
}
