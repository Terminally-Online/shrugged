package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateRole(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Roles: []parser.Role{
			{Name: "app_user", Login: true, Inherit: true},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateRole && c.ObjectName() == "app_user" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateRole change for app_user")
	}
}

func TestCompare_DropRole(t *testing.T) {
	current := &parser.Schema{
		Roles: []parser.Role{
			{Name: "old_user", Login: true},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropRole && c.ObjectName() == "old_user" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropRole change for old_user")
	}
}

func TestCompare_AlterRole(t *testing.T) {
	current := &parser.Schema{
		Roles: []parser.Role{
			{Name: "app_user", Login: true, SuperUser: false},
		},
	}
	desired := &parser.Schema{
		Roles: []parser.Role{
			{Name: "app_user", Login: true, SuperUser: true},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == AlterRole && c.ObjectName() == "app_user" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected AlterRole change for app_user")
	}
}

func TestRoleChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *RoleChange
		want   []string
	}{
		{
			name: "create role basic",
			change: &RoleChange{
				ChangeType: CreateRole,
				Role:       parser.Role{Name: "app_user", Inherit: true},
			},
			want: []string{"CREATE ROLE", "app_user"},
		},
		{
			name: "create role with all options",
			change: &RoleChange{
				ChangeType: CreateRole,
				Role: parser.Role{
					Name:            "admin_user",
					SuperUser:       true,
					CreateDB:        true,
					CreateRole:      true,
					Inherit:         false,
					Login:           true,
					Replication:     true,
					BypassRLS:       true,
					ConnectionLimit: 10,
					ValidUntil:      "2025-12-31",
				},
			},
			want: []string{"CREATE ROLE", "admin_user", "SUPERUSER", "CREATEDB", "CREATEROLE", "NOINHERIT", "LOGIN", "REPLICATION", "BYPASSRLS", "CONNECTION LIMIT 10", "VALID UNTIL"},
		},
		{
			name: "create role with membership",
			change: &RoleChange{
				ChangeType: CreateRole,
				Role: parser.Role{
					Name:    "team_member",
					Login:   true,
					Inherit: true,
					InRoles: []string{"team_role", "reader_role"},
				},
			},
			want: []string{"CREATE ROLE", "team_member", "GRANT", "team_role", "reader_role"},
		},
		{
			name: "drop role",
			change: &RoleChange{
				ChangeType: DropRole,
				Role:       parser.Role{Name: "old_user"},
			},
			want: []string{"DROP ROLE", "old_user"},
		},
		{
			name: "alter role superuser",
			change: &RoleChange{
				ChangeType: AlterRole,
				Role:       parser.Role{Name: "app_user", SuperUser: true},
				OldRole:    &parser.Role{Name: "app_user", SuperUser: false},
			},
			want: []string{"ALTER ROLE", "app_user", "SUPERUSER"},
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

func TestRoleChange_DownSQL(t *testing.T) {
	createChange := &RoleChange{
		ChangeType: CreateRole,
		Role:       parser.Role{Name: "app_user", Login: true, Inherit: true},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP ROLE") {
		t.Error("DownSQL for CreateRole should contain DROP ROLE")
	}

	oldRole := parser.Role{Name: "old_user", Login: true, SuperUser: true}
	dropChange := &RoleChange{
		ChangeType: DropRole,
		Role:       parser.Role{Name: "old_user"},
		OldRole:    &oldRole,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE ROLE") {
		t.Error("DownSQL for DropRole with OldRole should contain CREATE ROLE")
	}
	if !strings.Contains(downSQL, "SUPERUSER") {
		t.Error("DownSQL for DropRole should preserve SUPERUSER")
	}

	dropChangeNoOld := &RoleChange{
		ChangeType: DropRole,
		Role:       parser.Role{Name: "old_user"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropRole without OldRole should indicate IRREVERSIBLE")
	}
}

func TestRoleChange_IsReversible(t *testing.T) {
	createChange := &RoleChange{
		ChangeType: CreateRole,
		Role:       parser.Role{Name: "test", Login: true, Inherit: true},
	}
	if !createChange.IsReversible() {
		t.Error("CreateRole should be reversible")
	}

	oldRole := parser.Role{Name: "test", Login: true}
	dropChangeWithOld := &RoleChange{
		ChangeType: DropRole,
		Role:       parser.Role{Name: "test"},
		OldRole:    &oldRole,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropRole with OldRole should be reversible")
	}

	dropChangeNoOld := &RoleChange{
		ChangeType: DropRole,
		Role:       parser.Role{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropRole without OldRole should not be reversible")
	}
}

func TestCompare_CreateRoleGrant(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		RoleGrants: []parser.RoleGrant{
			{Privilege: "SELECT", ObjectName: "users", Grantee: "app_user"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateRoleGrant {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateRoleGrant change")
	}
}

func TestRoleGrantChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *RoleGrantChange
		want   []string
	}{
		{
			name: "create grant",
			change: &RoleGrantChange{
				ChangeType: CreateRoleGrant,
				RoleGrant: parser.RoleGrant{
					Privilege:  "SELECT",
					ObjectName: "users",
					Grantee:    "app_user",
				},
			},
			want: []string{"GRANT", "SELECT", "ON", "users", "TO", "app_user"},
		},
		{
			name: "create grant on type",
			change: &RoleGrantChange{
				ChangeType: CreateRoleGrant,
				RoleGrant: parser.RoleGrant{
					ObjectType: "TYPE",
					Privilege:  "USAGE",
					Schema:     "public",
					ObjectName: "contract",
					Grantee:    "PUBLIC",
				},
			},
			want: []string{"GRANT USAGE ON TYPE", "contract", "TO PUBLIC;"},
		},
		{
			name: "create grant on function",
			change: &RoleGrantChange{
				ChangeType: CreateRoleGrant,
				RoleGrant: parser.RoleGrant{
					ObjectType: "FUNCTION",
					Privilege:  "EXECUTE",
					Schema:     "public",
					ObjectName: "my_func",
					Grantee:    "app_user",
				},
			},
			want: []string{"GRANT EXECUTE ON FUNCTION", "my_func", "TO"},
		},
		{
			name: "create grant with grant option",
			change: &RoleGrantChange{
				ChangeType: CreateRoleGrant,
				RoleGrant: parser.RoleGrant{
					Privilege:  "ALL",
					ObjectName: "data",
					Grantee:    "admin",
					WithGrant:  true,
				},
			},
			want: []string{"GRANT", "ALL", "WITH GRANT OPTION"},
		},
		{
			name: "drop grant",
			change: &RoleGrantChange{
				ChangeType: DropRoleGrant,
				RoleGrant: parser.RoleGrant{
					Privilege:  "SELECT",
					ObjectName: "users",
					Grantee:    "app_user",
				},
			},
			want: []string{"REVOKE", "SELECT", "FROM", "app_user"},
		},
		{
			name: "revoke grant on type",
			change: &RoleGrantChange{
				ChangeType: DropRoleGrant,
				RoleGrant: parser.RoleGrant{
					ObjectType: "TYPE",
					Privilege:  "USAGE",
					Schema:     "public",
					ObjectName: "contract",
					Grantee:    "PUBLIC",
				},
			},
			want: []string{"REVOKE USAGE ON TYPE", "contract", "FROM"},
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

func TestDefaultPrivilegeChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *DefaultPrivilegeChange
		want   []string
	}{
		{
			name: "create default privilege",
			change: &DefaultPrivilegeChange{
				ChangeType: CreateDefaultPrivilege,
				DefaultPrivilege: parser.DefaultPrivilege{
					Schema:     "public",
					Role:       "admin",
					ObjectType: "TABLES",
					Privileges: []string{"SELECT", "INSERT"},
					Grantee:    "app_user",
				},
			},
			want: []string{"ALTER DEFAULT PRIVILEGES", "FOR ROLE", "admin", "IN SCHEMA", "public", "GRANT", "SELECT, INSERT", "ON", "TABLES", "TO", "app_user"},
		},
		{
			name: "drop default privilege",
			change: &DefaultPrivilegeChange{
				ChangeType: DropDefaultPrivilege,
				DefaultPrivilege: parser.DefaultPrivilege{
					Schema:     "public",
					Role:       "admin",
					ObjectType: "TABLES",
					Privileges: []string{"SELECT"},
					Grantee:    "app_user",
				},
			},
			want: []string{"ALTER DEFAULT PRIVILEGES", "REVOKE", "SELECT", "FROM", "app_user"},
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

func TestRoleGrantChange_IsReversible(t *testing.T) {
	change := &RoleGrantChange{
		ChangeType: CreateRoleGrant,
		RoleGrant: parser.RoleGrant{
			Privilege:  "SELECT",
			ObjectName: "users",
			Grantee:    "app_user",
		},
	}
	if !change.IsReversible() {
		t.Error("RoleGrantChange should be reversible")
	}
}

func TestDefaultPrivilegeChange_IsReversible(t *testing.T) {
	change := &DefaultPrivilegeChange{
		ChangeType: CreateDefaultPrivilege,
		DefaultPrivilege: parser.DefaultPrivilege{
			Schema:     "public",
			Role:       "admin",
			ObjectType: "TABLES",
			Privileges: []string{"SELECT"},
			Grantee:    "app_user",
		},
	}
	if !change.IsReversible() {
		t.Error("DefaultPrivilegeChange should be reversible")
	}
}
