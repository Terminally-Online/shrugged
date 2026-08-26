package diff

import (
	"fmt"
	"strings"

	"git.ca.plug.to/terminally-online/shrugged/internal/parser"
)

type IndexChange struct {
	ChangeType ChangeType
	Index      parser.Index
	OldIndex   *parser.Index
}

func (c *IndexChange) SQL() string {
	switch c.ChangeType {
	case CreateIndex:
		return generateCreateIndex(c.Index)
	case DropIndex:
		return fmt.Sprintf("DROP INDEX IF EXISTS %s;", qualifiedName(c.Index.Schema, c.Index.Name))
	}
	return ""
}

func (c *IndexChange) DownSQL() string {
	switch c.ChangeType {
	case CreateIndex:
		return fmt.Sprintf("DROP INDEX IF EXISTS %s;", qualifiedName(c.Index.Schema, c.Index.Name))
	case DropIndex:
		if c.OldIndex != nil {
			return generateCreateIndex(*c.OldIndex)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped index %s", c.Index.Name)
	}
	return ""
}

func (c *IndexChange) Type() ChangeType {
	return c.ChangeType
}

func (c *IndexChange) ObjectName() string {
	return c.Index.Name
}

func (c *IndexChange) IsReversible() bool {
	if c.ChangeType == DropIndex && c.OldIndex == nil {
		return false
	}
	return true
}

func compareIndexes(current, desired []parser.Index) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Index)
	for _, i := range current {
		currentMap[i.Name] = i
	}

	desiredMap := make(map[string]parser.Index)
	for _, i := range desired {
		desiredMap[i.Name] = i
	}

	for _, i := range desired {
		cur, exists := currentMap[i.Name]
		if !exists {
			changes = append(changes, &IndexChange{ChangeType: CreateIndex, Index: i})
			continue
		}
		// Same name, changed definition (columns, predicate, uniqueness, …): an
		// index can't be replaced in place, so drop the current one and recreate
		// it. Both definitions come from pg_get_indexdef — current and desired are
		// each introspected from Postgres — so the comparison is normalized and a
		// textual definition change is a real change, not a formatting artifact.
		if !indexesEqual(cur, i) {
			oldIdx := cur
			changes = append(changes, &IndexChange{ChangeType: DropIndex, Index: cur, OldIndex: &oldIdx})
			changes = append(changes, &IndexChange{ChangeType: CreateIndex, Index: i})
		}
	}

	for _, i := range current {
		if _, exists := desiredMap[i.Name]; !exists {
			oldIdx := i
			changes = append(changes, &IndexChange{ChangeType: DropIndex, Index: i, OldIndex: &oldIdx})
		}
	}

	return changes
}

// indexesEqual reports whether two indexes are structurally identical. Both
// definitions originate from pg_get_indexdef (the diff compares two introspected
// schemas), so a normalized definition comparison is exact — it catches predicate,
// column, ordering, and uniqueness changes a name-only comparison misses.
func indexesEqual(a, b parser.Index) bool {
	return normalizeIndexDef(a.Definition) == normalizeIndexDef(b.Definition)
}

func normalizeIndexDef(def string) string {
	return strings.TrimSuffix(strings.TrimSpace(def), ";")
}

func generateCreateIndex(i parser.Index) string {
	if i.Definition != "" {
		def := strings.TrimSpace(i.Definition)
		if !strings.HasSuffix(def, ";") {
			def += ";"
		}
		return def
	}

	var sb strings.Builder

	sb.WriteString("CREATE ")
	if i.Unique {
		sb.WriteString("UNIQUE ")
	}
	sb.WriteString(fmt.Sprintf("INDEX %s ON %s", quoteIdent(i.Name), qualifiedName(i.Schema, i.Table)))

	if i.Using != "" && i.Using != "btree" {
		sb.WriteString(fmt.Sprintf(" USING %s", i.Using))
	}

	sb.WriteString(fmt.Sprintf(" (%s)", strings.Join(i.Columns, ", ")))

	if i.Where != "" {
		sb.WriteString(fmt.Sprintf(" WHERE %s", i.Where))
	}

	sb.WriteString(";")
	return sb.String()
}
