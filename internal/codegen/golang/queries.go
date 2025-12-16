package golang

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/terminally-online/shrugged/internal/parser"
)

func GenerateQueries(queries []parser.Query, outDir string, modelsPackage string, modelsDir string, schema *parser.Schema) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := generateQuerierInterface(outDir); err != nil {
		return err
	}

	customTypes := buildCustomTypeSet(schema)
	extensionFields := loadAllExtensionFields(modelsDir)

	for _, q := range queries {
		if err := generateQueryFile(q, outDir, modelsPackage, customTypes, schema, extensionFields); err != nil {
			return err
		}
	}

	return nil
}

func loadAllExtensionFields(modelsDir string) map[string][]StructField {
	result := make(map[string][]StructField)

	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(modelsDir, entry.Name())
		extensions, err := ExtractExtensionFields(filePath)
		if err != nil {
			continue
		}

		for name, fields := range extensions {
			result[name] = fields
		}
	}

	return result
}

func buildCustomTypeSet(schema *parser.Schema) map[string]bool {
	types := make(map[string]bool)
	if schema == nil {
		return types
	}
	for _, e := range schema.Enums {
		types[toPascalCase(e.Name)] = true
	}
	for _, c := range schema.CompositeTypes {
		types[toPascalCase(c.Name)] = true
	}
	return types
}

type ModelMatch struct {
	Table           *parser.Table
	ExtensionFields map[string]bool
}

func findMatchingModel(q parser.Query, schema *parser.Schema, extensionFields map[string][]StructField) *ModelMatch {
	if schema == nil || len(q.Columns) == 0 {
		return nil
	}

	if q.ResultType != parser.QueryResultRow && q.ResultType != parser.QueryResultRows {
		return nil
	}

	queryColNames := make(map[string]bool)
	for _, col := range q.Columns {
		queryColNames[col.Name] = true
	}

	for i := range schema.Tables {
		table := &schema.Tables[i]
		tableName := toPascalCase(table.Name)
		extensionName := tableName + "Extension"

		tableColNames := make(map[string]bool)
		for _, col := range table.Columns {
			tableColNames[col.Name] = true
		}

		extensionColNames := make(map[string]bool)
		if ef, ok := extensionFields[extensionName]; ok {
			for _, f := range ef {
				extensionColNames[toSnakeCase(f.Name)] = true
			}
		}

		unmatchedQueryCols := []string{}
		for name := range queryColNames {
			if !tableColNames[name] {
				unmatchedQueryCols = append(unmatchedQueryCols, name)
			}
		}

		unmatchedTableCols := []string{}
		for name := range tableColNames {
			if !queryColNames[name] {
				unmatchedTableCols = append(unmatchedTableCols, name)
			}
		}

		if len(unmatchedTableCols) > 0 {
			continue
		}

		if len(unmatchedQueryCols) == 0 {
			return &ModelMatch{Table: table, ExtensionFields: nil}
		}

		allExtensionsMatched := true
		matchedExtensions := make(map[string]bool)
		for _, col := range unmatchedQueryCols {
			if extensionColNames[col] {
				matchedExtensions[col] = true
			} else {
				allExtensionsMatched = false
				break
			}
		}

		if allExtensionsMatched {
			return &ModelMatch{Table: table, ExtensionFields: matchedExtensions}
		}
	}

	return nil
}

func generateQuerierInterface(outDir string) error {
	content := `package queries

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Queries struct {
	db Querier
}

func New(db Querier) *Queries {
	return &Queries{db: db}
}

func (q *Queries) WithTx(tx pgx.Tx) *Queries {
	return &Queries{db: tx}
}
`
	filePath := filepath.Join(outDir, "querier.go")
	return os.WriteFile(filePath, []byte(content), 0644)
}

