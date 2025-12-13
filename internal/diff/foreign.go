package diff

import (
	"fmt"
	"strings"

	"github.com/terminally-online/shrugged/internal/parser"
)

type ForeignDataWrapperChange struct {
	ChangeType            ChangeType
	ForeignDataWrapper    parser.ForeignDataWrapper
	OldForeignDataWrapper *parser.ForeignDataWrapper
}

func (c *ForeignDataWrapperChange) SQL() string {
	switch c.ChangeType {
	case CreateForeignDataWrapper:
		sql := fmt.Sprintf("CREATE FOREIGN DATA WRAPPER %s", quoteIdent(c.ForeignDataWrapper.Name))
		if c.ForeignDataWrapper.Handler != "" {
			sql += fmt.Sprintf(" HANDLER %s", c.ForeignDataWrapper.Handler)
		}
		if c.ForeignDataWrapper.Validator != "" {
			sql += fmt.Sprintf(" VALIDATOR %s", c.ForeignDataWrapper.Validator)
		}
		if len(c.ForeignDataWrapper.Options) > 0 {
			sql += " OPTIONS (" + formatOptions(c.ForeignDataWrapper.Options) + ")"
		}
		return sql + ";"
	case DropForeignDataWrapper:
		return fmt.Sprintf("DROP FOREIGN DATA WRAPPER %s;", quoteIdent(c.ForeignDataWrapper.Name))
	}
	return ""
}

