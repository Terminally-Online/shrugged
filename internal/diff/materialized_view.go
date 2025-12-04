package diff

import (
	"fmt"

	"shrugged/internal/parser"
)

type MaterializedViewChange struct {
	ChangeType          ChangeType
	MaterializedView    parser.MaterializedView
	OldMaterializedView *parser.MaterializedView
}

func (c *MaterializedViewChange) SQL() string {
	mvName := qualifiedName(c.MaterializedView.Schema, c.MaterializedView.Name)
	switch c.ChangeType {
	case CreateMaterializedView:
		sql := fmt.Sprintf("CREATE MATERIALIZED VIEW %s AS %s", mvName, c.MaterializedView.Definition)
		if !c.MaterializedView.WithData {
			sql += " WITH NO DATA"
		}
		return sql + ";"
	case DropMaterializedView:
		return fmt.Sprintf("DROP MATERIALIZED VIEW %s;", mvName)
	case AlterMaterializedView:
		return fmt.Sprintf("DROP MATERIALIZED VIEW %s;\n%s", mvName,
			(&MaterializedViewChange{ChangeType: CreateMaterializedView, MaterializedView: c.MaterializedView}).SQL())
	}
	return ""
}

func (c *MaterializedViewChange) DownSQL() string {
	mvName := qualifiedName(c.MaterializedView.Schema, c.MaterializedView.Name)
	switch c.ChangeType {
	case CreateMaterializedView:
		return fmt.Sprintf("DROP MATERIALIZED VIEW %s;", mvName)
	case DropMaterializedView:
		if c.OldMaterializedView != nil {
			sql := fmt.Sprintf("CREATE MATERIALIZED VIEW %s AS %s",
				qualifiedName(c.OldMaterializedView.Schema, c.OldMaterializedView.Name),
				c.OldMaterializedView.Definition)
			if !c.OldMaterializedView.WithData {
				sql += " WITH NO DATA"
			}
			return sql + ";"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore materialized view %s", c.MaterializedView.Name)
	case AlterMaterializedView:
		if c.OldMaterializedView != nil {
			return fmt.Sprintf("DROP MATERIALIZED VIEW %s;\n%s", mvName,
				(&MaterializedViewChange{ChangeType: CreateMaterializedView, MaterializedView: *c.OldMaterializedView}).SQL())
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore materialized view %s", c.MaterializedView.Name)
	}
	return ""
}

func (c *MaterializedViewChange) Type() ChangeType   { return c.ChangeType }
func (c *MaterializedViewChange) ObjectName() string { return c.MaterializedView.Name }
func (c *MaterializedViewChange) IsReversible() bool {
	if c.ChangeType == CreateMaterializedView {
		return true
	}
	return c.OldMaterializedView != nil
}

func compareMaterializedViews(current, desired []parser.MaterializedView) []Change {
	var changes []Change

	currentMap := make(map[string]parser.MaterializedView)
	for _, v := range current {
		currentMap[objectKey(v.Schema, v.Name)] = v
	}

	desiredMap := make(map[string]parser.MaterializedView)
	for _, v := range desired {
		desiredMap[objectKey(v.Schema, v.Name)] = v
	}

	for _, v := range desired {
		if existing, exists := currentMap[objectKey(v.Schema, v.Name)]; !exists {
			changes = append(changes, &MaterializedViewChange{ChangeType: CreateMaterializedView, MaterializedView: v})
		} else if normalizeSQL(existing.Definition) != normalizeSQL(v.Definition) {
			oldMV := existing
			changes = append(changes, &MaterializedViewChange{ChangeType: AlterMaterializedView, MaterializedView: v, OldMaterializedView: &oldMV})
		}
	}

	for _, v := range current {
		if _, exists := desiredMap[objectKey(v.Schema, v.Name)]; !exists {
			oldMV := v
			changes = append(changes, &MaterializedViewChange{ChangeType: DropMaterializedView, MaterializedView: v, OldMaterializedView: &oldMV})
		}
	}

	return changes
}
