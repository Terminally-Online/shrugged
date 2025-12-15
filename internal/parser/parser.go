package parser

import (
	"fmt"
	"os"
	"strings"
)

type Schema struct {
	Namespaces          []Namespace
	Extensions          []Extension
	Enums               []Enum
	Domains             []Domain
	CompositeTypes      []CompositeType
	Sequences           []Sequence
	Tables              []Table
	Indexes             []Index
	Views               []View
	MaterializedViews   []MaterializedView
	Functions           []Function
	Procedures          []Procedure
	Aggregates          []Aggregate
	Triggers            []Trigger
	EventTriggers       []EventTrigger
	Rules               []Rule
	Policies            []Policy
	ForeignDataWrappers []ForeignDataWrapper
	ForeignServers      []ForeignServer
	ForeignTables       []ForeignTable
	TextSearchConfigs   []TextSearchConfig
	Publications        []Publication
	Subscriptions       []Subscription
	Operators           []Operator
	Collations          []Collation
	Comments            []Comment
	Roles               []Role
	RoleGrants          []RoleGrant
	DefaultPrivileges   []DefaultPrivilege
}

type Table struct {
	Schema         string
	Name           string
	Columns        []Column
	Constraints    []Constraint
	PartitionBy    string
	PartitionKey   string
	PartitionOf    string
	PartitionBound string
}

type Column struct {
	Name       string
	Type       string
	Nullable   bool
	Default    string
	PrimaryKey bool

	Identity          string
	IdentityStart     int64
	IdentityIncrement int64
	IdentityMinValue  *int64
	IdentityMaxValue  *int64
	IdentityCache     int64
	IdentityCycle     bool

	GeneratedAs   string
	GeneratedType string

	NotNullConstraintName string
}

type Constraint struct {
	Name       string
	Type       string
	Columns    []string
	RefTable   string
	RefColumns []string
	OnDelete   string
	OnUpdate   string
	Check      string

	ExclusionUsing     string
	ExclusionOperators map[string]string
	ExclusionWhere     string

	WithoutOverlaps bool
	PeriodColumn    string

	NotEnforced bool
	NotValid    bool
}

type Index struct {
	Schema     string
	Name       string
	Table      string
	Columns    []string
	Unique     bool
	Where      string
	Using      string
	Definition string
}

type View struct {
	Schema     string
	Name       string
	Definition string
}

type Function struct {
	Schema     string
	Name       string
	Args       string
	Returns    string
	Language   string
	Body       string
	Definition string
}

type Trigger struct {
	Schema     string
	Name       string
	Table      string
	Timing     string
	Events     []string
	Function   string
	ForEach    string
	When       string
	Definition string
}

type Sequence struct {
	Schema    string
	Name      string
	Start     int64
	Increment int64
	MinValue  int64
	MaxValue  int64
	Cache     int64
	Cycle     bool
}

type Enum struct {
	Schema string
	Name   string
	Values []string
}

type Extension struct {
	Name    string
	Version string
	Schema  string
}

type Domain struct {
	Schema     string
	Name       string
	Type       string
	Default    string
	NotNull    bool
	Check      string
	Collation  string
	Definition string
}

type CompositeType struct {
	Schema     string
	Name       string
	Attributes []Column
	Definition string
}

type MaterializedView struct {
	Schema     string
	Name       string
	Definition string
	Tablespace string
	WithData   bool
	Indexes    []string
}

type Aggregate struct {
	Schema     string
	Name       string
	Args       string
	SFunc      string
	SType      string
	FinalFunc  string
	InitCond   string
	SortOp     string
	Definition string
}

type Rule struct {
	Schema     string
	Name       string
	Table      string
	Event      string
	DoInstead  bool
	Definition string
}

type Policy struct {
	Schema     string
	Name       string
	Table      string
	Command    string
	Permissive bool
	Roles      []string
	Using      string
	WithCheck  string
	Definition string
}

type ForeignDataWrapper struct {
	Name       string
	Handler    string
	Validator  string
	Options    map[string]string
	Definition string
}

type ForeignServer struct {
	Name       string
	FDW        string
	Type       string
	Version    string
	Options    map[string]string
	Definition string
}