func generateQueryFile(q parser.Query, outDir string, modelsPackage string, customTypes map[string]bool, schema *parser.Schema, extensionFields map[string][]StructField) error {
	var sb strings.Builder

	match := findMatchingModel(q, schema, extensionFields)
	imports := collectQueryImports(q, modelsPackage, customTypes, match)

	sb.WriteString("package queries\n\n")

	if len(imports) > 0 {
		sb.WriteString("import (\n")
		for _, imp := range imports {
			sb.WriteString(fmt.Sprintf("\t%q\n", imp))
		}
		sb.WriteString(")\n\n")
	}

	needsResultStruct := needsCustomResultStruct(q) && match == nil
	if needsResultStruct {
		sb.WriteString(generateResultStruct(q, modelsPackage, customTypes))
		sb.WriteString("\n")
	}

	sb.WriteString(generateQueryConstant(q))
	sb.WriteString("\n")

	sb.WriteString(generateQueryFunction(q, modelsPackage, needsResultStruct, customTypes, match))

	fileName := toSnakeCaseLower(q.Name) + ".go"
	filePath := filepath.Join(outDir, fileName)
	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

func collectQueryImports(q parser.Query, modelsPackage string, customTypes map[string]bool, match *ModelMatch) []string {
	importSet := make(map[string]bool)
	importSet["context"] = true

	needsModels := match != nil

	switch q.ResultType {
	case parser.QueryResultRow, parser.QueryResultRows:
		for _, col := range q.Columns {
			if col.Import != "" && match == nil {
				importSet[col.Import] = true
			}
			if col.IsJSONAgg {
				needsModels = true
			}
			if isCustomType(col.GoType, customTypes) {
				needsModels = true
			}
		}
	}

	for _, p := range q.Parameters {
		if p.Import != "" {
			importSet[p.Import] = true
		}
		if strings.Contains(p.GoType, "time.") {
			importSet["time"] = true
		}
		if isCustomType(p.GoType, customTypes) {
			needsModels = true
		}
	}

	if needsModels && modelsPackage != "" {
		importSet[modelsPackage] = true
	}

	var imports []string
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return imports
}

func isCustomType(goType string, customTypes map[string]bool) bool {
	goType = strings.TrimPrefix(goType, "*")
	return customTypes[goType]
}

func needsCustomResultStruct(q parser.Query) bool {
	if q.ResultType == parser.QueryResultExec || q.ResultType == parser.QueryResultExecRows {
		return false
	}

	if len(q.Columns) == 0 {
		return false
	}

	return true
}

func generateResultStruct(q parser.Query, modelsPackage string, customTypes map[string]bool) string {
	var sb strings.Builder

	structName := q.Name + "Row"
	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for _, col := range q.Columns {
		fieldName := toPascalCase(col.Name)
		fieldType := col.GoType

		if col.IsJSONAgg && col.JSONElemGoType != "" {
			if modelsPackage != "" {
				parts := strings.Split(modelsPackage, "/")
				pkgName := parts[len(parts)-1]
				fieldType = fmt.Sprintf("[]%s.%s", pkgName, col.JSONElemGoType)
			} else {
				fieldType = "[]" + col.JSONElemGoType
			}
		} else {
			fieldType = prefixCustomType(fieldType, modelsPackage, customTypes)
		}

		jsonTag := toSnakeCase(col.Name)
		if col.Nullable {
			jsonTag += ",omitempty"
		}
		tag := fmt.Sprintf("`json:%q`", jsonTag)
		sb.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, tag))
	}

	sb.WriteString("}\n")
	return sb.String()
}

func prefixCustomType(goType string, modelsPackage string, customTypes map[string]bool) string {
	if modelsPackage == "" {
		return goType
	}

	isPointer := strings.HasPrefix(goType, "*")
	baseType := strings.TrimPrefix(goType, "*")

	if customTypes[baseType] {
		parts := strings.Split(modelsPackage, "/")
		pkgName := parts[len(parts)-1]
		if isPointer {
			return "*" + pkgName + "." + baseType
		}
		return pkgName + "." + baseType
	}

	return goType
}

func generateQueryConstant(q parser.Query) string {
	constName := toSnakeCaseLower(q.Name) + "SQL"
	return fmt.Sprintf("const %s = `\n%s`\n", constName, q.PreparedSQL)
}

