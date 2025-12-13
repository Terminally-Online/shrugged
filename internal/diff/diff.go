package diff

import (
	"strings"

	"github.com/terminally-online/shrugged/internal/parser"
)

type ChangeType int

const (
	CreateNamespace ChangeType = iota
	DropNamespace
	CreateExtension
	DropExtension
	CreateEnum
	DropEnum
	AlterEnum
	CreateDomain
	DropDomain
	AlterDomain
	CreateCompositeType
	DropCompositeType
	CreateSequence
	DropSequence
	AlterSequence
	CreateTable
	DropTable
	AlterTable
	CreateIndex
	DropIndex
	CreateView
	DropView
	AlterView
	CreateMaterializedView
	DropMaterializedView
	AlterMaterializedView
	CreateFunction
	DropFunction
	AlterFunction
	CreateProcedure
	DropProcedure
	AlterProcedure
	CreateAggregate
	DropAggregate
	CreateTrigger
	DropTrigger
	CreateEventTrigger
	DropEventTrigger
	CreateRule
	DropRule
	CreatePolicy
	DropPolicy
	CreateCollation
	DropCollation
	CreateTextSearchConfig
	DropTextSearchConfig
	CreatePublication
	DropPublication
	AlterPublication
	CreateSubscription
	DropSubscription
	AlterSubscription
	CreateForeignDataWrapper
	DropForeignDataWrapper
	AlterForeignDataWrapper
	CreateForeignServer
	DropForeignServer
	AlterForeignServer
	CreateForeignTable
	DropForeignTable
	AlterForeignTable
	CreateOperator
	DropOperator
	CreateRole
	DropRole
	AlterRole
	CreateRoleGrant
	DropRoleGrant
	CreateDefaultPrivilege
	DropDefaultPrivilege
	CreateComment
	DropComment
)

type Change interface {
	SQL() string
	DownSQL() string
	Type() ChangeType
	ObjectName() string
	IsReversible() bool
}

func Compare(current, desired *parser.Schema) []Change {
	var changes []Change

	changes = append(changes, compareNamespaces(current.Namespaces, desired.Namespaces)...)
	changes = append(changes, compareExtensions(current.Extensions, desired.Extensions)...)
	changes = append(changes, compareEnums(current.Enums, desired.Enums)...)
	changes = append(changes, compareDomains(current.Domains, desired.Domains)...)
	changes = append(changes, compareCompositeTypes(current.CompositeTypes, desired.CompositeTypes)...)
	changes = append(changes, compareSequences(current.Sequences, desired.Sequences)...)
	changes = append(changes, compareTables(current.Tables, desired.Tables)...)
	changes = append(changes, compareIndexes(current.Indexes, desired.Indexes)...)
	changes = append(changes, compareViews(current.Views, desired.Views)...)
	changes = append(changes, compareMaterializedViews(current.MaterializedViews, desired.MaterializedViews)...)
	changes = append(changes, compareFunctions(current.Functions, desired.Functions)...)
	changes = append(changes, compareProcedures(current.Procedures, desired.Procedures)...)
	changes = append(changes, compareAggregates(current.Aggregates, desired.Aggregates)...)
	changes = append(changes, compareTriggers(current.Triggers, desired.Triggers)...)
	changes = append(changes, compareEventTriggers(current.EventTriggers, desired.EventTriggers)...)
	changes = append(changes, compareRules(current.Rules, desired.Rules)...)
	changes = append(changes, comparePolicies(current.Policies, desired.Policies)...)
	changes = append(changes, compareCollations(current.Collations, desired.Collations)...)
	changes = append(changes, compareTextSearchConfigs(current.TextSearchConfigs, desired.TextSearchConfigs)...)
	changes = append(changes, comparePublications(current.Publications, desired.Publications)...)
	changes = append(changes, compareSubscriptions(current.Subscriptions, desired.Subscriptions)...)
	changes = append(changes, compareForeignDataWrappers(current.ForeignDataWrappers, desired.ForeignDataWrappers)...)
	changes = append(changes, compareForeignServers(current.ForeignServers, desired.ForeignServers)...)
	changes = append(changes, compareForeignTables(current.ForeignTables, desired.ForeignTables)...)
	changes = append(changes, compareOperators(current.Operators, desired.Operators)...)
	changes = append(changes, compareRoles(current.Roles, desired.Roles)...)
	changes = append(changes, compareRoleGrants(current.RoleGrants, desired.RoleGrants)...)
	changes = append(changes, compareDefaultPrivileges(current.DefaultPrivileges, desired.DefaultPrivileges)...)
	changes = append(changes, compareComments(current.Comments, desired.Comments)...)

	return changes
}

