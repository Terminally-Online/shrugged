package diff

import (
	"fmt"

	"github.com/terminally-online/shrugged/internal/parser"
)

type AggregateChange struct {
	ChangeType   ChangeType
	Aggregate    parser.Aggregate
	OldAggregate *parser.Aggregate
}

func (c *AggregateChange) SQL() string {
	switch c.ChangeType {
	case CreateAggregate:
		sql := fmt.Sprintf("CREATE AGGREGATE %s(%s) (\n    SFUNC = %s,\n    STYPE = %s",
			quoteIdent(c.Aggregate.Name), c.Aggregate.Args, c.Aggregate.SFunc, c.Aggregate.SType)
		if c.Aggregate.FinalFunc != "" {
			sql += fmt.Sprintf(",\n    FINALFUNC = %s", c.Aggregate.FinalFunc)
		}
		if c.Aggregate.InitCond != "" {
			sql += fmt.Sprintf(",\n    INITCOND = '%s'", c.Aggregate.InitCond)
		}
		if c.Aggregate.SortOp != "" {
			sql += fmt.Sprintf(",\n    SORTOP = %s", c.Aggregate.SortOp)
		}
		return sql + "\n);"
	case DropAggregate:
		return fmt.Sprintf("DROP AGGREGATE %s(%s);", quoteIdent(c.Aggregate.Name), c.Aggregate.Args)
	}
	return ""
}

func (c *AggregateChange) DownSQL() string {
	switch c.ChangeType {
	case CreateAggregate:
		return fmt.Sprintf("DROP AGGREGATE %s(%s);", quoteIdent(c.Aggregate.Name), c.Aggregate.Args)
	case DropAggregate:
		if c.OldAggregate != nil {
			sql := fmt.Sprintf("CREATE AGGREGATE %s(%s) (\n    SFUNC = %s,\n    STYPE = %s",
				quoteIdent(c.OldAggregate.Name), c.OldAggregate.Args, c.OldAggregate.SFunc, c.OldAggregate.SType)
			if c.OldAggregate.FinalFunc != "" {
				sql += fmt.Sprintf(",\n    FINALFUNC = %s", c.OldAggregate.FinalFunc)
			}
			if c.OldAggregate.InitCond != "" {
				sql += fmt.Sprintf(",\n    INITCOND = '%s'", c.OldAggregate.InitCond)
			}
			if c.OldAggregate.SortOp != "" {
				sql += fmt.Sprintf(",\n    SORTOP = %s", c.OldAggregate.SortOp)
			}
			return sql + "\n);"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped aggregate %s", c.Aggregate.Name)
	}
	return ""
}

func (c *AggregateChange) Type() ChangeType   { return c.ChangeType }
func (c *AggregateChange) ObjectName() string { return c.Aggregate.Name }
func (c *AggregateChange) IsReversible() bool {
	if c.ChangeType == CreateAggregate {
		return true
	}
	return c.OldAggregate != nil
}

func compareAggregates(current, desired []parser.Aggregate) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Aggregate)
	for _, a := range current {
		currentMap[a.Name] = a
	}

	desiredMap := make(map[string]parser.Aggregate)
	for _, a := range desired {
		desiredMap[a.Name] = a
	}

	for _, a := range desired {
		if _, exists := currentMap[a.Name]; !exists {
			changes = append(changes, &AggregateChange{ChangeType: CreateAggregate, Aggregate: a})
		}
	}

	for _, a := range current {
		if _, exists := desiredMap[a.Name]; !exists {
			oldAgg := a
			changes = append(changes, &AggregateChange{ChangeType: DropAggregate, Aggregate: a, OldAggregate: &oldAgg})
		}
	}

	return changes
}