func generateQueryFunction(q parser.Query, modelsPackage string, needsResultStruct bool, customTypes map[string]bool, match *ModelMatch) string {
	var sb strings.Builder

	funcName := q.Name
	constName := toSnakeCaseLower(q.Name) + "SQL"

	var structName string
	if match != nil && modelsPackage != "" {
		parts := strings.Split(modelsPackage, "/")
		pkgName := parts[len(parts)-1]
		structName = pkgName + "." + toPascalCase(match.Table.Name)
	} else {
		structName = q.Name + "Row"
	}

	params := []string{"ctx context.Context"}
	for _, p := range q.Parameters {
		paramType := p.GoType
		if paramType == "" {
			paramType = "interface{}"
		}
		paramType = prefixCustomType(paramType, modelsPackage, customTypes)
		if p.Nullable && !strings.HasPrefix(paramType, "*") {
			paramType = "*" + paramType
		}
		params = append(params, fmt.Sprintf("%s %s", p.Name, paramType))
	}

	var returnType string
	switch q.ResultType {
	case parser.QueryResultRow:
		returnType = fmt.Sprintf("(*%s, error)", structName)
	case parser.QueryResultRows:
		returnType = fmt.Sprintf("([]%s, error)", structName)
	case parser.QueryResultExec:
		returnType = "error"
	case parser.QueryResultExecRows:
		returnType = "(int64, error)"
	}

	sb.WriteString(fmt.Sprintf("func (q *Queries) %s(%s) %s {\n", funcName, strings.Join(params, ", "), returnType))

	args := make([]string, len(q.Parameters))
	for i, p := range q.Parameters {
		args[i] = p.Name
	}
	argsStr := strings.Join(args, ", ")

	switch q.ResultType {
	case parser.QueryResultRow:
		sb.WriteString(generateRowQuery(q, constName, structName, argsStr, match))
	case parser.QueryResultRows:
		sb.WriteString(generateRowsQuery(q, constName, structName, argsStr, match))
	case parser.QueryResultExec:
		sb.WriteString(generateExecQuery(constName, argsStr))
	case parser.QueryResultExecRows:
		sb.WriteString(generateExecRowsQuery(constName, argsStr))
	}

	sb.WriteString("}\n")
	return sb.String()
}

func generateRowQuery(q parser.Query, constName, structName, argsStr string, match *ModelMatch) string {
	var sb strings.Builder

	if argsStr != "" {
		sb.WriteString(fmt.Sprintf("\trow := q.db.QueryRow(ctx, %s, %s)\n\n", constName, argsStr))
	} else {
		sb.WriteString(fmt.Sprintf("\trow := q.db.QueryRow(ctx, %s)\n\n", constName))
	}

	sb.WriteString(fmt.Sprintf("\tvar result %s\n", structName))

	var jsonAggCols []parser.QueryColumn
	for _, col := range q.Columns {
		if col.IsJSONAgg && !isExtensionField(col.Name, match) {
			jsonAggCols = append(jsonAggCols, col)
		}
	}

	if len(jsonAggCols) > 0 {
		for _, col := range jsonAggCols {
			varName := toSnakeCaseLower(col.Name) + "JSON"
			sb.WriteString(fmt.Sprintf("\tvar %s []byte\n", varName))
		}
		sb.WriteString("\n")
	}

	scanArgs := generateScanArgsWithMatch(q.Columns, "result", match)
	sb.WriteString(fmt.Sprintf("\terr := row.Scan(%s)\n", scanArgs))
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString("\t\treturn nil, err\n")
	sb.WriteString("\t}\n")

	for _, col := range jsonAggCols {
		varName := toSnakeCaseLower(col.Name) + "JSON"
		fieldName := toPascalCase(col.Name)
		sb.WriteString(fmt.Sprintf("\n\tif %s != nil {\n", varName))
		sb.WriteString(fmt.Sprintf("\t\tif err := json.Unmarshal(%s, &result.%s); err != nil {\n", varName, fieldName))
		sb.WriteString("\t\t\treturn nil, err\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t}\n")
	}

	sb.WriteString("\n\treturn &result, nil\n")
	return sb.String()
}

func isExtensionField(colName string, match *ModelMatch) bool {
	if match == nil || match.ExtensionFields == nil {
		return false
	}
	return match.ExtensionFields[colName]
}

