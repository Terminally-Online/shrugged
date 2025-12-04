package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateTable(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Columns: []parser.Column{{Name: "id", Type: "integer"}}},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateTable && c.ObjectName() == "users" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateTable change for users")
	}
}

func TestCompare_DropTable(t *testing.T) {
	current := &parser.Schema{
		Tables: []parser.Table{
			{Name: "old_table", Columns: []parser.Column{{Name: "id", Type: "integer"}}},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropTable && c.ObjectName() == "old_table" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropTable change for old_table")
	}
}

func TestCompare_AlterTable_AddColumn(t *testing.T) {
	current := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Columns: []parser.Column{{Name: "id", Type: "integer"}}},
		},
	}
	desired := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Columns: []parser.Column{
				{Name: "id", Type: "integer"},
				{Name: "email", Type: "text"},
			}},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterTable && c.ObjectName() == "users" {
			tableChange := c.(*TableChange)
			if len(tableChange.AddColumns) == 1 && tableChange.AddColumns[0].Name == "email" {
				found = true
			}
			break
		}
	}

	if !found {
		t.Error("expected AlterTable change with AddColumn for email")
	}
}

func TestCompare_AlterTable_DropColumn(t *testing.T) {
	current := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Columns: []parser.Column{
				{Name: "id", Type: "integer"},
				{Name: "deprecated_col", Type: "text"},
			}},
		},
	}
	desired := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Columns: []parser.Column{{Name: "id", Type: "integer"}}},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterTable && c.ObjectName() == "users" {
			tableChange := c.(*TableChange)
			if len(tableChange.DropColumns) == 1 && tableChange.DropColumns[0] == "deprecated_col" {
				found = true
			}
			break
		}
	}

	if !found {
		t.Error("expected AlterTable change with DropColumn for deprecated_col")
	}
}

func TestCompare_AlterTable_ChangeColumnType(t *testing.T) {
	current := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Columns: []parser.Column{{Name: "age", Type: "integer"}}},
		},
	}
	desired := &parser.Schema{
		Tables: []parser.Table{
			{Name: "users", Columns: []parser.Column{{Name: "age", Type: "bigint"}}},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterTable && c.ObjectName() == "users" {
			tableChange := c.(*TableChange)
			for _, alt := range tableChange.AlterColumns {
				if alt.Column.Name == "age" {
					for _, ch := range alt.Changes {
						if ch == "type" {
							found = true
						}
					}
				}
			}
			break
		}
	}

	if !found {
		t.Error("expected AlterTable change with type alteration for age column")
	}
}

func TestTableChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *TableChange
		want   []string
	}{
		{
			name: "create simple table",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "users",
					Columns: []parser.Column{
						{Name: "id", Type: "integer", Nullable: false},
						{Name: "email", Type: "text", Nullable: true},
					},
				},
			},
			want: []string{"CREATE TABLE", "users", "id", "integer", "NOT NULL", "email", "text"},
		},
		{
			name: "create table with primary key",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "users",
					Columns: []parser.Column{
						{Name: "id", Type: "integer", Nullable: false},
					},
					Constraints: []parser.Constraint{
						{Name: "users_pkey", Type: "PRIMARY KEY", Columns: []string{"id"}},
					},
				},
			},
			want: []string{"CREATE TABLE", "CONSTRAINT", "users_pkey", "PRIMARY KEY"},
		},
		{
			name: "create table with foreign key",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "posts",
					Columns: []parser.Column{
						{Name: "id", Type: "integer"},
						{Name: "user_id", Type: "integer"},
					},
					Constraints: []parser.Constraint{
						{Name: "posts_user_fkey", Type: "FOREIGN KEY", Columns: []string{"user_id"}, RefTable: "users", RefColumns: []string{"id"}, OnDelete: "CASCADE"},
					},
				},
			},
			want: []string{"FOREIGN KEY", "REFERENCES", "users", "ON DELETE CASCADE"},
		},
		{
			name: "create table with unique constraint",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "users",
					Columns: []parser.Column{
						{Name: "email", Type: "text"},
					},
					Constraints: []parser.Constraint{
						{Name: "users_email_unique", Type: "UNIQUE", Columns: []string{"email"}},
					},
				},
			},
			want: []string{"CONSTRAINT", "users_email_unique", "UNIQUE"},
		},
		{
			name: "create partitioned table",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "events",
					Columns: []parser.Column{
						{Name: "id", Type: "integer"},
						{Name: "created_at", Type: "timestamp"},
					},
					PartitionBy:  "RANGE",
					PartitionKey: "created_at",
				},
			},
			want: []string{"CREATE TABLE", "PARTITION BY RANGE (created_at)"},
		},
		{
			name: "drop table",
			change: &TableChange{
				ChangeType: DropTable,
				Table:      parser.Table{Name: "old_table"},
			},
			want: []string{"DROP TABLE", "old_table"},
		},
		{
			name: "alter table add column",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "users"},
				AddColumns: []parser.Column{
					{Name: "phone", Type: "text", Nullable: true},
				},
			},
			want: []string{"ALTER TABLE", "ADD COLUMN", "phone", "text"},
		},
		{
			name: "alter table drop column",
			change: &TableChange{
				ChangeType:  AlterTable,
				Table:       parser.Table{Name: "users"},
				DropColumns: []string{"deprecated"},
			},
			want: []string{"ALTER TABLE", "DROP COLUMN", "deprecated"},
		},
		{
			name: "alter table change column type",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "users"},
				AlterColumns: []ColumnAlteration{
					{
						Column:    parser.Column{Name: "age", Type: "bigint"},
						OldColumn: parser.Column{Name: "age", Type: "integer"},
						Changes:   []string{"type"},
					},
				},
			},
			want: []string{"ALTER TABLE", "ALTER COLUMN", "age", "TYPE", "bigint"},
		},
		{
			name: "alter table set not null",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "users"},
				AlterColumns: []ColumnAlteration{
					{
						Column:    parser.Column{Name: "email", Type: "text", Nullable: false},
						OldColumn: parser.Column{Name: "email", Type: "text", Nullable: true},
						Changes:   []string{"nullable"},
					},
				},
			},
			want: []string{"ALTER TABLE", "ALTER COLUMN", "email", "SET NOT NULL"},
		},
		{
			name: "alter table drop not null",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "users"},
				AlterColumns: []ColumnAlteration{
					{
						Column:    parser.Column{Name: "email", Type: "text", Nullable: true},
						OldColumn: parser.Column{Name: "email", Type: "text", Nullable: false},
						Changes:   []string{"nullable"},
					},
				},
			},
			want: []string{"ALTER TABLE", "ALTER COLUMN", "email", "DROP NOT NULL"},
		},
		{
			name: "alter table set default",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "users"},
				AlterColumns: []ColumnAlteration{
					{
						Column:    parser.Column{Name: "status", Type: "text", Default: "'active'"},
						OldColumn: parser.Column{Name: "status", Type: "text", Default: ""},
						Changes:   []string{"default"},
					},
				},
			},
			want: []string{"ALTER TABLE", "ALTER COLUMN", "status", "SET DEFAULT", "'active'"},
		},
		{
			name: "alter table drop default",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "users"},
				AlterColumns: []ColumnAlteration{
					{
						Column:    parser.Column{Name: "status", Type: "text", Default: ""},
						OldColumn: parser.Column{Name: "status", Type: "text", Default: "'active'"},
						Changes:   []string{"default"},
					},
				},
			},
			want: []string{"ALTER TABLE", "ALTER COLUMN", "status", "DROP DEFAULT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.change.SQL()
			for _, want := range tt.want {
				if !strings.Contains(sql, want) {
					t.Errorf("SQL = %q, should contain %q", sql, want)
				}
			}
		})
	}
}

