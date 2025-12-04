package diff

import (
	"fmt"
	"strings"

	"shrugged/internal/parser"
)

type PublicationChange struct {
	ChangeType     ChangeType
	Publication    parser.Publication
	OldPublication *parser.Publication
}

func (c *PublicationChange) SQL() string {
	switch c.ChangeType {
	case CreatePublication:
		sql := fmt.Sprintf("CREATE PUBLICATION %s", quoteIdent(c.Publication.Name))
		if c.Publication.AllTables {
			sql += " FOR ALL TABLES"
		} else if len(c.Publication.Tables) > 0 {
			sql += " FOR TABLE " + strings.Join(c.Publication.Tables, ", ")
		}
		return sql + ";"
	case DropPublication:
		return fmt.Sprintf("DROP PUBLICATION %s;", quoteIdent(c.Publication.Name))
	}
	return ""
}

func (c *PublicationChange) DownSQL() string {
	switch c.ChangeType {
	case CreatePublication:
		return fmt.Sprintf("DROP PUBLICATION %s;", quoteIdent(c.Publication.Name))
	case DropPublication:
		if c.OldPublication != nil {
			sql := fmt.Sprintf("CREATE PUBLICATION %s", quoteIdent(c.OldPublication.Name))
			if c.OldPublication.AllTables {
				sql += " FOR ALL TABLES"
			} else if len(c.OldPublication.Tables) > 0 {
				sql += " FOR TABLE " + strings.Join(c.OldPublication.Tables, ", ")
			}
			return sql + ";"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped publication %s", c.Publication.Name)
	}
	return ""
}

func (c *PublicationChange) Type() ChangeType   { return c.ChangeType }
func (c *PublicationChange) ObjectName() string { return c.Publication.Name }
func (c *PublicationChange) IsReversible() bool {
	if c.ChangeType == CreatePublication {
		return true
	}
	return c.OldPublication != nil
}

func comparePublications(current, desired []parser.Publication) []Change {
	var changes []Change
	currentMap := make(map[string]parser.Publication)
	for _, p := range current {
		currentMap[p.Name] = p
	}
	for _, p := range desired {
		if _, exists := currentMap[p.Name]; !exists {
			changes = append(changes, &PublicationChange{ChangeType: CreatePublication, Publication: p})
		}
	}
	desiredMap := make(map[string]bool)
	for _, p := range desired {
		desiredMap[p.Name] = true
	}
	for _, p := range current {
		if !desiredMap[p.Name] {
			oldPub := p
			changes = append(changes, &PublicationChange{ChangeType: DropPublication, Publication: p, OldPublication: &oldPub})
		}
	}
	return changes
}