func generateRowsQuery(q parser.Query, constName, structName, argsStr string, match *ModelMatch) string {
	var sb strings.Builder

	if argsStr != "" {
		sb.WriteString(fmt.Sprintf("\trows, err := q.db.Query(ctx, %s, %s)\n", constName, argsStr))
	} else {
		sb.WriteString(fmt.Sprintf("\trows, err := q.db.Query(ctx, %s)\n", constName))
	}
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString("\t\treturn nil, err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tdefer rows.Close()\n\n")

	sb.WriteString(fmt.Sprintf("\tvar result []%s\n", structName))
	sb.WriteString("\tfor rows.Next() {\n")
	sb.WriteString(fmt.Sprintf("\t\tvar item %s\n", structName))

	var jsonAggCols []parser.QueryColumn
	for _, col := range q.Columns {
		if col.IsJSONAgg && !isExtensionField(col.Name, match) {
			jsonAggCols = append(jsonAggCols, col)
		}
	}

	if len(jsonAggCols) > 0 {
		for _, col := range jsonAggCols {
			varName := toSnakeCaseLower(col.Name) + "JSON"
			sb.WriteString(fmt.Sprintf("\t\tvar %s []byte\n", varName))
		}
	}

	scanArgs := generateScanArgsWithMatch(q.Columns, "item", match)
	sb.WriteString(fmt.Sprintf("\t\terr := rows.Scan(%s)\n", scanArgs))
	sb.WriteString("\t\tif err != nil {\n")
	sb.WriteString("\t\t\treturn nil, err\n")
	sb.WriteString("\t\t}\n")

	for _, col := range jsonAggCols {
		varName := toSnakeCaseLower(col.Name) + "JSON"
		fieldName := toPascalCase(col.Name)
		sb.WriteString(fmt.Sprintf("\t\tif %s != nil {\n", varName))
		sb.WriteString(fmt.Sprintf("\t\t\tif err := json.Unmarshal(%s, &item.%s); err != nil {\n", varName, fieldName))
		sb.WriteString("\t\t\t\treturn nil, err\n")
		sb.WriteString("\t\t\t}\n")
		sb.WriteString("\t\t}\n")
	}

	sb.WriteString("\t\tresult = append(result, item)\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString("\tif err := rows.Err(); err != nil {\n")
	sb.WriteString("\t\treturn nil, err\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString("\treturn result, nil\n")
	return sb.String()
}

func generateExecQuery(constName, argsStr string) string {
	var sb strings.Builder

	if argsStr != "" {
		sb.WriteString(fmt.Sprintf("\t_, err := q.db.Exec(ctx, %s, %s)\n", constName, argsStr))
	} else {
		sb.WriteString(fmt.Sprintf("\t_, err := q.db.Exec(ctx, %s)\n", constName))
	}
	sb.WriteString("\treturn err\n")

	return sb.String()
}

func generateExecRowsQuery(constName, argsStr string) string {
	var sb strings.Builder

	if argsStr != "" {
		sb.WriteString(fmt.Sprintf("\tresult, err := q.db.Exec(ctx, %s, %s)\n", constName, argsStr))
	} else {
		sb.WriteString(fmt.Sprintf("\tresult, err := q.db.Exec(ctx, %s)\n", constName))
	}
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString("\t\treturn 0, err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn result.RowsAffected(), nil\n")

	return sb.String()
}

func generateScanArgsWithMatch(cols []parser.QueryColumn, varName string, match *ModelMatch) string {
	var args []string
	for _, col := range cols {
		fieldName := toPascalCase(col.Name)
		if col.IsJSONAgg && !isExtensionField(col.Name, match) {
			jsonVarName := toSnakeCaseLower(col.Name) + "JSON"
			args = append(args, "&"+jsonVarName)
		} else {
			args = append(args, "&"+varName+"."+fieldName)
		}
	}
	return strings.Join(args, ", ")
}

func toSnakeCaseLower(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				prevLower := runes[i-1] >= 'a' && runes[i-1] <= 'z'
				nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'

				if prevLower || nextLower {
					result.WriteRune('_')
				}
			}
			result.WriteRune(r + 32)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