func TestTableChange_DownSQL(t *testing.T) {
	createChange := &TableChange{
		ChangeType: CreateTable,
		Table: parser.Table{
			Name:    "users",
			Columns: []parser.Column{{Name: "id", Type: "integer"}},
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP TABLE") {
		t.Error("DownSQL for CreateTable should contain DROP TABLE")
	}

	oldTable := parser.Table{Name: "old_table", Columns: []parser.Column{{Name: "id", Type: "integer"}, {Name: "name", Type: "text"}}}
	dropChange := &TableChange{
		ChangeType: DropTable,
		Table:      parser.Table{Name: "old_table"},
		OldTable:   &oldTable,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE TABLE") {
		t.Error("DownSQL for DropTable with OldTable should contain CREATE TABLE")
	}

	dropChangeNoOld := &TableChange{
		ChangeType: DropTable,
		Table:      parser.Table{Name: "old_table"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropTable without OldTable should indicate IRREVERSIBLE")
	}

	alterAddColChange := &TableChange{
		ChangeType: AlterTable,
		Table:      parser.Table{Name: "users"},
		AddColumns: []parser.Column{{Name: "new_col", Type: "text"}},
	}
	alterDownSQL := alterAddColChange.DownSQL()
	if !strings.Contains(alterDownSQL, "DROP COLUMN") {
		t.Error("DownSQL for AlterTable AddColumn should contain DROP COLUMN")
	}
}

func TestTableChange_IsReversible(t *testing.T) {
	createChange := &TableChange{
		ChangeType: CreateTable,
		Table: parser.Table{
			Name:    "test",
			Columns: []parser.Column{{Name: "id", Type: "integer"}},
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateTable should be reversible")
	}

	dropChangeNoOld := &TableChange{
		ChangeType: DropTable,
		Table:      parser.Table{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropTable without OldTable should not be reversible")
	}

	alterWithDropCol := &TableChange{
		ChangeType:  AlterTable,
		Table:       parser.Table{Name: "test"},
		DropColumns: []string{"col"},
	}
	if alterWithDropCol.IsReversible() {
		t.Error("AlterTable with DropColumns should not be reversible")
	}
}

func TestSortTablesByDependency(t *testing.T) {
	tables := []parser.Table{
		{
			Name: "orders",
			Constraints: []parser.Constraint{
				{Type: "FOREIGN KEY", RefTable: "users"},
				{Type: "FOREIGN KEY", RefTable: "products"},
			},
		},
		{Name: "users"},
		{
			Name: "products",
			Constraints: []parser.Constraint{
				{Type: "FOREIGN KEY", RefTable: "categories"},
			},
		},
		{Name: "categories"},
	}

	sorted := sortTablesByDependency(tables)

	indexOf := func(name string) int {
		for i, t := range sorted {
			if t.Name == name {
				return i
			}
		}
		return -1
	}

	if indexOf("users") > indexOf("orders") {
		t.Error("users should come before orders")
	}
	if indexOf("categories") > indexOf("products") {
		t.Error("categories should come before products")
	}
	if indexOf("products") > indexOf("orders") {
		t.Error("products should come before orders")
	}
}

func TestCompareColumn(t *testing.T) {
	noChange := compareColumn(
		parser.Column{Name: "id", Type: "integer", Nullable: false, Default: ""},
		parser.Column{Name: "id", Type: "integer", Nullable: false, Default: ""},
	)
	if noChange != nil {
		t.Error("identical columns should return nil")
	}

	typeChange := compareColumn(
		parser.Column{Name: "id", Type: "integer"},
		parser.Column{Name: "id", Type: "bigint"},
	)
	if typeChange == nil || len(typeChange.Changes) == 0 {
		t.Error("type change should be detected")
	}

	nullableChange := compareColumn(
		parser.Column{Name: "name", Type: "text", Nullable: true},
		parser.Column{Name: "name", Type: "text", Nullable: false},
	)
	if nullableChange == nil {
		t.Error("nullable change should be detected")
	}

	defaultChange := compareColumn(
		parser.Column{Name: "status", Type: "text", Default: ""},
		parser.Column{Name: "status", Type: "text", Default: "'active'"},
	)
	if defaultChange == nil {
		t.Error("default change should be detected")
	}
}

// PG 17/18 Feature Tests

func TestTableChange_SQL_IdentityColumn(t *testing.T) {
	tests := []struct {
		name   string
		change *TableChange
		want   []string
	}{
		{
			name: "create table with identity column always",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "users",
					Columns: []parser.Column{
						{Name: "id", Type: "bigint", Identity: "ALWAYS"},
					},
				},
			},
			want: []string{"GENERATED ALWAYS AS IDENTITY"},
		},
		{
			name: "create table with identity column by default",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "users",
					Columns: []parser.Column{
						{Name: "id", Type: "bigint", Identity: "BY DEFAULT"},
					},
				},
			},
			want: []string{"GENERATED BY DEFAULT AS IDENTITY"},
		},
		{
			name: "create table with identity column with options",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "users",
					Columns: []parser.Column{
						{
							Name:              "id",
							Type:              "bigint",
							Identity:          "ALWAYS",
							IdentityStart:     1000,
							IdentityIncrement: 10,
							IdentityCache:     20,
							IdentityCycle:     true,
						},
					},
				},
			},
			want: []string{"GENERATED ALWAYS AS IDENTITY", "START WITH 1000", "INCREMENT BY 10", "CACHE 20", "CYCLE"},
		},
		{
			name: "alter table add identity",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "users"},
				AlterColumns: []ColumnAlteration{
					{
						Column:    parser.Column{Name: "id", Type: "bigint", Identity: "ALWAYS"},
						OldColumn: parser.Column{Name: "id", Type: "bigint", Identity: ""},
						Changes:   []string{"identity"},
					},
				},
			},
			want: []string{"ADD GENERATED ALWAYS AS IDENTITY"},
		},
		{
			name: "alter table drop identity",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "users"},
				AlterColumns: []ColumnAlteration{
					{
						Column:    parser.Column{Name: "id", Type: "bigint", Identity: ""},
						OldColumn: parser.Column{Name: "id", Type: "bigint", Identity: "ALWAYS"},
						Changes:   []string{"identity"},
					},
				},
			},
			want: []string{"DROP IDENTITY"},
		},
		{
			name: "alter table change identity type",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "users"},
				AlterColumns: []ColumnAlteration{
					{
						Column:    parser.Column{Name: "id", Type: "bigint", Identity: "BY DEFAULT"},
						OldColumn: parser.Column{Name: "id", Type: "bigint", Identity: "ALWAYS"},
						Changes:   []string{"identity"},
					},
				},
			},
			want: []string{"SET GENERATED BY DEFAULT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.change.SQL()
			for _, want := range tt.want {
				if !strings.Contains(sql, want) {
					t.Errorf("SQL = %q, should contain %q", sql, want)
				}
			}
		})
	}
}

