package diff

import (
	"fmt"
	"strings"

	"github.com/terminally-online/shrugged/internal/parser"
)

type TableChange struct {
	ChangeType      ChangeType
	Table           parser.Table
	OldTable        *parser.Table
	AddColumns      []parser.Column
	DropColumns     []string
	AlterColumns    []ColumnAlteration
	AddConstraints  []parser.Constraint
	DropConstraints []string
}

type ColumnAlteration struct {
	Column    parser.Column
	OldColumn parser.Column
	Changes   []string
}

func (c *TableChange) SQL() string {
	switch c.ChangeType {
	case CreateTable:
		return generateCreateTable(c.Table)
	case DropTable:
		return fmt.Sprintf("DROP TABLE %s;", qualifiedName(c.Table.Schema, c.Table.Name))
	case AlterTable:
		return generateAlterTable(c)
	}
	return ""
}

func (c *TableChange) DownSQL() string {
	switch c.ChangeType {
	case CreateTable:
		return fmt.Sprintf("DROP TABLE %s;", qualifiedName(c.Table.Schema, c.Table.Name))
	case DropTable:
		if c.OldTable != nil {
			return generateCreateTable(*c.OldTable)
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped table %s", c.Table.Name)
	case AlterTable:
		return generateAlterTableDown(c)
	}
	return ""
}

func (c *TableChange) Type() ChangeType {
	return c.ChangeType
}

func (c *TableChange) ObjectName() string {
	return c.Table.Name
}

func (c *TableChange) IsReversible() bool {
	if c.ChangeType == DropTable && c.OldTable == nil {
		return false
	}
	for _, col := range c.DropColumns {
		_ = col
		return false
	}
	return true
}

func compareTables(current, desired []parser.Table) []Change {
	var changes []Change

	currentMap := make(map[string]parser.Table)
	for _, t := range current {
		currentMap[objectKey(t.Schema, t.Name)] = t
	}

	desiredMap := make(map[string]parser.Table)
	for _, t := range desired {
		desiredMap[objectKey(t.Schema, t.Name)] = t
	}

	var tablesToCreate []parser.Table
	for _, t := range desired {
		if _, exists := currentMap[objectKey(t.Schema, t.Name)]; !exists {
			tablesToCreate = append(tablesToCreate, t)
		}
	}

	sortedTables := sortTablesByDependency(tablesToCreate)
	for _, t := range sortedTables {
		changes = append(changes, &TableChange{ChangeType: CreateTable, Table: t})
	}

	for _, t := range current {
		if _, exists := desiredMap[objectKey(t.Schema, t.Name)]; !exists {
			changes = append(changes, &TableChange{ChangeType: DropTable, Table: t})
		}
	}

	for key, desiredTable := range desiredMap {
		if currentTable, exists := currentMap[key]; exists {
			if tableChanges := compareTableColumns(currentTable, desiredTable); tableChanges != nil {
				changes = append(changes, tableChanges)
			}
		}
	}

	return changes
}

func sortTablesByDependency(tables []parser.Table) []parser.Table {
	tableMap := make(map[string]parser.Table)
	for _, t := range tables {
		tableMap[objectKey(t.Schema, t.Name)] = t
	}

	deps := make(map[string][]string)
	for _, t := range tables {
		key := objectKey(t.Schema, t.Name)
		deps[key] = []string{}
		for _, c := range t.Constraints {
			if c.Type == "FOREIGN KEY" && c.RefTable != "" {
				refKey := c.RefTable
				if !strings.Contains(refKey, ".") {
					refKey = objectKey(t.Schema, c.RefTable)
				}
				if _, inSet := tableMap[refKey]; inSet {
					deps[key] = append(deps[key], refKey)
				}
			}
		}
	}

	var sorted []parser.Table
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
		key := objectKey(t.Schema, t.Name)
		if !visited[key] {
			visit(key)
		}
	}

	return sorted
}