func (c *ForeignDataWrapperChange) DownSQL() string {
	switch c.ChangeType {
	case CreateForeignDataWrapper:
		return fmt.Sprintf("DROP FOREIGN DATA WRAPPER %s;", quoteIdent(c.ForeignDataWrapper.Name))
	case DropForeignDataWrapper:
		if c.OldForeignDataWrapper != nil {
			sql := fmt.Sprintf("CREATE FOREIGN DATA WRAPPER %s", quoteIdent(c.OldForeignDataWrapper.Name))
			if c.OldForeignDataWrapper.Handler != "" {
				sql += fmt.Sprintf(" HANDLER %s", c.OldForeignDataWrapper.Handler)
			}
			if c.OldForeignDataWrapper.Validator != "" {
				sql += fmt.Sprintf(" VALIDATOR %s", c.OldForeignDataWrapper.Validator)
			}
			if len(c.OldForeignDataWrapper.Options) > 0 {
				sql += " OPTIONS (" + formatOptions(c.OldForeignDataWrapper.Options) + ")"
			}
			return sql + ";"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped FDW %s", c.ForeignDataWrapper.Name)
	}
	return ""
}

func (c *ForeignDataWrapperChange) Type() ChangeType   { return c.ChangeType }
func (c *ForeignDataWrapperChange) ObjectName() string { return c.ForeignDataWrapper.Name }
func (c *ForeignDataWrapperChange) IsReversible() bool {
	if c.ChangeType == CreateForeignDataWrapper {
		return true
	}
	return c.OldForeignDataWrapper != nil
}

type ForeignServerChange struct {
	ChangeType       ChangeType
	ForeignServer    parser.ForeignServer
	OldForeignServer *parser.ForeignServer
}

func (c *ForeignServerChange) SQL() string {
	switch c.ChangeType {
	case CreateForeignServer:
		sql := fmt.Sprintf("CREATE SERVER %s", quoteIdent(c.ForeignServer.Name))
		if c.ForeignServer.Type != "" {
			sql += fmt.Sprintf(" TYPE '%s'", c.ForeignServer.Type)
		}
		if c.ForeignServer.Version != "" {
			sql += fmt.Sprintf(" VERSION '%s'", c.ForeignServer.Version)
		}
		sql += fmt.Sprintf(" FOREIGN DATA WRAPPER %s", quoteIdent(c.ForeignServer.FDW))
		if len(c.ForeignServer.Options) > 0 {
			sql += " OPTIONS (" + formatOptions(c.ForeignServer.Options) + ")"
		}
		return sql + ";"
	case DropForeignServer:
		return fmt.Sprintf("DROP SERVER %s;", quoteIdent(c.ForeignServer.Name))
	}
	return ""
}

func (c *ForeignServerChange) DownSQL() string {
	switch c.ChangeType {
	case CreateForeignServer:
		return fmt.Sprintf("DROP SERVER %s;", quoteIdent(c.ForeignServer.Name))
	case DropForeignServer:
		if c.OldForeignServer != nil {
			sql := fmt.Sprintf("CREATE SERVER %s", quoteIdent(c.OldForeignServer.Name))
			if c.OldForeignServer.Type != "" {
				sql += fmt.Sprintf(" TYPE '%s'", c.OldForeignServer.Type)
			}
			if c.OldForeignServer.Version != "" {
				sql += fmt.Sprintf(" VERSION '%s'", c.OldForeignServer.Version)
			}
			sql += fmt.Sprintf(" FOREIGN DATA WRAPPER %s", quoteIdent(c.OldForeignServer.FDW))
			if len(c.OldForeignServer.Options) > 0 {
				sql += " OPTIONS (" + formatOptions(c.OldForeignServer.Options) + ")"
			}
			return sql + ";"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped server %s", c.ForeignServer.Name)
	}
	return ""
}

func (c *ForeignServerChange) Type() ChangeType   { return c.ChangeType }
func (c *ForeignServerChange) ObjectName() string { return c.ForeignServer.Name }
func (c *ForeignServerChange) IsReversible() bool {
	if c.ChangeType == CreateForeignServer {
		return true
	}
	return c.OldForeignServer != nil
}

type ForeignTableChange struct {
	ChangeType      ChangeType
	ForeignTable    parser.ForeignTable
	OldForeignTable *parser.ForeignTable
}

func (c *ForeignTableChange) SQL() string {
	ftName := qualifiedName(c.ForeignTable.Schema, c.ForeignTable.Name)
	switch c.ChangeType {
	case CreateForeignTable:
		var cols []string
		for _, col := range c.ForeignTable.Columns {
			colDef := fmt.Sprintf("%s %s", quoteIdent(col.Name), col.Type)
			if !col.Nullable {
				colDef += " NOT NULL"
			}
			cols = append(cols, colDef)
		}
		sql := fmt.Sprintf("CREATE FOREIGN TABLE %s (\n    %s\n) SERVER %s",
			ftName,
			strings.Join(cols, ",\n    "),
			quoteIdent(c.ForeignTable.Server))
		if len(c.ForeignTable.Options) > 0 {
			sql += " OPTIONS (" + formatOptions(c.ForeignTable.Options) + ")"
		}
		return sql + ";"
	case DropForeignTable:
		return fmt.Sprintf("DROP FOREIGN TABLE %s;", ftName)
	}
	return ""
}

func (c *ForeignTableChange) DownSQL() string {
	ftName := qualifiedName(c.ForeignTable.Schema, c.ForeignTable.Name)
	switch c.ChangeType {
	case CreateForeignTable:
		return fmt.Sprintf("DROP FOREIGN TABLE %s;", ftName)
	case DropForeignTable:
		if c.OldForeignTable != nil {
			var cols []string
			for _, col := range c.OldForeignTable.Columns {
				colDef := fmt.Sprintf("%s %s", quoteIdent(col.Name), col.Type)
				if !col.Nullable {
					colDef += " NOT NULL"
				}
				cols = append(cols, colDef)
			}
			sql := fmt.Sprintf("CREATE FOREIGN TABLE %s (\n    %s\n) SERVER %s",
				qualifiedName(c.OldForeignTable.Schema, c.OldForeignTable.Name),
				strings.Join(cols, ",\n    "),
				quoteIdent(c.OldForeignTable.Server))
			if len(c.OldForeignTable.Options) > 0 {
				sql += " OPTIONS (" + formatOptions(c.OldForeignTable.Options) + ")"
			}
			return sql + ";"
		}
		return fmt.Sprintf("-- IRREVERSIBLE: Cannot restore dropped foreign table %s", c.ForeignTable.Name)
	}
	return ""
}

func (c *ForeignTableChange) Type() ChangeType   { return c.ChangeType }
func (c *ForeignTableChange) ObjectName() string { return c.ForeignTable.Name }
func (c *ForeignTableChange) IsReversible() bool {
	if c.ChangeType == CreateForeignTable {
		return true
	}
	return c.OldForeignTable != nil
}

func compareForeignDataWrappers(current, desired []parser.ForeignDataWrapper) []Change {
	var changes []Change
	currentMap := make(map[string]parser.ForeignDataWrapper)
	for _, f := range current {
		currentMap[f.Name] = f
	}
	for _, f := range desired {
		if _, exists := currentMap[f.Name]; !exists {
			changes = append(changes, &ForeignDataWrapperChange{ChangeType: CreateForeignDataWrapper, ForeignDataWrapper: f})
		}
	}
	desiredMap := make(map[string]bool)
	for _, f := range desired {
		desiredMap[f.Name] = true
	}
	for _, f := range current {
		if !desiredMap[f.Name] {
			oldFDW := f
			changes = append(changes, &ForeignDataWrapperChange{ChangeType: DropForeignDataWrapper, ForeignDataWrapper: f, OldForeignDataWrapper: &oldFDW})
		}
	}
	return changes
}

func compareForeignServers(current, desired []parser.ForeignServer) []Change {
	var changes []Change
	currentMap := make(map[string]parser.ForeignServer)
	for _, s := range current {
		currentMap[s.Name] = s
	}
	for _, s := range desired {
		if _, exists := currentMap[s.Name]; !exists {
			changes = append(changes, &ForeignServerChange{ChangeType: CreateForeignServer, ForeignServer: s})
		}
	}
	desiredMap := make(map[string]bool)
	for _, s := range desired {
		desiredMap[s.Name] = true
	}
	for _, s := range current {
		if !desiredMap[s.Name] {
			oldServer := s
			changes = append(changes, &ForeignServerChange{ChangeType: DropForeignServer, ForeignServer: s, OldForeignServer: &oldServer})
		}
	}
	return changes
}

func compareForeignTables(current, desired []parser.ForeignTable) []Change {
	var changes []Change
	currentMap := make(map[string]parser.ForeignTable)
	for _, ft := range current {
		currentMap[ft.Name] = ft
	}
	for _, ft := range desired {
		if _, exists := currentMap[ft.Name]; !exists {
			changes = append(changes, &ForeignTableChange{ChangeType: CreateForeignTable, ForeignTable: ft})
		}
	}
	desiredMap := make(map[string]bool)
	for _, ft := range desired {
		desiredMap[ft.Name] = true
	}
	for _, ft := range current {
		if !desiredMap[ft.Name] {
			oldFT := ft
			changes = append(changes, &ForeignTableChange{ChangeType: DropForeignTable, ForeignTable: ft, OldForeignTable: &oldFT})
		}
	}
	return changes
}

func formatOptions(opts map[string]string) string {
	var parts []string
	for k, v := range opts {
		parts = append(parts, fmt.Sprintf("%s '%s'", k, v))
	}
	return strings.Join(parts, ", ")
}