func TestTableChange_SQL_GeneratedColumn(t *testing.T) {
	tests := []struct {
		name   string
		change *TableChange
		want   []string
	}{
		{
			name: "create table with generated column stored",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "products",
					Columns: []parser.Column{
						{Name: "price", Type: "numeric"},
						{Name: "quantity", Type: "integer"},
						{Name: "total", Type: "numeric", GeneratedAs: "price * quantity", GeneratedType: "STORED"},
					},
				},
			},
			want: []string{"GENERATED ALWAYS AS (price * quantity)", "STORED"},
		},
		{
			name: "create table with generated column virtual",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "products",
					Columns: []parser.Column{
						{Name: "price", Type: "numeric"},
						{Name: "quantity", Type: "integer"},
						{Name: "total", Type: "numeric", GeneratedAs: "price * quantity", GeneratedType: "VIRTUAL"},
					},
				},
			},
			want: []string{"GENERATED ALWAYS AS (price * quantity)", "VIRTUAL"},
		},
		{
			name: "alter column set generated expression",
			change: &TableChange{
				ChangeType: AlterTable,
				Table:      parser.Table{Name: "products"},
				AlterColumns: []ColumnAlteration{
					{
						Column:    parser.Column{Name: "total", Type: "numeric", GeneratedAs: "price * quantity * 1.1"},
						OldColumn: parser.Column{Name: "total", Type: "numeric", GeneratedAs: "price * quantity"},
						Changes:   []string{"generated"},
					},
				},
			},
			want: []string{"SET EXPRESSION AS (price * quantity * 1.1)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.change.SQL()
			for _, want := range tt.want {
				if !strings.Contains(sql, want) {
					t.Errorf("SQL = %q, should contain %q", sql, want)
				}
			}
		})
	}
}