func compareTableColumns(current, desired parser.Table) *TableChange {
	change := &TableChange{
		ChangeType: AlterTable,
		Table:      desired,
		OldTable:   &current,
	}

	currentCols := make(map[string]parser.Column)
	for _, c := range current.Columns {
		currentCols[c.Name] = c
	}

	desiredCols := make(map[string]parser.Column)
	for _, c := range desired.Columns {
		desiredCols[c.Name] = c
	}

	for _, col := range desired.Columns {
		if _, exists := currentCols[col.Name]; !exists {
			change.AddColumns = append(change.AddColumns, col)
		}
	}

	for _, col := range current.Columns {
		if _, exists := desiredCols[col.Name]; !exists {
			change.DropColumns = append(change.DropColumns, col.Name)
		}
	}

	for name, desiredCol := range desiredCols {
		if currentCol, exists := currentCols[name]; exists {
			if alt := compareColumn(currentCol, desiredCol); alt != nil {
				change.AlterColumns = append(change.AlterColumns, *alt)
			}
		}
	}

	if len(change.AddColumns) == 0 && len(change.DropColumns) == 0 && len(change.AlterColumns) == 0 {
		return nil
	}

	return change
}

func compareColumn(current, desired parser.Column) *ColumnAlteration {
	var changes []string

	if normalizeType(current.Type) != normalizeType(desired.Type) {
		changes = append(changes, "type")
	}
	if current.Nullable != desired.Nullable {
		changes = append(changes, "nullable")
	}
	if current.Default != desired.Default {
		changes = append(changes, "default")
	}

	if current.Identity != desired.Identity {
		changes = append(changes, "identity")
	}
	if current.Identity != "" && desired.Identity != "" {
		if current.IdentityStart != desired.IdentityStart ||
			current.IdentityIncrement != desired.IdentityIncrement ||
			current.IdentityCache != desired.IdentityCache ||
			current.IdentityCycle != desired.IdentityCycle {
			changes = append(changes, "identity_options")
		}
	}

	if current.GeneratedAs != desired.GeneratedAs {
		changes = append(changes, "generated")
	}
	if current.GeneratedType != desired.GeneratedType {
		changes = append(changes, "generated_type")
	}

	if len(changes) == 0 {
		return nil
	}

	return &ColumnAlteration{
		Column:    desired,
		OldColumn: current,
		Changes:   changes,
	}
}

func generateCreateTable(t parser.Table) string {
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

func generateAlterTable(c *TableChange) string {
	var stmts []string
	tableName := qualifiedName(c.Table.Schema, c.Table.Name)

	for _, col := range c.AddColumns {
		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, quoteIdent(col.Name), col.Type)

		if col.Identity != "" {
			stmt += fmt.Sprintf(" GENERATED %s AS IDENTITY", col.Identity)
		}

		if col.GeneratedAs != "" {
			stmt += fmt.Sprintf(" GENERATED ALWAYS AS (%s)", col.GeneratedAs)
			if col.GeneratedType != "" {
				stmt += fmt.Sprintf(" %s", col.GeneratedType)
			}
		}

		if !col.Nullable {
			stmt += " NOT NULL"
		}
		if col.Default != "" {
			stmt += fmt.Sprintf(" DEFAULT %s", col.Default)
		}
		stmts = append(stmts, stmt+";")
	}

	for _, colName := range c.DropColumns {
		stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableName, quoteIdent(colName)))
	}

	for _, alt := range c.AlterColumns {
		colName := quoteIdent(alt.Column.Name)
		for _, change := range alt.Changes {
			switch change {
			case "type":
				stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
					tableName, colName, alt.Column.Type))
			case "nullable":
				if alt.Column.Nullable {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
						tableName, colName))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
						tableName, colName))
				}
			case "default":
				if alt.Column.Default == "" {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
						tableName, colName))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
						tableName, colName, alt.Column.Default))
				}
			case "identity":
				if alt.Column.Identity == "" {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP IDENTITY;",
						tableName, colName))
				} else if alt.OldColumn.Identity == "" {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s ADD GENERATED %s AS IDENTITY;",
						tableName, colName, alt.Column.Identity))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET GENERATED %s;",
						tableName, colName, alt.Column.Identity))
				}
			case "identity_options":
				var opts []string
				if alt.Column.IdentityStart != alt.OldColumn.IdentityStart {
					opts = append(opts, fmt.Sprintf("RESTART WITH %d", alt.Column.IdentityStart))
				}
				if alt.Column.IdentityIncrement != alt.OldColumn.IdentityIncrement {
					opts = append(opts, fmt.Sprintf("INCREMENT BY %d", alt.Column.IdentityIncrement))
				}
				if alt.Column.IdentityCache != alt.OldColumn.IdentityCache {
					opts = append(opts, fmt.Sprintf("CACHE %d", alt.Column.IdentityCache))
				}
				if alt.Column.IdentityCycle != alt.OldColumn.IdentityCycle {
					if alt.Column.IdentityCycle {
						opts = append(opts, "CYCLE")
					} else {
						opts = append(opts, "NO CYCLE")
					}
				}
				if len(opts) > 0 {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET %s;",
						tableName, colName, strings.Join(opts, " ")))
				}
			case "generated":
				if alt.Column.GeneratedAs == "" {
					stmts = append(stmts, fmt.Sprintf("-- Cannot directly drop generated expression for %s; column must be dropped and recreated", colName))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET EXPRESSION AS (%s);",
						tableName, colName, alt.Column.GeneratedAs))
				}
			case "generated_type":
				stmts = append(stmts, fmt.Sprintf("-- Cannot directly change generated column type for %s; column must be dropped and recreated", colName))
			}
		}
	}

	return strings.Join(stmts, "\n")
}