type ForeignTable struct {
	Schema     string
	Name       string
	Server     string
	Columns    []Column
	Options    map[string]string
	Definition string
}

type TextSearchConfig struct {
	Schema     string
	Name       string
	Parser     string
	Mappings   map[string][]string
	Definition string
}

type Publication struct {
	Name       string
	AllTables  bool
	Tables     []string
	Operations []string
	Definition string
}

type Subscription struct {
	Name        string
	Publication string
	ConnInfo    string
	Enabled     bool
	SlotName    string
	Definition  string
}

type Operator struct {
	Schema     string
	Name       string
	LeftType   string
	RightType  string
	ResultType string
	Procedure  string
	Commutator string
	Negator    string
	Definition string
}

type Collation struct {
	Schema     string
	Name       string
	Provider   string
	Locale     string
	LcCollate  string
	LcCtype    string
	Definition string
}

type Namespace struct {
	Name  string
	Owner string
}

type Procedure struct {
	Schema     string
	Name       string
	Args       string
	Language   string
	Body       string
	Definition string
}

type EventTrigger struct {
	Name       string
	Event      string
	Function   string
	Enabled    string
	Tags       []string
	Definition string
}

type Comment struct {
	ObjectType string
	Schema     string
	Name       string
	Column     string
	Comment    string
}

type Role struct {
	Name            string
	SuperUser       bool
	CreateDB        bool
	CreateRole      bool
	Inherit         bool
	Login           bool
	Replication     bool
	BypassRLS       bool
	ConnectionLimit int
	ValidUntil      string
	InRoles         []string
}

type RoleGrant struct {
	Privilege  string
	ObjectType string
	Schema     string
	ObjectName string
	Grantee    string
	WithGrant  bool
	GrantedBy  string
}

type DefaultPrivilege struct {
	Schema     string
	Role       string
	ObjectType string
	Privileges []string
	Grantee    string
}

func LoadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file: %w", err)
	}
	return string(data), nil
}