func TestTableChange_SQL_ExclusionConstraint(t *testing.T) {
	change := &TableChange{
		ChangeType: CreateTable,
		Table: parser.Table{
			Name: "bookings",
			Columns: []parser.Column{
				{Name: "room_id", Type: "integer"},
				{Name: "during", Type: "tsrange"},
			},
			Constraints: []parser.Constraint{
				{
					Name:               "no_overlap",
					Type:               "EXCLUSION",
					Columns:            []string{"room_id", "during"},
					ExclusionUsing:     "gist",
					ExclusionOperators: map[string]string{"room_id": "=", "during": "&&"},
				},
			},
		},
	}

	sql := change.SQL()
	wants := []string{"EXCLUDE USING gist", "room_id", "during", "WITH =", "WITH &&"}
	for _, want := range wants {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL = %q, should contain %q", sql, want)
		}
	}
}

func TestTableChange_SQL_TemporalConstraint(t *testing.T) {
	tests := []struct {
		name   string
		change *TableChange
		want   []string
	}{
		{
			name: "temporal primary key with without overlaps",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "employee_positions",
					Columns: []parser.Column{
						{Name: "employee_id", Type: "integer"},
						{Name: "valid_period", Type: "tsrange"},
					},
					Constraints: []parser.Constraint{
						{
							Name:            "pk_employee_period",
							Type:            "PRIMARY KEY",
							Columns:         []string{"employee_id"},
							WithoutOverlaps: true,
							PeriodColumn:    "valid_period",
						},
					},
				},
			},
			want: []string{"PRIMARY KEY", "employee_id", "valid_period", "WITHOUT OVERLAPS"},
		},
		{
			name: "temporal unique with without overlaps",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "room_reservations",
					Columns: []parser.Column{
						{Name: "room_id", Type: "integer"},
						{Name: "reservation_period", Type: "tsrange"},
					},
					Constraints: []parser.Constraint{
						{
							Name:            "unique_room_period",
							Type:            "UNIQUE",
							Columns:         []string{"room_id"},
							WithoutOverlaps: true,
							PeriodColumn:    "reservation_period",
						},
					},
				},
			},
			want: []string{"UNIQUE", "room_id", "reservation_period", "WITHOUT OVERLAPS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.change.SQL()
			for _, want := range tt.want {
				if !strings.Contains(sql, want) {
					t.Errorf("SQL = %q, should contain %q", sql, want)
				}
			}
		})
	}
}