func generateAlterTableDown(c *TableChange) string {
	var stmts []string
	tableName := qualifiedName(c.Table.Schema, c.Table.Name)

	for _, col := range c.AddColumns {
		stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableName, quoteIdent(col.Name)))
	}

	for _, colName := range c.DropColumns {
		stmts = append(stmts, fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped column %s", colName))
	}

	for _, alt := range c.AlterColumns {
		colName := quoteIdent(alt.Column.Name)
		for _, change := range alt.Changes {
			switch change {
			case "type":
				stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
					tableName, colName, alt.OldColumn.Type))
			case "nullable":
				if alt.OldColumn.Nullable {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
						tableName, colName))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
						tableName, colName))
				}
			case "default":
				if alt.OldColumn.Default == "" {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
						tableName, colName))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
						tableName, colName, alt.OldColumn.Default))
				}
			case "identity":
				if alt.OldColumn.Identity == "" {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP IDENTITY;",
						tableName, colName))
				} else if alt.Column.Identity == "" {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s ADD GENERATED %s AS IDENTITY;",
						tableName, colName, alt.OldColumn.Identity))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET GENERATED %s;",
						tableName, colName, alt.OldColumn.Identity))
				}
			case "identity_options":
				var opts []string
				if alt.Column.IdentityStart != alt.OldColumn.IdentityStart {
					opts = append(opts, fmt.Sprintf("RESTART WITH %d", alt.OldColumn.IdentityStart))
				}
				if alt.Column.IdentityIncrement != alt.OldColumn.IdentityIncrement {
					opts = append(opts, fmt.Sprintf("INCREMENT BY %d", alt.OldColumn.IdentityIncrement))
				}
				if alt.Column.IdentityCache != alt.OldColumn.IdentityCache {
					opts = append(opts, fmt.Sprintf("CACHE %d", alt.OldColumn.IdentityCache))
				}
				if alt.Column.IdentityCycle != alt.OldColumn.IdentityCycle {
					if alt.OldColumn.IdentityCycle {
						opts = append(opts, "CYCLE")
					} else {
						opts = append(opts, "NO CYCLE")
					}
				}
				if len(opts) > 0 {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET %s;",
						tableName, colName, strings.Join(opts, " ")))
				}
			case "generated":
				if alt.OldColumn.GeneratedAs == "" {
					stmts = append(stmts, fmt.Sprintf("-- IRREVERSIBLE: Cannot directly drop generated expression for %s", colName))
				} else {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET EXPRESSION AS (%s);",
						tableName, colName, alt.OldColumn.GeneratedAs))
				}
			case "generated_type":
				stmts = append(stmts, fmt.Sprintf("-- IRREVERSIBLE: Cannot directly change generated column type for %s", colName))
			}
		}
	}

	return strings.Join(stmts, "\n")
}
