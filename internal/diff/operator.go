package diff

import (
	"fmt"

	"shrugged/internal/parser"
)

type OperatorChange struct {
	ChangeType  ChangeType
	Operator    parser.Operator
	OldOperator *parser.Operator
}

func (c *OperatorChange) SQL() string {
	switch c.ChangeType {
	case CreateOperator:
		sql := fmt.Sprintf("CREATE OPERATOR %s (\n    FUNCTION = %s",
			c.Operator.Name, c.Operator.Procedure)
		if c.Operator.LeftType != "NONE" {
			sql += fmt.Sprintf(",\n    LEFTARG = %s", c.Operator.LeftType)
		}
		if c.Operator.RightType != "NONE" {
			sql += fmt.Sprintf(",\n    RIGHTARG = %s", c.Operator.RightType)
		}
		if c.Operator.Commutator != "" {
			sql += fmt.Sprintf(",\n    COMMUTATOR = %s", c.Operator.Commutator)
		}
		if c.Operator.Negator != "" {
			sql += fmt.Sprintf(",\n    NEGATOR = %s", c.Operator.Negator)
		}
		return sql + "\n);"
	case DropOperator:
		left := c.Operator.LeftType
		right := c.Operator.RightType
		if left == "NONE" {
			left = "NONE"
		}
		if right == "NONE" {
			right = "NONE"
		}
		return fmt.Sprintf("DROP OPERATOR %s (%s, %s);", c.Operator.Name, left, right)
	}
	return ""
}

func (c *OperatorChange) DownSQL() string {
	switch c.ChangeType {
	case CreateOperator:
		left := c.Operator.LeftType
		right := c.Operator.RightType
		return fmt.Sprintf("DROP OPERATOR %s (%s, %s);", c.Operator.Name, left, right)
	case DropOperator:
		if c.OldOperator != nil {
			sql := fmt.Sprintf("CREATE OPERATOR %s (\n    FUNCTION = %s",
				c.OldOperator.Name, c.OldOperator.Procedure)
			if c.OldOperator.LeftType != "NONE" {
				sql += fmt.Sprintf(",\n    LEFTARG = %s", c.OldOperator.LeftType)
			}
			if c.OldOperator.RightType != "NONE" {
				sql += fmt.Sprintf(",\n    RIGHTARG = %s", c.OldOperator.RightType)
			}
			if c.OldOperator.Commutator != "" {
				sql += fmt.Sprintf(",\n    COMMUTATOR = %s", c.OldOperator.Commutator)
			}
			if c.OldOperator.Negator != "" {
				sql += fmt.Sprintf(",\n    NEGATOR = %s", c.OldOperator.Negator)
			}
			return sql + "\n);"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped operator %s", c.Operator.Name)
	}
	return ""
}

func (c *OperatorChange) Type() ChangeType   { return c.ChangeType }
func (c *OperatorChange) ObjectName() string { return c.Operator.Name }
func (c *OperatorChange) IsReversible() bool {
	if c.ChangeType == CreateOperator {
		return true
	}
	return c.OldOperator != nil
}

func compareOperators(current, desired []parser.Operator) []Change {
	var changes []Change
	currentMap := make(map[string]parser.Operator)
	for _, o := range current {
		key := fmt.Sprintf("%s(%s,%s)", o.Name, o.LeftType, o.RightType)
		currentMap[key] = o
	}
	for _, o := range desired {
		key := fmt.Sprintf("%s(%s,%s)", o.Name, o.LeftType, o.RightType)
		if _, exists := currentMap[key]; !exists {
			changes = append(changes, &OperatorChange{ChangeType: CreateOperator, Operator: o})
		}
	}
	desiredMap := make(map[string]bool)
	for _, o := range desired {
		key := fmt.Sprintf("%s(%s,%s)", o.Name, o.LeftType, o.RightType)
		desiredMap[key] = true
	}
	for _, o := range current {
		key := fmt.Sprintf("%s(%s,%s)", o.Name, o.LeftType, o.RightType)
		if !desiredMap[key] {
			oldOp := o
			changes = append(changes, &OperatorChange{ChangeType: DropOperator, Operator: o, OldOperator: &oldOp})
		}
	}
	return changes
}
