package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateDomain(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Domains: []parser.Domain{
			{Name: "email_address", Type: "text", Check: "CHECK (VALUE ~ '^.+@.+$')"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateDomain && c.ObjectName() == "email_address" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateDomain change for email_address")
	}
}

func TestCompare_DropDomain(t *testing.T) {
	current := &parser.Schema{
		Domains: []parser.Domain{
			{Name: "old_domain", Type: "integer"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropDomain && c.ObjectName() == "old_domain" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropDomain change for old_domain")
	}
}

func TestDomainChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *DomainChange
		want   []string
	}{
		{
			name: "create domain basic",
			change: &DomainChange{
				ChangeType: CreateDomain,
				Domain:     parser.Domain{Name: "positive_int", Type: "integer"},
			},
			want: []string{"CREATE DOMAIN", "positive_int", "AS", "integer"},
		},
		{
			name: "create domain with all options",
			change: &DomainChange{
				ChangeType: CreateDomain,
				Domain: parser.Domain{
					Name:      "email",
					Type:      "text",
					Collation: "en_US",
					Default:   "'unknown@example.com'",
					NotNull:   true,
					Check:     "CHECK (VALUE ~ '^.+@.+$')",
				},
			},
			want: []string{"CREATE DOMAIN", "email", "AS", "text", "COLLATE", "DEFAULT", "NOT NULL", "CHECK"},
		},
		{
			name: "create domain with schema",
			change: &DomainChange{
				ChangeType: CreateDomain,
				Domain:     parser.Domain{Schema: "myschema", Name: "my_domain", Type: "text"},
			},
			want: []string{"CREATE DOMAIN", "myschema", "my_domain"},
		},
		{
			name: "drop domain",
			change: &DomainChange{
				ChangeType: DropDomain,
				Domain:     parser.Domain{Name: "old_domain", Type: "integer"},
			},
			want: []string{"DROP DOMAIN", "old_domain"},
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

func TestDomainChange_DownSQL(t *testing.T) {
	createChange := &DomainChange{
		ChangeType: CreateDomain,
		Domain:     parser.Domain{Name: "my_domain", Type: "text"},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP DOMAIN") {
		t.Error("DownSQL for CreateDomain should contain DROP DOMAIN")
	}

	oldDomain := parser.Domain{Name: "old_domain", Type: "integer", NotNull: true}
	dropChange := &DomainChange{
		ChangeType: DropDomain,
		Domain:     parser.Domain{Name: "old_domain", Type: "integer"},
		OldDomain:  &oldDomain,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE DOMAIN") {
		t.Error("DownSQL for DropDomain with OldDomain should contain CREATE DOMAIN")
	}
	if !strings.Contains(downSQL, "NOT NULL") {
		t.Error("DownSQL for DropDomain should preserve NOT NULL constraint")
	}

	dropChangeNoOld := &DomainChange{
		ChangeType: DropDomain,
		Domain:     parser.Domain{Name: "old_domain", Type: "integer"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropDomain without OldDomain should indicate IRREVERSIBLE")
	}
}

func TestDomainChange_IsReversible(t *testing.T) {
	createChange := &DomainChange{
		ChangeType: CreateDomain,
		Domain:     parser.Domain{Name: "test", Type: "text"},
	}
	if !createChange.IsReversible() {
		t.Error("CreateDomain should be reversible")
	}

	oldDomain := parser.Domain{Name: "test", Type: "text"}
	dropChangeWithOld := &DomainChange{
		ChangeType: DropDomain,
		Domain:     parser.Domain{Name: "test", Type: "text"},
		OldDomain:  &oldDomain,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropDomain with OldDomain should be reversible")
	}

	dropChangeNoOld := &DomainChange{
		ChangeType: DropDomain,
		Domain:     parser.Domain{Name: "test", Type: "text"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropDomain without OldDomain should not be reversible")
	}
}
