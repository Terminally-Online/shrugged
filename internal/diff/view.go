package diff

import (
	"fmt"

	"shrugged/internal/parser"
)

type ViewChange struct {
	ChangeType ChangeType
	View       parser.View
	OldView    *parser.View
}

func (c *ViewChange) SQL() string {
	switch c.ChangeType {
	case CreateView, AlterView:
		return fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s;",
			qualifiedName(c.View.Schema, c.View.Name),
			c.View.Definition)
	case DropView:
		return fmt.Sprintf("DROP VIEW %s;", qualifiedName(c.View.Schema, c.View.Name))
	}
	return ""
}

func (c *ViewChange) DownSQL() string {
	switch c.ChangeType {
	case CreateView:
		return fmt.Sprintf("DROP VIEW %s;", qualifiedName(c.View.Schema, c.View.Name))
	case DropView:
		if c.OldView != nil {
			return fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s;",
				qualifiedName(c.OldView.Schema, c.OldView.Name),
				c.OldView.Definition)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped view %s", c.View.Name)
	case AlterView:
		if c.OldView != nil {
			return fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s;",
				qualifiedName(c.OldView.Schema, c.OldView.Name),
				c.OldView.Definition)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore previous view %s", c.View.Name)
	}
	return ""
}

func (c *ViewChange) Type() ChangeType {
	return c.ChangeType
}

func (c *ViewChange) ObjectName() string {
	return c.View.Name
}

func (c *ViewChange) IsReversible() bool {
	switch c.ChangeType {
	case CreateView:
		return true
	case DropView, AlterView:
		return c.OldView != nil
	}
	return false
}

func compareViews(current, desired []parser.View) []Change {
	var changes []Change

	currentMap := make(map[string]parser.View)
	for _, v := range current {
		currentMap[v.Name] = v
	}

	desiredMap := make(map[string]parser.View)
	for _, v := range desired {
		desiredMap[v.Name] = v
	}

	for _, v := range desired {
		if existing, exists := currentMap[v.Name]; !exists {
			changes = append(changes, &ViewChange{ChangeType: CreateView, View: v})
		} else if normalizeSQL(existing.Definition) != normalizeSQL(v.Definition) {
			oldView := existing
			changes = append(changes, &ViewChange{ChangeType: AlterView, View: v, OldView: &oldView})
		}
	}

	for _, v := range current {
		if _, exists := desiredMap[v.Name]; !exists {
			oldView := v
			changes = append(changes, &ViewChange{ChangeType: DropView, View: v, OldView: &oldView})
		}
	}

	return changes
}
