package diff

import (
	"fmt"
	"strings"

	"github.com/terminally-online/shrugged/internal/parser"
)

type RoleChange struct {
	ChangeType ChangeType
	Role       parser.Role
	OldRole    *parser.Role
}

func (c *RoleChange) SQL() string {
	switch c.ChangeType {
	case CreateRole:
		sql := fmt.Sprintf("CREATE ROLE %s", quoteIdent(c.Role.Name))
		var opts []string
		if c.Role.SuperUser {
			opts = append(opts, "SUPERUSER")
		}
		if c.Role.CreateDB {
			opts = append(opts, "CREATEDB")
		}
		if c.Role.CreateRole {
			opts = append(opts, "CREATEROLE")
		}
		if !c.Role.Inherit {
			opts = append(opts, "NOINHERIT")
		}
		if c.Role.Login {
			opts = append(opts, "LOGIN")
		}
		if c.Role.Replication {
			opts = append(opts, "REPLICATION")
		}
		if c.Role.BypassRLS {
			opts = append(opts, "BYPASSRLS")
		}
		if c.Role.ConnectionLimit >= 0 {
			opts = append(opts, fmt.Sprintf("CONNECTION LIMIT %d", c.Role.ConnectionLimit))
		}
		if c.Role.ValidUntil != "" {
			opts = append(opts, fmt.Sprintf("VALID UNTIL '%s'", c.Role.ValidUntil))
		}
		if len(opts) > 0 {
			sql += " WITH " + strings.Join(opts, " ")
		}
		sql += ";"
		if len(c.Role.InRoles) > 0 {
			for _, r := range c.Role.InRoles {
				sql += fmt.Sprintf("\nGRANT %s TO %s;", quoteIdent(r), quoteIdent(c.Role.Name))
			}
		}
		return sql
	case DropRole:
		return fmt.Sprintf("DROP ROLE %s;", quoteIdent(c.Role.Name))
	case AlterRole:
		var stmts []string
		if c.OldRole != nil {
			if c.Role.SuperUser != c.OldRole.SuperUser {
				if c.Role.SuperUser {
					stmts = append(stmts, fmt.Sprintf("ALTER ROLE %s SUPERUSER;", quoteIdent(c.Role.Name)))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER ROLE %s NOSUPERUSER;", quoteIdent(c.Role.Name)))
				}
			}
			if c.Role.Login != c.OldRole.Login {
				if c.Role.Login {
					stmts = append(stmts, fmt.Sprintf("ALTER ROLE %s LOGIN;", quoteIdent(c.Role.Name)))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER ROLE %s NOLOGIN;", quoteIdent(c.Role.Name)))
				}
			}
		}
		return strings.Join(stmts, "\n")
	}
	return ""
}