func (s *Schema) ToSQL() string {
	var sb strings.Builder

	for _, ns := range s.Namespaces {
		sb.WriteString(fmt.Sprintf("CREATE SCHEMA %s", quoteIdent(ns.Name)))
		if ns.Owner != "" {
			sb.WriteString(fmt.Sprintf(" AUTHORIZATION %s", quoteIdent(ns.Owner)))
		}
		sb.WriteString(";\n\n")
	}

	for _, ext := range s.Extensions {
		sb.WriteString(fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", quoteIdent(ext.Name)))
		if ext.Schema != "" && ext.Schema != "public" {
			sb.WriteString(fmt.Sprintf(" SCHEMA %s", quoteIdent(ext.Schema)))
		}
		sb.WriteString(";\n\n")
	}

	for _, e := range s.Enums {
		var values []string
		for _, v := range e.Values {
			values = append(values, quoteLiteral(v))
		}
		sb.WriteString(fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);\n\n",
			qualifiedName(e.Schema, e.Name), strings.Join(values, ", ")))
	}

	for _, d := range s.Domains {
		sql := fmt.Sprintf("CREATE DOMAIN %s AS %s", qualifiedName(d.Schema, d.Name), d.Type)
		if d.Collation != "" {
			sql += fmt.Sprintf(" COLLATE %s", quoteIdent(d.Collation))
		}
		if d.Default != "" {
			sql += fmt.Sprintf(" DEFAULT %s", d.Default)
		}
		if d.NotNull {
			sql += " NOT NULL"
		}
		if d.Check != "" {
			sql += fmt.Sprintf(" %s", d.Check)
		}
		sb.WriteString(sql + ";\n\n")
	}

	for _, ct := range s.CompositeTypes {
		var attrs []string
		for _, a := range ct.Attributes {
			attrs = append(attrs, fmt.Sprintf("%s %s", quoteIdent(a.Name), a.Type))
		}
		sb.WriteString(fmt.Sprintf("CREATE TYPE %s AS (%s);\n\n",
			qualifiedName(ct.Schema, ct.Name), strings.Join(attrs, ", ")))
	}

	for _, seq := range s.Sequences {
		sql := fmt.Sprintf("CREATE SEQUENCE %s", qualifiedName(seq.Schema, seq.Name))
		if seq.Start != 0 {
			sql += fmt.Sprintf(" START %d", seq.Start)
		}
		if seq.Increment != 0 {
			sql += fmt.Sprintf(" INCREMENT %d", seq.Increment)
		}
		if seq.MinValue != 0 {
			sql += fmt.Sprintf(" MINVALUE %d", seq.MinValue)
		}
		if seq.MaxValue != 0 {
			sql += fmt.Sprintf(" MAXVALUE %d", seq.MaxValue)
		}
		if seq.Cache != 0 {
			sql += fmt.Sprintf(" CACHE %d", seq.Cache)
		}
		if seq.Cycle {
			sql += " CYCLE"
		}
		sb.WriteString(sql + ";\n\n")
	}

	sortedTables := sortTablesByDependency(s.Tables)
	for _, t := range sortedTables {
		sb.WriteString(generateCreateTable(t))
		sb.WriteString("\n\n")
	}

	for _, idx := range s.Indexes {
		sb.WriteString(generateCreateIndex(idx))
		sb.WriteString("\n\n")
	}

	for _, v := range s.Views {
		sb.WriteString(fmt.Sprintf("CREATE VIEW %s AS %s;\n\n",
			qualifiedName(v.Schema, v.Name), v.Definition))
	}

	for _, mv := range s.MaterializedViews {
		sql := fmt.Sprintf("CREATE MATERIALIZED VIEW %s AS %s",
			qualifiedName(mv.Schema, mv.Name), mv.Definition)
		if !mv.WithData {
			sql += " WITH NO DATA"
		}
		sb.WriteString(sql + ";\n\n")
	}

	for _, f := range s.Functions {
		if f.Definition != "" {
			sb.WriteString(f.Definition + ";\n\n")
		} else {
			sb.WriteString(fmt.Sprintf("CREATE FUNCTION %s(%s) RETURNS %s LANGUAGE %s AS $$%s$$;\n\n",
				qualifiedName(f.Schema, f.Name), f.Args, f.Returns, f.Language, f.Body))
		}
	}

	for _, p := range s.Procedures {
		if p.Definition != "" {
			sb.WriteString(p.Definition + ";\n\n")
		} else {
			sb.WriteString(fmt.Sprintf("CREATE PROCEDURE %s(%s) LANGUAGE %s AS $$%s$$;\n\n",
				qualifiedName(p.Schema, p.Name), p.Args, p.Language, p.Body))
		}
	}

	for _, a := range s.Aggregates {
		sql := fmt.Sprintf("CREATE AGGREGATE %s(%s) (\n    SFUNC = %s,\n    STYPE = %s",
			qualifiedName(a.Schema, a.Name), a.Args, a.SFunc, a.SType)
		if a.FinalFunc != "" {
			sql += fmt.Sprintf(",\n    FINALFUNC = %s", a.FinalFunc)
		}
		if a.InitCond != "" {
			sql += fmt.Sprintf(",\n    INITCOND = '%s'", a.InitCond)
		}
		if a.SortOp != "" {
			sql += fmt.Sprintf(",\n    SORTOP = %s", a.SortOp)
		}
		sb.WriteString(sql + "\n);\n\n")
	}

	for _, t := range s.Triggers {
		if t.Definition != "" {
			sb.WriteString(t.Definition + ";\n\n")
		}
	}

	for _, e := range s.EventTriggers {
		sql := fmt.Sprintf("CREATE EVENT TRIGGER %s ON %s", quoteIdent(e.Name), e.Event)
		if len(e.Tags) > 0 {
			var quotedTags []string
			for _, t := range e.Tags {
				quotedTags = append(quotedTags, quoteLiteral(t))
			}
			sql += fmt.Sprintf(" WHEN TAG IN (%s)", strings.Join(quotedTags, ", "))
		}
		sql += fmt.Sprintf(" EXECUTE FUNCTION %s();", e.Function)
		sb.WriteString(sql + "\n\n")
	}

	for _, r := range s.Rules {
		if r.Definition != "" {
			sb.WriteString(r.Definition + ";\n\n")
		}
	}

	for _, p := range s.Policies {
		tableName := qualifiedName(p.Schema, p.Table)
		sql := fmt.Sprintf("CREATE POLICY %s ON %s", quoteIdent(p.Name), tableName)
		if !p.Permissive {
			sql += " AS RESTRICTIVE"
		}
		if p.Command != "ALL" {
			sql += fmt.Sprintf(" FOR %s", p.Command)
		}
		if len(p.Roles) > 0 {
			sql += fmt.Sprintf(" TO %s", strings.Join(p.Roles, ", "))
		}
		if p.Using != "" {
			sql += fmt.Sprintf(" USING (%s)", p.Using)
		}
		if p.WithCheck != "" {
			sql += fmt.Sprintf(" WITH CHECK (%s)", p.WithCheck)
		}
		sb.WriteString(sql + ";\n\n")
	}

	for _, c := range s.Comments {
		var objectRef string
		switch c.ObjectType {
		case "TABLE", "VIEW", "MATERIALIZED VIEW", "INDEX", "SEQUENCE", "FOREIGN TABLE":
			objectRef = fmt.Sprintf("%s %s", c.ObjectType, qualifiedName(c.Schema, c.Name))
		case "COLUMN":
			objectRef = fmt.Sprintf("COLUMN %s.%s", qualifiedName(c.Schema, c.Name), quoteIdent(c.Column))
		case "FUNCTION":
			objectRef = fmt.Sprintf("FUNCTION %s", qualifiedName(c.Schema, c.Name))
		case "TYPE":
			objectRef = fmt.Sprintf("TYPE %s", qualifiedName(c.Schema, c.Name))
		case "SCHEMA":
			objectRef = fmt.Sprintf("SCHEMA %s", quoteIdent(c.Name))
		default:
			objectRef = fmt.Sprintf("%s %s", c.ObjectType, qualifiedName(c.Schema, c.Name))
		}
		sb.WriteString(fmt.Sprintf("COMMENT ON %s IS %s;\n\n", objectRef, quoteLiteral(c.Comment)))
	}

	for _, r := range s.Roles {
		sql := fmt.Sprintf("CREATE ROLE %s", quoteIdent(r.Name))
		var opts []string
		if r.SuperUser {
			opts = append(opts, "SUPERUSER")
		}
		if r.CreateDB {
			opts = append(opts, "CREATEDB")
		}
		if r.CreateRole {
			opts = append(opts, "CREATEROLE")
		}
		if !r.Inherit {
			opts = append(opts, "NOINHERIT")
		}
		if r.Login {
			opts = append(opts, "LOGIN")
		}
		if r.Replication {
			opts = append(opts, "REPLICATION")
		}
		if r.BypassRLS {
			opts = append(opts, "BYPASSRLS")
		}
		if r.ConnectionLimit >= 0 {
			opts = append(opts, fmt.Sprintf("CONNECTION LIMIT %d", r.ConnectionLimit))
		}
		if r.ValidUntil != "" {
			opts = append(opts, fmt.Sprintf("VALID UNTIL '%s'", r.ValidUntil))
		}
		if len(opts) > 0 {
			sql += " WITH " + strings.Join(opts, " ")
		}
		sb.WriteString(sql + ";\n")
		for _, m := range r.InRoles {
			sb.WriteString(fmt.Sprintf("GRANT %s TO %s;\n", quoteIdent(m), quoteIdent(r.Name)))
		}
		sb.WriteString("\n")
	}

	for _, g := range s.RoleGrants {
		objectRef := qualifiedName(g.Schema, g.ObjectName)
		sql := fmt.Sprintf("GRANT %s ON %s TO %s", g.Privilege, objectRef, quoteIdent(g.Grantee))
		if g.WithGrant {
			sql += " WITH GRANT OPTION"
		}
		sb.WriteString(sql + ";\n")
	}
	if len(s.RoleGrants) > 0 {
		sb.WriteString("\n")
	}

	for _, dp := range s.DefaultPrivileges {
		privs := strings.Join(dp.Privileges, ", ")
		sb.WriteString(fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA %s GRANT %s ON %s TO %s;\n",
			quoteIdent(dp.Role), quoteIdent(dp.Schema), privs, dp.ObjectType, quoteIdent(dp.Grantee)))
	}

	return sb.String()
}

