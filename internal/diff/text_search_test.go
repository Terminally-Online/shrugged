package diff

import (
	"strings"
	"testing"

	"shrugged/internal/parser"
)

func TestCompare_CreateTextSearchConfig(t *testing.T) {
	current := &parser.Schema{}
	desired := &parser.Schema{
		TextSearchConfigs: []parser.TextSearchConfig{
			{Name: "english_config", Parser: "pg_catalog.default"},
		},
	}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == CreateTextSearchConfig && c.ObjectName() == "english_config" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected CreateTextSearchConfig change for english_config")
	}
}

func TestCompare_DropTextSearchConfig(t *testing.T) {
	current := &parser.Schema{
		TextSearchConfigs: []parser.TextSearchConfig{
			{Name: "old_config", Parser: "pg_catalog.default"},
		},
	}
	desired := &parser.Schema{}

	changes := Compare(current, desired)

	found := false
	for _, c := range changes {
		if c.Type() == DropTextSearchConfig && c.ObjectName() == "old_config" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected DropTextSearchConfig change for old_config")
	}
}

func TestTextSearchConfigChange_SQL(t *testing.T) {
	tests := []struct {
		name   string
		change *TextSearchConfigChange
		want   []string
	}{
		{
			name: "create text search config",
			change: &TextSearchConfigChange{
				ChangeType: CreateTextSearchConfig,
				TextSearchConfig: parser.TextSearchConfig{
					Name:   "my_config",
					Parser: "pg_catalog.default",
				},
			},
			want: []string{"CREATE TEXT SEARCH CONFIGURATION", "my_config", "PARSER = pg_catalog.default"},
		},
		{
			name: "create text search config with schema",
			change: &TextSearchConfigChange{
				ChangeType: CreateTextSearchConfig,
				TextSearchConfig: parser.TextSearchConfig{
					Schema: "myschema",
					Name:   "my_config",
					Parser: "pg_catalog.default",
				},
			},
			want: []string{"CREATE TEXT SEARCH CONFIGURATION", "myschema", "my_config"},
		},
		{
			name: "drop text search config",
			change: &TextSearchConfigChange{
				ChangeType:       DropTextSearchConfig,
				TextSearchConfig: parser.TextSearchConfig{Name: "old_config"},
			},
			want: []string{"DROP TEXT SEARCH CONFIGURATION", "old_config"},
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

func TestTextSearchConfigChange_DownSQL(t *testing.T) {
	createChange := &TextSearchConfigChange{
		ChangeType: CreateTextSearchConfig,
		TextSearchConfig: parser.TextSearchConfig{
			Name:   "my_config",
			Parser: "pg_catalog.default",
		},
	}
	if !strings.Contains(createChange.DownSQL(), "DROP TEXT SEARCH CONFIGURATION") {
		t.Error("DownSQL for CreateTextSearchConfig should contain DROP TEXT SEARCH CONFIGURATION")
	}

	oldConfig := parser.TextSearchConfig{Name: "old_config", Parser: "pg_catalog.default"}
	dropChange := &TextSearchConfigChange{
		ChangeType:          DropTextSearchConfig,
		TextSearchConfig:    parser.TextSearchConfig{Name: "old_config"},
		OldTextSearchConfig: &oldConfig,
	}
	downSQL := dropChange.DownSQL()
	if !strings.Contains(downSQL, "CREATE TEXT SEARCH CONFIGURATION") {
		t.Error("DownSQL for DropTextSearchConfig with OldTextSearchConfig should contain CREATE TEXT SEARCH CONFIGURATION")
	}
	if !strings.Contains(downSQL, "PARSER = pg_catalog.default") {
		t.Error("DownSQL for DropTextSearchConfig should preserve PARSER")
	}

	dropChangeNoOld := &TextSearchConfigChange{
		ChangeType:       DropTextSearchConfig,
		TextSearchConfig: parser.TextSearchConfig{Name: "old_config"},
	}
	downSQLNoOld := dropChangeNoOld.DownSQL()
	if !strings.Contains(downSQLNoOld, "IRREVERSIBLE") {
		t.Error("DownSQL for DropTextSearchConfig without OldTextSearchConfig should indicate IRREVERSIBLE")
	}
}

func TestTextSearchConfigChange_IsReversible(t *testing.T) {
	createChange := &TextSearchConfigChange{
		ChangeType: CreateTextSearchConfig,
		TextSearchConfig: parser.TextSearchConfig{
			Name:   "test",
			Parser: "pg_catalog.default",
		},
	}
	if !createChange.IsReversible() {
		t.Error("CreateTextSearchConfig should be reversible")
	}

	oldConfig := parser.TextSearchConfig{Name: "test", Parser: "pg_catalog.default"}
	dropChangeWithOld := &TextSearchConfigChange{
		ChangeType:          DropTextSearchConfig,
		TextSearchConfig:    parser.TextSearchConfig{Name: "test"},
		OldTextSearchConfig: &oldConfig,
	}
	if !dropChangeWithOld.IsReversible() {
		t.Error("DropTextSearchConfig with OldTextSearchConfig should be reversible")
	}

	dropChangeNoOld := &TextSearchConfigChange{
		ChangeType:       DropTextSearchConfig,
		TextSearchConfig: parser.TextSearchConfig{Name: "test"},
	}
	if dropChangeNoOld.IsReversible() {
		t.Error("DropTextSearchConfig without OldTextSearchConfig should not be reversible")
	}
}
