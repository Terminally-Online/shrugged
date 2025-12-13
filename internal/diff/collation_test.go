package diff

import (
	"strings"
	"testing"

	"github.com/terminally-online/shrugged/internal/parser"
)

func TestCompare_CreateCollation(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		Collations: []parser.Collation{
			{Name: "my_collation", Provider: "icu", Locale: "en-US-u-ks-level2"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateCollation && c.ObjectName() == "my_collation" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateCollation change for my_collation")
	}
}

func TestCompare_DropCollation(t *testing.T) {
	current := &parser.Schema{
		Collations: []parser.Collation{
			{Name: "old_collation", Provider: "libc", LcCollate: "en_US.UTF-8"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropCollation && c.ObjectName() == "old_collation" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropCollation change for old_collation")
	}
}

func TestCollationChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *CollationChange
		want   []string
	}{
		{
			name: "create collation with provider and locale",
			change: &CollationChange{
				ChangeType: CreateCollation,
				Collation: parser.Collation{
					Name:     "my_collation",
					Provider: "icu",
					Locale:   "en-US",
				},
			},
			want: []string{"CREATE COLLATION", "my_collation", "PROVIDER = icu", "LOCALE = 'en-US'"},
		},
		{
			name: "create collation with lc_collate and lc_ctype",
			change: &CollationChange{
				ChangeType: CreateCollation,
				Collation: parser.Collation{
					Name:      "posix_collation",
					Provider:  "libc",
					LcCollate: "en_US.UTF-8",
					LcCtype:   "en_US.UTF-8",
				},
			},
			want: []string{"CREATE COLLATION", "posix_collation", "PROVIDER = libc", "LC_COLLATE = 'en_US.UTF-8'", "LC_CTYPE = 'en_US.UTF-8'"},
		},
		{
			name: "create collation with schema",
			change: &CollationChange{
				ChangeType: CreateCollation,
				Collation: parser.Collation{
					Schema:   "myschema",
					Name:     "my_collation",
					Provider: "icu",
					Locale:   "de-DE",
				},
			},
			want: []string{"CREATE COLLATION", "myschema", "my_collation"},
		},
		{
			name: "drop collation",
			change: &CollationChange{
				ChangeType: DropCollation,
				Collation:  parser.Collation{Name: "old_collation"},
			},
			want: []string{"DROP COLLATION", "old_collation"},
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

func TestCollationChange_DownSQL(t *testing.T) {
	createChange := &CollationChange{
		ChangeType: CreateCollation,
		Collation: parser.Collation{
			Name:     "my_collation",
			Provider: "icu",
			Locale:   "en-US",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP COLLATION") {
		t.Error("DownSQL for CreateCollation should contain DROP COLLATION")
	}

	oldCollation := parser.Collation{Name: "old_collation", Provider: "icu", Locale: "de-DE"}
	dropChange := &CollationChange{
		ChangeType:   DropCollation,
		Collation:    parser.Collation{Name: "old_collation"},
		OldCollation: &oldCollation,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE COLLATION") {
		t.Error("DownSQL for DropCollation with OldCollation should contain CREATE COLLATION")
	}
	if !strings.Contains(downSQL, "LOCALE = 'de-DE'") {
		t.Error("DownSQL for DropCollation should preserve LOCALE")
	}

	dropChangeNoOld := &CollationChange{
		ChangeType: DropCollation,
		Collation:  parser.Collation{Name: "old_collation"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropCollation without OldCollation should indicate IRREVERSIBLE")
	}
}

func TestCollationChange_IsReversible(t *testing.T) {
	createChange := &CollationChange{
		ChangeType: CreateCollation,
		Collation: parser.Collation{
			Name:     "test",
			Provider: "icu",
			Locale:   "en-US",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateCollation should be reversible")
	}

	oldCollation := parser.Collation{Name: "test", Provider: "icu", Locale: "en-US"}
	dropChangeWithOld := &CollationChange{
		ChangeType:   DropCollation,
		Collation:    parser.Collation{Name: "test"},
		OldCollation: &oldCollation,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropCollation with OldCollation should be reversible")
	}

	dropChangeNoOld := &CollationChange{
		ChangeType: DropCollation,
		Collation:  parser.Collation{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropCollation without OldCollation should not be reversible")
	}
}