func quoteIdent(s string) string {
	return fmt.Sprintf("\"%s\"", s)
}

func quoteLiteral(s string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "''"))
}

func qualifiedName(schema, name string) string {
	if schema == "" || schema == "public" {
		return quoteIdent(name)
	}
	return fmt.Sprintf("%s.%s", quoteIdent(schema), quoteIdent(name))
}

func sortTablesByDependency(tables []Table) []Table {
	tableMap := make(map[string]Table)
	for _, t := range tables {
		key := t.Schema + "." + t.Name
		if t.Schema == "" || t.Schema == "public" {
			key = t.Name
		}
		tableMap[key] = t
	}

	deps := make(map[string][]string)
	for _, t := range tables {
		key := t.Schema + "." + t.Name
		if t.Schema == "" || t.Schema == "public" {
			key = t.Name
		}
		deps[key] = []string{}
		for _, c := range t.Constraints {
			if c.Type == "FOREIGN KEY" && c.RefTable != "" {
				refKey := c.RefTable
				if !strings.Contains(refKey, ".") && t.Schema != "" && t.Schema != "public" {
					refKey = t.Schema + "." + c.RefTable
				}
				if refKey == key {
					continue
				}
				if _, inSet := tableMap[refKey]; inSet {
					deps[key] = append(deps[key], refKey)
				}
			}
		}
	}

	var sorted []Table
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	var visit func(key string) bool
	visit = func(key string) bool {
		if temp[key] {
			return false
		}
		if visited[key] {
			return true
		}
		temp[key] = true
		for _, dep := range deps[key] {
			if !visit(dep) {
				return false
			}
		}
		temp[key] = false
		visited[key] = true
		sorted = append(sorted, tableMap[key])
		return true
	}

	for _, t := range tables {
		key := t.Schema + "." + t.Name
		if t.Schema == "" || t.Schema == "public" {
			key = t.Name
		}
		if !visited[key] {
			visit(key)
		}
	}

	return sorted
}