func TestTableChange_SQL_NotEnforcedConstraint(t *testing.T) {
	tests := []struct {
		name   string
		change *TableChange
		want   []string
	}{
		{
			name: "foreign key not enforced",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "orders",
					Columns: []parser.Column{
						{Name: "id", Type: "integer"},
						{Name: "user_id", Type: "integer"},
					},
					Constraints: []parser.Constraint{
						{
							Name:        "fk_user",
							Type:        "FOREIGN KEY",
							Columns:     []string{"user_id"},
							RefTable:    "users",
							RefColumns:  []string{"id"},
							NotEnforced: true,
						},
					},
				},
			},
			want: []string{"FOREIGN KEY", "NOT ENFORCED"},
		},
		{
			name: "foreign key not valid",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "orders",
					Columns: []parser.Column{
						{Name: "id", Type: "integer"},
						{Name: "user_id", Type: "integer"},
					},
					Constraints: []parser.Constraint{
						{
							Name:       "fk_user",
							Type:       "FOREIGN KEY",
							Columns:    []string{"user_id"},
							RefTable:   "users",
							RefColumns: []string{"id"},
							NotValid:   true,
						},
					},
				},
			},
			want: []string{"FOREIGN KEY", "NOT VALID"},
		},
		{
			name: "check constraint not enforced",
			change: &TableChange{
				ChangeType: CreateTable,
				Table: parser.Table{
					Name: "products",
					Columns: []parser.Column{
						{Name: "price", Type: "numeric"},
					},
					Constraints: []parser.Constraint{
						{
							Name:        "price_positive",
							Type:        "CHECK",
							Check:       "price > 0",
							NotEnforced: true,
						},
					},
				},
			},
			want: []string{"CHECK (price > 0)", "NOT ENFORCED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.change.SQL()
			for _, want := range tt.want {
				if !strings.Contains(sql, want) {
					t.Errorf("SQL = %q, should contain %q", sql, want)
				}
			}
		})
	}
}

func TestTableChange_SQL_NamedNotNullConstraint(t *testing.T) {
	change := &TableChange{
		ChangeType: CreateTable,
		Table: parser.Table{
			Name: "users",
			Columns: []parser.Column{
				{Name: "email", Type: "text", Nullable: false, NotNullConstraintName: "email_not_null"},
			},
		},
	}

	sql := change.SQL()
	if !strings.Contains(sql, "CONSTRAINT") || !strings.Contains(sql, "email_not_null") || !strings.Contains(sql, "NOT NULL") {
		t.Errorf("SQL = %q, should contain named NOT NULL constraint", sql)
	}
}

func TestCompareColumn_Identity(t *testing.T) {
	addIdentity := compareColumn(
		parser.Column{Name: "id", Type: "bigint", Identity: ""},
		parser.Column{Name: "id", Type: "bigint", Identity: "ALWAYS"},
	)
	if addIdentity == nil {
		t.Error("adding identity should be detected")
	}
	found := false
	for _, c := range addIdentity.Changes {
		if c == "identity" {
			found = true
		}
	}
	if !found {
		t.Error("identity change should be in changes list")
	}

	changeIdentityOpts := compareColumn(
		parser.Column{Name: "id", Type: "bigint", Identity: "ALWAYS", IdentityStart: 1, IdentityIncrement: 1},
		parser.Column{Name: "id", Type: "bigint", Identity: "ALWAYS", IdentityStart: 1000, IdentityIncrement: 10},
	)
	if changeIdentityOpts == nil {
		t.Error("changing identity options should be detected")
	}
	found = false
	for _, c := range changeIdentityOpts.Changes {
		if c == "identity_options" {
			found = true
		}
	}
	if !found {
		t.Error("identity_options change should be in changes list")
	}
}

func TestCompareColumn_Generated(t *testing.T) {
	addGenerated := compareColumn(
		parser.Column{Name: "total", Type: "numeric", GeneratedAs: ""},
		parser.Column{Name: "total", Type: "numeric", GeneratedAs: "price * qty"},
	)
	if addGenerated == nil {
		t.Error("adding generated expression should be detected")
	}

	changeGeneratedType := compareColumn(
		parser.Column{Name: "total", Type: "numeric", GeneratedAs: "price * qty", GeneratedType: "STORED"},
		parser.Column{Name: "total", Type: "numeric", GeneratedAs: "price * qty", GeneratedType: "VIRTUAL"},
	)
	if changeGeneratedType == nil {
		t.Error("changing generated type should be detected")
	}
}
