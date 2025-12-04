package diff

import (
	"fmt"
	"strings"

	"shrugged/internal/parser"
)

type EventTriggerChange struct {
	ChangeType      ChangeType
	EventTrigger    parser.EventTrigger
	OldEventTrigger *parser.EventTrigger
}

func (c *EventTriggerChange) SQL() string {
	switch c.ChangeType {
	case CreateEventTrigger:
		sql := fmt.Sprintf("CREATE EVENT TRIGGER %s ON %s",
			quoteIdent(c.EventTrigger.Name),
			c.EventTrigger.Event)
		if len(c.EventTrigger.Tags) > 0 {
			var quotedTags []string
			for _, t := range c.EventTrigger.Tags {
				quotedTags = append(quotedTags, quoteLiteral(t))
			}
			sql += fmt.Sprintf(" WHEN TAG IN (%s)", strings.Join(quotedTags, ", "))
		}
		sql += fmt.Sprintf(" EXECUTE FUNCTION %s();", c.EventTrigger.Function)
		return sql
	case DropEventTrigger:
		return fmt.Sprintf("DROP EVENT TRIGGER %s;", quoteIdent(c.EventTrigger.Name))
	}
	return ""
}

func (c *EventTriggerChange) DownSQL() string {
	switch c.ChangeType {
	case CreateEventTrigger:
		return fmt.Sprintf("DROP EVENT TRIGGER %s;", quoteIdent(c.EventTrigger.Name))
	case DropEventTrigger:
		if c.OldEventTrigger != nil {
			sql := fmt.Sprintf("CREATE EVENT TRIGGER %s ON %s",
				quoteIdent(c.OldEventTrigger.Name),
				c.OldEventTrigger.Event)
			if len(c.OldEventTrigger.Tags) > 0 {
				var quotedTags []string
				for _, t := range c.OldEventTrigger.Tags {
					quotedTags = append(quotedTags, quoteLiteral(t))
				}
				sql += fmt.Sprintf(" WHEN TAG IN (%s)", strings.Join(quotedTags, ", "))
			}
			sql += fmt.Sprintf(" EXECUTE FUNCTION %s();", c.OldEventTrigger.Function)
			return sql
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped event trigger %s", c.EventTrigger.Name)
	}
	return ""
}

func (c *EventTriggerChange) Type() ChangeType   { return c.ChangeType }
func (c *EventTriggerChange) ObjectName() string { return c.EventTrigger.Name }

func (c *EventTriggerChange) IsReversible() bool {
	switch c.ChangeType {
	case CreateEventTrigger:
		return true
	case DropEventTrigger:
		return c.OldEventTrigger != nil
	}
	return false
}

func compareEventTriggers(current, desired []parser.EventTrigger) []Change {
	var changes []Change
	currentMap := make(map[string]parser.EventTrigger)
	for _, e := range current {
		currentMap[e.Name] = e
	}
	for _, e := range desired {
		if _, exists := currentMap[e.Name]; !exists {
			changes = append(changes, &EventTriggerChange{ChangeType: CreateEventTrigger, EventTrigger: e})
		}
	}
	desiredMap := make(map[string]bool)
	for _, e := range desired {
		desiredMap[e.Name] = true
	}
	for _, e := range current {
		if !desiredMap[e.Name] {
			oldET := e
			changes = append(changes, &EventTriggerChange{ChangeType: DropEventTrigger, EventTrigger: e, OldEventTrigger: &oldET})
		}
	}
	return changes
}