func (c *RoleChange) DownSQL() string {
	switch c.ChangeType {
	case CreateRole:
		return fmt.Sprintf("DROP ROLE %s;", quoteIdent(c.Role.Name))
	case DropRole:
		if c.OldRole != nil {
			sql := fmt.Sprintf("CREATE ROLE %s", quoteIdent(c.OldRole.Name))
			var opts []string
			if c.OldRole.SuperUser {
				opts = append(opts, "SUPERUSER")
			}
			if c.OldRole.CreateDB {
				opts = append(opts, "CREATEDB")
			}
			if c.OldRole.CreateRole {
				opts = append(opts, "CREATEROLE")
			}
			if !c.OldRole.Inherit {
				opts = append(opts, "NOINHERIT")
			}
			if c.OldRole.Login {
				opts = append(opts, "LOGIN")
			}
			if c.OldRole.Replication {
				opts = append(opts, "REPLICATION")
			}
			if c.OldRole.BypassRLS {
				opts = append(opts, "BYPASSRLS")
			}
			if len(opts) > 0 {
				sql += " WITH " + strings.Join(opts, " ")
			}
			sql += ";"
			for _, r := range c.OldRole.InRoles {
				sql += fmt.Sprintf("\nGRANT %s TO %s;", quoteIdent(r), quoteIdent(c.OldRole.Name))
			}
			return sql
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped role %s", c.Role.Name)
	case AlterRole:
		if c.OldRole != nil {
			var stmts []string
			if c.Role.SuperUser != c.OldRole.SuperUser {
				if c.OldRole.SuperUser {
					stmts = append(stmts, fmt.Sprintf("ALTER ROLE %s SUPERUSER;", quoteIdent(c.Role.Name)))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER ROLE %s NOSUPERUSER;", quoteIdent(c.Role.Name)))
				}
			}
			if c.Role.Login != c.OldRole.Login {
				if c.OldRole.Login {
					stmts = append(stmts, fmt.Sprintf("ALTER ROLE %s LOGIN;", quoteIdent(c.Role.Name)))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER ROLE %s NOLOGIN;", quoteIdent(c.Role.Name)))
				}
			}
			return strings.Join(stmts, "\n")
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore previous role settings for %s", c.Role.Name)
	}
	return ""
}

func (c *RoleChange) Type() ChangeType   { return c.ChangeType }
func (c *RoleChange) ObjectName() string { return c.Role.Name }

func (c *RoleChange) IsReversible() bool {
	switch c.ChangeType {
	case CreateRole:
		return true
	case DropRole, AlterRole:
		return c.OldRole != nil
	}
	return false
}

func compareRoles(current, desired []parser.Role) []Change {
	var changes []Change
	currentMap := make(map[string]parser.Role)
	for _, r := range current {
		currentMap[r.Name] = r
	}
	for _, r := range desired {
		if existing, exists := currentMap[r.Name]; !exists {
			changes = append(changes, &RoleChange{ChangeType: CreateRole, Role: r})
		} else if rolesNeedAlter(existing, r) {
			changes = append(changes, &RoleChange{ChangeType: AlterRole, Role: r, OldRole: &existing})
		}
	}
	desiredMap := make(map[string]bool)
	for _, r := range desired {
		desiredMap[r.Name] = true
	}
	for _, r := range current {
		if !desiredMap[r.Name] {
			oldRole := r
			changes = append(changes, &RoleChange{ChangeType: DropRole, Role: r, OldRole: &oldRole})
		}
	}
	return changes
}

func rolesNeedAlter(current, desired parser.Role) bool {
	return current.SuperUser != desired.SuperUser ||
		current.CreateDB != desired.CreateDB ||
		current.CreateRole != desired.CreateRole ||
		current.Inherit != desired.Inherit ||
		current.Login != desired.Login ||
		current.Replication != desired.Replication ||
		current.BypassRLS != desired.BypassRLS
}

type RoleGrantChange struct {
	ChangeType ChangeType
	RoleGrant  parser.RoleGrant
}

func quoteGrantee(grantee string) string {
	if strings.ToUpper(grantee) == "PUBLIC" {
		return "PUBLIC"
	}
	return quoteIdent(grantee)
}

func (c *RoleGrantChange) SQL() string {
	objectRef := qualifiedName(c.RoleGrant.Schema, c.RoleGrant.ObjectName)
	objectTypePrefix := ""
	switch c.RoleGrant.ObjectType {
	case "TYPE":
		objectTypePrefix = "TYPE "
	case "FUNCTION":
		objectTypePrefix = "FUNCTION "
	}
	switch c.ChangeType {
	case CreateRoleGrant:
		sql := fmt.Sprintf("GRANT %s ON %s%s TO %s",
			c.RoleGrant.Privilege,
			objectTypePrefix,
			objectRef,
			quoteGrantee(c.RoleGrant.Grantee))
		if c.RoleGrant.WithGrant {
			sql += " WITH GRANT OPTION"
		}
		return sql + ";"
	case DropRoleGrant:
		return fmt.Sprintf("REVOKE %s ON %s%s FROM %s;",
			c.RoleGrant.Privilege,
			objectTypePrefix,
			objectRef,
			quoteGrantee(c.RoleGrant.Grantee))
	}
	return ""
}

func (c *RoleGrantChange) DownSQL() string {
	objectRef := qualifiedName(c.RoleGrant.Schema, c.RoleGrant.ObjectName)
	objectTypePrefix := ""
	switch c.RoleGrant.ObjectType {
	case "TYPE":
		objectTypePrefix = "TYPE "
	case "FUNCTION":
		objectTypePrefix = "FUNCTION "
	}
	switch c.ChangeType {
	case CreateRoleGrant:
		return fmt.Sprintf("REVOKE %s ON %s%s FROM %s;",
			c.RoleGrant.Privilege,
			objectTypePrefix,
			objectRef,
			quoteGrantee(c.RoleGrant.Grantee))
	case DropRoleGrant:
		sql := fmt.Sprintf("GRANT %s ON %s%s TO %s",
			c.RoleGrant.Privilege,
			objectTypePrefix,
			objectRef,
			quoteGrantee(c.RoleGrant.Grantee))
		if c.RoleGrant.WithGrant {
			sql += " WITH GRANT OPTION"
		}
		return sql + ";"
	}
	return ""
}

func (c *RoleGrantChange) Type() ChangeType { return c.ChangeType }
func (c *RoleGrantChange) ObjectName() string {
	return c.RoleGrant.ObjectName + ":" + c.RoleGrant.Grantee
}
func (c *RoleGrantChange) IsReversible() bool { return true }

func compareRoleGrants(current, desired []parser.RoleGrant) []Change {
	var changes []Change

	grantKey := func(g parser.RoleGrant) string {
		return fmt.Sprintf("%s:%s:%s:%s", g.Schema, g.ObjectName, g.Privilege, g.Grantee)
	}

	currentMap := make(map[string]parser.RoleGrant)
	for _, g := range current {
		currentMap[grantKey(g)] = g
	}
	for _, g := range desired {
		if _, exists := currentMap[grantKey(g)]; !exists {
			changes = append(changes, &RoleGrantChange{ChangeType: CreateRoleGrant, RoleGrant: g})
		}
	}
	desiredMap := make(map[string]bool)
	for _, g := range desired {
		desiredMap[grantKey(g)] = true
	}
	for _, g := range current {
		if !desiredMap[grantKey(g)] {
			changes = append(changes, &RoleGrantChange{ChangeType: DropRoleGrant, RoleGrant: g})
		}
	}
	return changes
}

type DefaultPrivilegeChange struct {
	ChangeType       ChangeType
	DefaultPrivilege parser.DefaultPrivilege
}

func (c *DefaultPrivilegeChange) SQL() string {
	switch c.ChangeType {
	case CreateDefaultPrivilege:
		privs := strings.Join(c.DefaultPrivilege.Privileges, ", ")
		return fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA %s GRANT %s ON %s TO %s;",
			quoteIdent(c.DefaultPrivilege.Role),
			quoteIdent(c.DefaultPrivilege.Schema),
			privs,
			c.DefaultPrivilege.ObjectType,
			quoteIdent(c.DefaultPrivilege.Grantee))
	case DropDefaultPrivilege:
		privs := strings.Join(c.DefaultPrivilege.Privileges, ", ")
		return fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA %s REVOKE %s ON %s FROM %s;",
			quoteIdent(c.DefaultPrivilege.Role),
			quoteIdent(c.DefaultPrivilege.Schema),
			privs,
			c.DefaultPrivilege.ObjectType,
			quoteIdent(c.DefaultPrivilege.Grantee))
	}
	return ""
}

