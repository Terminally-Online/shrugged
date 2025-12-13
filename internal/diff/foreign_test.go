package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateForeignDataWrapper(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		ForeignDataWrappers: []parser.ForeignDataWrapper{
			{Name: "postgres_fdw", Handler: "postgres_fdw_handler", Validator: "postgres_fdw_validator"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateForeignDataWrapper && c.ObjectName() == "postgres_fdw" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateForeignDataWrapper change for postgres_fdw")
	}
}

func TestCompare_DropForeignDataWrapper(t *testing.T) {
	current := &parser.Schema{
		ForeignDataWrappers: []parser.ForeignDataWrapper{
			{Name: "old_fdw"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropForeignDataWrapper && c.ObjectName() == "old_fdw" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropForeignDataWrapper change for old_fdw")
	}
}

func TestForeignDataWrapperChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *ForeignDataWrapperChange
		want   []string
	}{
		{
			name: "create fdw basic",
			change: &ForeignDataWrapperChange{
				ChangeType:         CreateForeignDataWrapper,
				ForeignDataWrapper: parser.ForeignDataWrapper{Name: "my_fdw"},
			},
			want: []string{"CREATE FOREIGN DATA WRAPPER", "my_fdw"},
		},
		{
			name: "create fdw with handler and validator",
			change: &ForeignDataWrapperChange{
				ChangeType: CreateForeignDataWrapper,
				ForeignDataWrapper: parser.ForeignDataWrapper{
					Name:      "postgres_fdw",
					Handler:   "postgres_fdw_handler",
					Validator: "postgres_fdw_validator",
				},
			},
			want: []string{"CREATE FOREIGN DATA WRAPPER", "postgres_fdw", "HANDLER", "postgres_fdw_handler", "VALIDATOR", "postgres_fdw_validator"},
		},
		{
			name: "drop fdw",
			change: &ForeignDataWrapperChange{
				ChangeType:         DropForeignDataWrapper,
				ForeignDataWrapper: parser.ForeignDataWrapper{Name: "old_fdw"},
			},
			want: []string{"DROP FOREIGN DATA WRAPPER", "old_fdw"},
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

func TestCompare_CreateForeignServer(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		ForeignServers: []parser.ForeignServer{
			{Name: "remote_server", FDW: "postgres_fdw"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateForeignServer && c.ObjectName() == "remote_server" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateForeignServer change for remote_server")
	}
}

func TestForeignServerChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *ForeignServerChange
		want   []string
	}{
		{
			name: "create server basic",
			change: &ForeignServerChange{
				ChangeType: CreateForeignServer,
				ForeignServer: parser.ForeignServer{
					Name: "my_server",
					FDW:  "postgres_fdw",
				},
			},
			want: []string{"CREATE SERVER", "my_server", "FOREIGN DATA WRAPPER", "postgres_fdw"},
		},
		{
			name: "create server with type and version",
			change: &ForeignServerChange{
				ChangeType: CreateForeignServer,
				ForeignServer: parser.ForeignServer{
					Name:    "typed_server",
					FDW:     "file_fdw",
					Type:    "file_server",
					Version: "1.0",
				},
			},
			want: []string{"CREATE SERVER", "typed_server", "TYPE 'file_server'", "VERSION '1.0'"},
		},
		{
			name: "drop server",
			change: &ForeignServerChange{
				ChangeType:    DropForeignServer,
				ForeignServer: parser.ForeignServer{Name: "old_server"},
			},
			want: []string{"DROP SERVER", "old_server"},
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

func TestCompare_CreateForeignTable(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		ForeignTables: []parser.ForeignTable{
			{
				Name:    "remote_users",
				Server:  "remote_server",
				Columns: []parser.Column{{Name: "id", Type: "integer"}, {Name: "name", Type: "text"}},
			},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateForeignTable && c.ObjectName() == "remote_users" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateForeignTable change for remote_users")
	}
}

func TestForeignTableChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *ForeignTableChange
		want   []string
	}{
		{
			name: "create foreign table",
			change: &ForeignTableChange{
				ChangeType: CreateForeignTable,
				ForeignTable: parser.ForeignTable{
					Name:   "remote_users",
					Server: "remote_server",
					Columns: []parser.Column{
						{Name: "id", Type: "integer", Nullable: false},
						{Name: "name", Type: "text", Nullable: true},
					},
				},
			},
			want: []string{"CREATE FOREIGN TABLE", "remote_users", "id", "integer", "NOT NULL", "name", "text", "SERVER", "remote_server"},
		},
		{
			name: "create foreign table with schema",
			change: &ForeignTableChange{
				ChangeType: CreateForeignTable,
				ForeignTable: parser.ForeignTable{
					Schema: "external",
					Name:   "remote_data",
					Server: "data_server",
					Columns: []parser.Column{
						{Name: "value", Type: "text"},
					},
				},
			},
			want: []string{"CREATE FOREIGN TABLE", "external", "remote_data"},
		},
		{
			name: "drop foreign table",
			change: &ForeignTableChange{
				ChangeType:   DropForeignTable,
				ForeignTable: parser.ForeignTable{Name: "old_table"},
			},
			want: []string{"DROP FOREIGN TABLE", "old_table"},
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

func TestForeignDataWrapperChange_IsReversible(t *testing.T) {
	createChange := &ForeignDataWrapperChange{
		ChangeType:         CreateForeignDataWrapper,
		ForeignDataWrapper: parser.ForeignDataWrapper{Name: "test"},
	}
	if !createChange.IsReversible() {
		t.Error("CreateForeignDataWrapper should be reversible")
	}

	oldFDW := parser.ForeignDataWrapper{Name: "test"}
	dropChangeWithOld := &ForeignDataWrapperChange{
		ChangeType:            DropForeignDataWrapper,
		ForeignDataWrapper:    parser.ForeignDataWrapper{Name: "test"},
		OldForeignDataWrapper: &oldFDW,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropForeignDataWrapper with OldForeignDataWrapper should be reversible")
	}

	dropChangeNoOld := &ForeignDataWrapperChange{
		ChangeType:         DropForeignDataWrapper,
		ForeignDataWrapper: parser.ForeignDataWrapper{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropForeignDataWrapper without OldForeignDataWrapper should not be reversible")
	}
}

func TestForeignServerChange_IsReversible(t *testing.T) {
	createChange := &ForeignServerChange{
		ChangeType:    CreateForeignServer,
		ForeignServer: parser.ForeignServer{Name: "test", FDW: "postgres_fdw"},
	}
	if !createChange.IsReversible() {
		t.Error("CreateForeignServer should be reversible")
	}
}

func TestForeignTableChange_IsReversible(t *testing.T) {
	createChange := &ForeignTableChange{
		ChangeType:   CreateForeignTable,
		ForeignTable: parser.ForeignTable{Name: "test", Server: "srv"},
	}
	if !createChange.IsReversible() {
		t.Error("CreateForeignTable should be reversible")
	}
}