func generateCreateTable(t Table) string {
	var sb strings.Builder

	if t.PartitionOf != "" {
		sb.WriteString(fmt.Sprintf("CREATE TABLE %s PARTITION OF %s %s;",
			qualifiedName(t.Schema, t.Name),
			quoteIdent(t.PartitionOf),
			t.PartitionBound))
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", qualifiedName(t.Schema, t.Name)))

	for i, col := range t.Columns {
		sb.WriteString(fmt.Sprintf("    %s %s", quoteIdent(col.Name), col.Type))

		if col.Identity != "" {
			sb.WriteString(fmt.Sprintf(" GENERATED %s AS IDENTITY", col.Identity))
			var identityOpts []string
			if col.IdentityStart != 0 {
				identityOpts = append(identityOpts, fmt.Sprintf("START WITH %d", col.IdentityStart))
			}
			if col.IdentityIncrement != 0 && col.IdentityIncrement != 1 {
				identityOpts = append(identityOpts, fmt.Sprintf("INCREMENT BY %d", col.IdentityIncrement))
			}
			if col.IdentityMinValue != nil {
				identityOpts = append(identityOpts, fmt.Sprintf("MINVALUE %d", *col.IdentityMinValue))
			}
			if col.IdentityMaxValue != nil {
				identityOpts = append(identityOpts, fmt.Sprintf("MAXVALUE %d", *col.IdentityMaxValue))
			}
			if col.IdentityCache != 0 && col.IdentityCache != 1 {
				identityOpts = append(identityOpts, fmt.Sprintf("CACHE %d", col.IdentityCache))
			}
			if col.IdentityCycle {
				identityOpts = append(identityOpts, "CYCLE")
			}
			if len(identityOpts) > 0 {
				sb.WriteString(fmt.Sprintf(" (%s)", strings.Join(identityOpts, " ")))
			}
		}

		if col.GeneratedAs != "" {
			sb.WriteString(fmt.Sprintf(" GENERATED ALWAYS AS (%s)", col.GeneratedAs))
			if col.GeneratedType != "" {
				sb.WriteString(fmt.Sprintf(" %s", col.GeneratedType))
			}
		}

		if !col.Nullable {
			if col.NotNullConstraintName != "" {
				sb.WriteString(fmt.Sprintf(" CONSTRAINT %s NOT NULL", quoteIdent(col.NotNullConstraintName)))
			} else {
				sb.WriteString(" NOT NULL")
			}
		}

		if col.Default != "" {
			sb.WriteString(fmt.Sprintf(" DEFAULT %s", col.Default))
		}
		if col.PrimaryKey {
			sb.WriteString(" PRIMARY KEY")
		}
		if i < len(t.Columns)-1 || len(t.Constraints) > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	for i, constraint := range t.Constraints {
		switch constraint.Type {
		case "PRIMARY KEY":
			cols := strings.Join(quoteIdents(constraint.Columns), ", ")
			if constraint.WithoutOverlaps && constraint.PeriodColumn != "" {
				cols = fmt.Sprintf("%s, %s WITHOUT OVERLAPS", cols, quoteIdent(constraint.PeriodColumn))
			}
			sb.WriteString(fmt.Sprintf("    CONSTRAINT %s PRIMARY KEY (%s)",
				quoteIdent(constraint.Name), cols))

		case "FOREIGN KEY":
			refTable := constraint.RefTable
			if !strings.Contains(refTable, ".") {
				refTable = quoteIdent(refTable)
			}
			sb.WriteString(fmt.Sprintf("    CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
				quoteIdent(constraint.Name),
				strings.Join(quoteIdents(constraint.Columns), ", "),
				refTable,
				strings.Join(quoteIdents(constraint.RefColumns), ", ")))
			if constraint.OnDelete != "" && constraint.OnDelete != "NO ACTION" {
				sb.WriteString(fmt.Sprintf(" ON DELETE %s", constraint.OnDelete))
			}
			if constraint.OnUpdate != "" && constraint.OnUpdate != "NO ACTION" {
				sb.WriteString(fmt.Sprintf(" ON UPDATE %s", constraint.OnUpdate))
			}
			if constraint.NotValid {
				sb.WriteString(" NOT VALID")
			}
			if constraint.NotEnforced {
				sb.WriteString(" NOT ENFORCED")
			}

		case "UNIQUE":
			cols := strings.Join(quoteIdents(constraint.Columns), ", ")
			if constraint.WithoutOverlaps && constraint.PeriodColumn != "" {
				cols = fmt.Sprintf("%s, %s WITHOUT OVERLAPS", cols, quoteIdent(constraint.PeriodColumn))
			}
			sb.WriteString(fmt.Sprintf("    CONSTRAINT %s UNIQUE (%s)",
				quoteIdent(constraint.Name), cols))

		case "CHECK":
			if constraint.Name != "" {
				sb.WriteString(fmt.Sprintf("    CONSTRAINT %s CHECK (%s)",
					quoteIdent(constraint.Name), constraint.Check))
			} else {
				sb.WriteString(fmt.Sprintf("    CHECK (%s)", constraint.Check))
			}
			if constraint.NotValid {
				sb.WriteString(" NOT VALID")
			}
			if constraint.NotEnforced {
				sb.WriteString(" NOT ENFORCED")
			}

		case "EXCLUSION":
			using := constraint.ExclusionUsing
			if using == "" {
				using = "gist"
			}
			var elemParts []string
			for _, col := range constraint.Columns {
				op := "="
				if constraint.ExclusionOperators != nil {
					if mappedOp, ok := constraint.ExclusionOperators[col]; ok {
						op = mappedOp
					}
				}
				elemParts = append(elemParts, fmt.Sprintf("%s WITH %s", quoteIdent(col), op))
			}
			sb.WriteString(fmt.Sprintf("    CONSTRAINT %s EXCLUDE USING %s (%s)",
				quoteIdent(constraint.Name), using, strings.Join(elemParts, ", ")))
			if constraint.ExclusionWhere != "" {
				sb.WriteString(fmt.Sprintf(" WHERE (%s)", constraint.ExclusionWhere))
			}
		}
		if i < len(t.Constraints)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(")")

	if t.PartitionBy != "" {
		sb.WriteString(fmt.Sprintf(" PARTITION BY %s (%s)", t.PartitionBy, t.PartitionKey))
	}

	sb.WriteString(";")
	return sb.String()
}

func generateCreateIndex(i Index) string {
	var sb strings.Builder

	sb.WriteString("CREATE ")
	if i.Unique {
		sb.WriteString("UNIQUE ")
	}
	sb.WriteString(fmt.Sprintf("INDEX %s ON %s", quoteIdent(i.Name), qualifiedName(i.Schema, i.Table)))

	if i.Using != "" && i.Using != "btree" {
		sb.WriteString(fmt.Sprintf(" USING %s", i.Using))
	}

	sb.WriteString(fmt.Sprintf(" (%s)", strings.Join(i.Columns, ", ")))

	if i.Where != "" {
		sb.WriteString(fmt.Sprintf(" WHERE %s", i.Where))
	}

	sb.WriteString(";")
	return sb.String()
}

func quoteIdents(ss []string) []string {
	result := make([]string, len(ss))
	for i, s := range ss {
		result[i] = quoteIdent(s)
	}
	return result
}

func (s *Schema) Lint() []string {
	var warnings []string

	for _, table := range s.Tables {
		hasPK := false
		for _, col := range table.Columns {
			if col.PrimaryKey {
				hasPK = true
				break
			}
		}
		for _, constraint := range table.Constraints {
			if constraint.Type == "PRIMARY KEY" {
				hasPK = true
				break
			}
		}
		if !hasPK {
			warnings = append(warnings, fmt.Sprintf("table %q has no primary key", table.Name))
		}
	}

	warnings = append(warnings, s.lintMissingForeignKeyIndexes()...)

	return warnings
}

func (s *Schema) lintMissingForeignKeyIndexes() []string {
	var warnings []string

	tableIndexes := make(map[string]map[string]bool)
	for _, idx := range s.Indexes {
		key := idx.Schema + "." + idx.Table
		if tableIndexes[key] == nil {
			tableIndexes[key] = make(map[string]bool)
		}
		if len(idx.Columns) > 0 {
			tableIndexes[key][idx.Columns[0]] = true
		}
	}

	for _, table := range s.Tables {
		tableKey := table.Schema + "." + table.Name

		pkCols := make(map[string]bool)
		for _, col := range table.Columns {
			if col.PrimaryKey {
				pkCols[col.Name] = true
			}
		}
		for _, constraint := range table.Constraints {
			if constraint.Type == "PRIMARY KEY" {
				for _, col := range constraint.Columns {
					pkCols[col] = true
				}
			}
		}

		for _, constraint := range table.Constraints {
			if constraint.Type != "FOREIGN KEY" || len(constraint.Columns) == 0 {
				continue
			}

			leadingCol := constraint.Columns[0]

			if pkCols[leadingCol] {
				continue
			}

			if tableIndexes[tableKey] != nil && tableIndexes[tableKey][leadingCol] {
				continue
			}

			colList := constraint.Columns[0]
			if len(constraint.Columns) > 1 {
				colList = fmt.Sprintf("(%s, ...)", constraint.Columns[0])
			}

			warnings = append(warnings, fmt.Sprintf(
				"table %q: foreign key on %s referencing %q has no index (queries joining on this column will be slow)",
				table.Name, colList, constraint.RefTable,
			))
		}
	}

	return warnings
}

func (s *Schema) ObjectCount() int {
	return len(s.Namespaces) +
		len(s.Extensions) +
		len(s.Enums) +
		len(s.Domains) +
		len(s.CompositeTypes) +
		len(s.Sequences) +
		len(s.Tables) +
		len(s.Indexes) +
		len(s.Views) +
		len(s.MaterializedViews) +
		len(s.Functions) +
		len(s.Procedures) +
		len(s.Aggregates) +
		len(s.Triggers) +
		len(s.EventTriggers) +
		len(s.Rules) +
		len(s.Policies) +
		len(s.ForeignDataWrappers) +
		len(s.ForeignServers) +
		len(s.ForeignTables) +
		len(s.TextSearchConfigs) +
		len(s.Publications) +
		len(s.Subscriptions) +
		len(s.Operators) +
		len(s.Collations) +
		len(s.Comments) +
		len(s.Roles) +
		len(s.RoleGrants) +
		len(s.DefaultPrivileges)
}
