package diff

import (
	"fmt"
	"strings"

	"github.com/terminally-online/shrugged/internal/parser"
)

type SequenceChange struct {
	ChangeType  ChangeType
	Sequence    parser.Sequence
	OldSequence *parser.Sequence
}

func (c *SequenceChange) SQL() string {
	switch c.ChangeType {
	case CreateSequence:
		return generateCreateSequence(c.Sequence)
	case DropSequence:
		return fmt.Sprintf("DROP SEQUENCE %s;", qualifiedName(c.Sequence.Schema, c.Sequence.Name))
	case AlterSequence:
		return fmt.Sprintf("ALTER SEQUENCE %s;", qualifiedName(c.Sequence.Schema, c.Sequence.Name))
	}
	return ""
}

func (c *SequenceChange) DownSQL() string {
	switch c.ChangeType {
	case CreateSequence:
		return fmt.Sprintf("DROP SEQUENCE %s;", qualifiedName(c.Sequence.Schema, c.Sequence.Name))
	case DropSequence:
		if c.OldSequence != nil {
			return generateCreateSequence(*c.OldSequence)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped sequence %s", c.Sequence.Name)
	case AlterSequence:
		if c.OldSequence != nil {
			return generateCreateSequence(*c.OldSequence)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore previous sequence %s", c.Sequence.Name)
	}
	return ""
}

func (c *SequenceChange) Type() ChangeType {
	return c.ChangeType
}

func (c *SequenceChange) ObjectName() string {
	return c.Sequence.Name
}

func (c *SequenceChange) IsReversible() bool {
	switch c.ChangeType {
	case CreateSequence:
		return true
	case DropSequence, AlterSequence:
		return c.OldSequence != nil
	}
	return false
}

func compareSequences(current, desired []parser.Sequence) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Sequence)
	for _, s := range current {
		currentMap[s.Name] = s
	}

	desiredMap := make(map[string]parser.Sequence)
	for _, s := range desired {
		desiredMap[s.Name] = s
	}

	for _, s := range desired {
		if _, exists := currentMap[s.Name]; !exists {
			changes = append(changes, &SequenceChange{ChangeType: CreateSequence, Sequence: s})
		}
	}

	for _, s := range current {
		if _, exists := desiredMap[s.Name]; !exists {
			oldSeq := s
			changes = append(changes, &SequenceChange{ChangeType: DropSequence, Sequence: s, OldSequence: &oldSeq})
		}
	}

	return changes
}

func generateCreateSequence(s parser.Sequence) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE SEQUENCE %s", qualifiedName(s.Schema, s.Name)))

	if s.Start != 0 {
		sb.WriteString(fmt.Sprintf(" START %d", s.Start))
	}
	if s.Increment != 0 {
		sb.WriteString(fmt.Sprintf(" INCREMENT %d", s.Increment))
	}
	if s.MinValue != 0 {
		sb.WriteString(fmt.Sprintf(" MINVALUE %d", s.MinValue))
	}
	if s.MaxValue != 0 {
		sb.WriteString(fmt.Sprintf(" MAXVALUE %d", s.MaxValue))
	}
	if s.Cache != 0 {
		sb.WriteString(fmt.Sprintf(" CACHE %d", s.Cache))
	}
	if s.Cycle {
		sb.WriteString(" CYCLE")
	}

	sb.WriteString(";")
	return sb.String()
}
