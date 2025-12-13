package diff

import (
	"fmt"

	"github.com/terminally-online/shrugged/internal/parser"
)

type DomainChange struct {
	ChangeType ChangeType
	Domain     parser.Domain
	OldDomain  *parser.Domain
}

func (c *DomainChange) SQL() string {
	domainName := qualifiedName(c.Domain.Schema, c.Domain.Name)
	switch c.ChangeType {
	case CreateDomain:
		sql := fmt.Sprintf("CREATE DOMAIN %s AS %s", domainName, c.Domain.Type)
		if c.Domain.Collation != "" {
			sql += fmt.Sprintf(" COLLATE %s", quoteIdent(c.Domain.Collation))
		}
		if c.Domain.Default != "" {
			sql += fmt.Sprintf(" DEFAULT %s", c.Domain.Default)
		}
		if c.Domain.NotNull {
			sql += " NOT NULL"
		}
		if c.Domain.Check != "" {
			sql += fmt.Sprintf(" %s", c.Domain.Check)
		}
		return sql + ";"
	case DropDomain:
		return fmt.Sprintf("DROP DOMAIN %s;", domainName)
	}
	return ""
}

func (c *DomainChange) DownSQL() string {
	domainName := qualifiedName(c.Domain.Schema, c.Domain.Name)
	switch c.ChangeType {
	case CreateDomain:
		return fmt.Sprintf("DROP DOMAIN %s;", domainName)
	case DropDomain:
		if c.OldDomain != nil {
			sql := fmt.Sprintf("CREATE DOMAIN %s AS %s", qualifiedName(c.OldDomain.Schema, c.OldDomain.Name), c.OldDomain.Type)
			if c.OldDomain.Collation != "" {
				sql += fmt.Sprintf(" COLLATE %s", quoteIdent(c.OldDomain.Collation))
			}
			if c.OldDomain.Default != "" {
				sql += fmt.Sprintf(" DEFAULT %s", c.OldDomain.Default)
			}
			if c.OldDomain.NotNull {
				sql += " NOT NULL"
			}
			if c.OldDomain.Check != "" {
				sql += fmt.Sprintf(" %s", c.OldDomain.Check)
			}
			return sql + ";"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped domain %s", c.Domain.Name)
	}
	return ""
}

func (c *DomainChange) Type() ChangeType   { return c.ChangeType }
func (c *DomainChange) ObjectName() string { return c.Domain.Name }
func (c *DomainChange) IsReversible() bool {
	if c.ChangeType == CreateDomain {
		return true
	}
	return c.OldDomain != nil
}

func compareDomains(current, desired []parser.Domain) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Domain)
	for _, d := range current {
		currentMap[objectKey(d.Schema, d.Name)] = d
	}

	desiredMap := make(map[string]parser.Domain)
	for _, d := range desired {
		desiredMap[objectKey(d.Schema, d.Name)] = d
	}

	for _, d := range desired {
		if _, exists := currentMap[objectKey(d.Schema, d.Name)]; !exists {
			changes = append(changes, &DomainChange{ChangeType: CreateDomain, Domain: d})
		}
	}

	for _, d := range current {
		if _, exists := desiredMap[objectKey(d.Schema, d.Name)]; !exists {
			oldDom := d
			changes = append(changes, &DomainChange{ChangeType: DropDomain, Domain: d, OldDomain: &oldDom})
		}
	}

	return changes
}
