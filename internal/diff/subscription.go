package diff

import (
	"fmt"

	"github.com/terminally-online/shrugged/internal/parser"
)

type SubscriptionChange struct {
	ChangeType      ChangeType
	Subscription    parser.Subscription
	OldSubscription *parser.Subscription
}

func (c *SubscriptionChange) SQL() string {
	switch c.ChangeType {
	case CreateSubscription:
		sql := fmt.Sprintf("CREATE SUBSCRIPTION %s CONNECTION '%s' PUBLICATION %s",
			quoteIdent(c.Subscription.Name), c.Subscription.ConnInfo, c.Subscription.Publication)
		if !c.Subscription.Enabled {
			sql += " WITH (enabled = false)"
		}
		return sql + ";"
	case DropSubscription:
		return fmt.Sprintf("DROP SUBSCRIPTION %s;", quoteIdent(c.Subscription.Name))
	}
	return ""
}

func (c *SubscriptionChange) DownSQL() string {
	switch c.ChangeType {
	case CreateSubscription:
		return fmt.Sprintf("DROP SUBSCRIPTION %s;", quoteIdent(c.Subscription.Name))
	case DropSubscription:
		if c.OldSubscription != nil {
			sql := fmt.Sprintf("CREATE SUBSCRIPTION %s CONNECTION '%s' PUBLICATION %s",
				quoteIdent(c.OldSubscription.Name), c.OldSubscription.ConnInfo, c.OldSubscription.Publication)
			if !c.OldSubscription.Enabled {
				sql += " WITH (enabled = false)"
			}
			return sql + ";"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped subscription %s", c.Subscription.Name)
	}
	return ""
}

func (c *SubscriptionChange) Type() ChangeType   { return c.ChangeType }
func (c *SubscriptionChange) ObjectName() string { return c.Subscription.Name }
func (c *SubscriptionChange) IsReversible() bool {
	if c.ChangeType == CreateSubscription {
		return true
	}
	return c.OldSubscription != nil
}

func compareSubscriptions(current, desired []parser.Subscription) []Change {
	var changes []Change
	currentMap := make(map[string]parser.Subscription)
	for _, s := range current {
		currentMap[s.Name] = s
	}
	for _, s := range desired {
		if _, exists := currentMap[s.Name]; !exists {
			changes = append(changes, &SubscriptionChange{ChangeType: CreateSubscription, Subscription: s})
		}
	}
	desiredMap := make(map[string]bool)
	for _, s := range desired {
		desiredMap[s.Name] = true
	}
	for _, s := range current {
		if !desiredMap[s.Name] {
			oldSub := s
			changes = append(changes, &SubscriptionChange{ChangeType: DropSubscription, Subscription: s, OldSubscription: &oldSub})
		}
	}
	return changes
}