func (c *DefaultPrivilegeChange) DownSQL() string {
	privs := strings.Join(c.DefaultPrivilege.Privileges, ", ")
	switch c.ChangeType {
	case CreateDefaultPrivilege:
		return fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA %s REVOKE %s ON %s FROM %s;",
			quoteIdent(c.DefaultPrivilege.Role),
			quoteIdent(c.DefaultPrivilege.Schema),
			privs,
			c.DefaultPrivilege.ObjectType,
			quoteIdent(c.DefaultPrivilege.Grantee))
	case DropDefaultPrivilege:
		return fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA %s GRANT %s ON %s TO %s;",
			quoteIdent(c.DefaultPrivilege.Role),
			quoteIdent(c.DefaultPrivilege.Schema),
			privs,
			c.DefaultPrivilege.ObjectType,
			quoteIdent(c.DefaultPrivilege.Grantee))
	}
	return ""
}

func (c *DefaultPrivilegeChange) Type() ChangeType { return c.ChangeType }
func (c *DefaultPrivilegeChange) ObjectName() string {
	return c.DefaultPrivilege.Schema + ":" + c.DefaultPrivilege.Role
}
func (c *DefaultPrivilegeChange) IsReversible() bool { return true }

func compareDefaultPrivileges(current, desired []parser.DefaultPrivilege) []Change {
	var changes []Change

	privKey := func(p parser.DefaultPrivilege) string {
		return fmt.Sprintf("%s:%s:%s:%s", p.Schema, p.Role, p.ObjectType, p.Grantee)
	}

	currentMap := make(map[string]parser.DefaultPrivilege)
	for _, p := range current {
		currentMap[privKey(p)] = p
	}
	for _, p := range desired {
		if _, exists := currentMap[privKey(p)]; !exists {
			changes = append(changes, &DefaultPrivilegeChange{ChangeType: CreateDefaultPrivilege, DefaultPrivilege: p})
		}
	}
	desiredMap := make(map[string]bool)
	for _, p := range desired {
		desiredMap[privKey(p)] = true
	}
	for _, p := range current {
		if !desiredMap[privKey(p)] {
			changes = append(changes, &DefaultPrivilegeChange{ChangeType: DropDefaultPrivilege, DefaultPrivilege: p})
		}
	}
	return changes
}
