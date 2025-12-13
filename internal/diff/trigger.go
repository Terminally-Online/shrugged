package diff

import (
	"fmt"
	"strings"

	"github.com/terminally-online/shrugged/internal/parser"
)

type TriggerChange struct {
	ChangeType ChangeType
	Trigger    parser.Trigger
	OldTrigger *parser.Trigger
}

func (c *TriggerChange) SQL() string {
	switch c.ChangeType {
	case CreateTrigger:
		return generateCreateTrigger(c.Trigger)
	case DropTrigger:
		return fmt.Sprintf("DROP TRIGGER %s ON %s;", quoteIdent(c.Trigger.Name), quoteIdent(c.Trigger.Table))
	}
	return ""
}

func (c *TriggerChange) DownSQL() string {
	switch c.ChangeType {
	case CreateTrigger:
		return fmt.Sprintf("DROP TRIGGER %s ON %s;", quoteIdent(c.Trigger.Name), quoteIdent(c.Trigger.Table))
	case DropTrigger:
		if c.OldTrigger != nil {
			return generateCreateTrigger(*c.OldTrigger)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped trigger %s", c.Trigger.Name)
	}
	return ""
}

func (c *TriggerChange) Type() ChangeType {
	return c.ChangeType
}

func (c *TriggerChange) ObjectName() string {
	return c.Trigger.Name
}

func (c *TriggerChange) IsReversible() bool {
	if c.ChangeType == DropTrigger && c.OldTrigger == nil {
		return false
	}
	return true
}

func compareTriggers(current, desired []parser.Trigger) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Trigger)
	for _, t := range current {
		currentMap[t.Name] = t
	}

	desiredMap := make(map[string]parser.Trigger)
	for _, t := range desired {
		desiredMap[t.Name] = t
	}

	for _, t := range desired {
		if _, exists := currentMap[t.Name]; !exists {
			changes = append(changes, &TriggerChange{ChangeType: CreateTrigger, Trigger: t})
		}
	}

	for _, t := range current {
		if _, exists := desiredMap[t.Name]; !exists {
			oldTrig := t
			changes = append(changes, &TriggerChange{ChangeType: DropTrigger, Trigger: t, OldTrigger: &oldTrig})
		}
	}

	return changes
}

func generateCreateTrigger(t parser.Trigger) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TRIGGER %s ", quoteIdent(t.Name)))
	sb.WriteString(fmt.Sprintf("%s %s ", t.Timing, strings.Join(t.Events, " OR ")))
	sb.WriteString(fmt.Sprintf("ON %s ", quoteIdent(t.Table)))
	sb.WriteString(fmt.Sprintf("FOR EACH %s ", t.ForEach))

	if t.When != "" {
		sb.WriteString(fmt.Sprintf("WHEN (%s) ", t.When))
	}

	sb.WriteString(fmt.Sprintf("EXECUTE FUNCTION %s();", t.Function))

	return sb.String()
}