func quoteIdent(s string) string {
	if s == "" {
		return s
	}
	needsQuoting := false
	for i, r := range s {
		if i == 0 {
			if (r < 'a' || r > 'z') && r != '_' {
				needsQuoting = true
				break
			}
		} else {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
				needsQuoting = true
				break
			}
		}
	}
	reserved := map[string]bool{
		"all": true, "analyse": true, "analyze": true, "and": true, "any": true,
		"array": true, "as": true, "asc": true, "asymmetric": true, "both": true,
		"case": true, "cast": true, "check": true, "collate": true, "column": true,
		"constraint": true, "create": true, "current_catalog": true, "current_date": true,
		"current_role": true, "current_time": true, "current_timestamp": true,
		"current_user": true, "default": true, "deferrable": true, "desc": true,
		"distinct": true, "do": true, "else": true, "end": true, "except": true,
		"false": true, "fetch": true, "for": true, "foreign": true, "from": true,
		"grant": true, "group": true, "having": true, "in": true, "initially": true,
		"intersect": true, "into": true, "lateral": true, "leading": true, "limit": true,
		"localtime": true, "localtimestamp": true, "not": true, "null": true, "offset": true,
		"on": true, "only": true, "or": true, "order": true, "placing": true, "primary": true,
		"references": true, "returning": true, "select": true, "session_user": true,
		"some": true, "symmetric": true, "table": true, "then": true, "to": true,
		"trailing": true, "true": true, "union": true, "unique": true, "user": true,
		"using": true, "variadic": true, "when": true, "where": true, "window": true, "with": true,
	}
	if reserved[strings.ToLower(s)] {
		needsQuoting = true
	}
	if needsQuoting {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func qualifiedName(schema, name string) string {
	if schema == "" || schema == "public" {
		return quoteIdent(name)
	}
	return quoteIdent(schema) + "." + quoteIdent(name)
}

func objectKey(schema, name string) string {
	if schema == "" {
		schema = "public"
	}
	return schema + "." + name
}

func quoteIdents(names []string) []string {
	result := make([]string, len(names))
	for i, n := range names {
		result[i] = quoteIdent(n)
	}
	return result
}

func quoteLiteral(s string) string {
	escaped := strings.ReplaceAll(s, "'", "''")
	return "'" + escaped + "'"
}

func normalizeType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	t = strings.ReplaceAll(t, "character varying", "varchar")
	t = strings.ReplaceAll(t, "integer", "int4")
	t = strings.ReplaceAll(t, "bigint", "int8")
	t = strings.ReplaceAll(t, "smallint", "int2")
	t = strings.ReplaceAll(t, "boolean", "bool")
	t = strings.ReplaceAll(t, "double precision", "float8")
	t = strings.ReplaceAll(t, "real", "float4")
	t = strings.ReplaceAll(t, "timestamp without time zone", "timestamp")
	t = strings.ReplaceAll(t, "timestamp with time zone", "timestamptz")
	t = strings.ReplaceAll(t, "time without time zone", "time")
	t = strings.ReplaceAll(t, "time with time zone", "timetz")
	return t
}

func normalizeSQL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ";")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	s = strings.Join(lines, " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.ToLower(s)
}
