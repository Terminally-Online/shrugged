package golang

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/terminally-online/shrugged/internal/codegen"
	"github.com/terminally-online/shrugged/internal/parser"
)

func init() {
	codegen.Register(&GoGenerator{})
}

type GoGenerator struct{}

func (g *GoGenerator) Language() string {
	return "go"
}

func (g *GoGenerator) Generate(schema *parser.Schema, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, enum := range schema.Enums {
		if err := g.generateEnum(enum, outDir); err != nil {
			return fmt.Errorf("failed to generate enum %s: %w", enum.Name, err)
		}
	}

	for _, ct := range schema.CompositeTypes {
		if err := g.generateCompositeType(ct, outDir); err != nil {
			return fmt.Errorf("failed to generate composite type %s: %w", ct.Name, err)
		}
	}

	for _, table := range schema.Tables {
		if table.Name == "shrugged_migrations" {
			continue
		}
		if err := g.generateTable(table, outDir); err != nil {
			return fmt.Errorf("failed to generate table %s: %w", table.Name, err)
		}
	}

	return nil
}

func (g *GoGenerator) GenerateQueries(opts codegen.QueryGenOptions) ([]string, error) {
	return generateQueries(opts.Queries, opts.OutDir, opts.ModelsPackage, opts.ModelsDir, opts.Schema, opts.Clean)
}

func (g *GoGenerator) generateEnum(enum parser.Enum, outDir string) error {
	typeName := toPascalCase(enum.Name)
	fileName := toSnakeCase(enum.Name) + ".go"
	filePath := filepath.Join(outDir, fileName)

	var values []EnumValue
	for _, value := range enum.Values {
		values = append(values, EnumValue{
			Name:  typeName + toPascalCase(value),
			Value: value,
		})
	}

	if fileExists(filePath) {
		content, err := mergeEnumFile(filePath, typeName, values)
		if err != nil {
			return fmt.Errorf("failed to merge enum file: %w", err)
		}
		return os.WriteFile(filePath, content, 0o644)
	}

	var sb strings.Builder
	sb.WriteString("package models\n\n")
	fmt.Fprintf(&sb, "type %s string\n\n", typeName)
	sb.WriteString("const (\n")

	for _, v := range values {
		fmt.Fprintf(&sb, "\t%s %s = %q\n", v.Name, typeName, v.Value)
	}

	sb.WriteString(")\n")

	return os.WriteFile(filePath, []byte(sb.String()), 0o644)
}

func (g *GoGenerator) generateCompositeType(ct parser.CompositeType, outDir string) error {
	typeName := toPascalCase(ct.Name)
	fileName := toSnakeCase(ct.Name) + ".go"
	filePath := filepath.Join(outDir, fileName)

	var fields []StructField
	var imports []string
	importSet := make(map[string]bool)

	for _, attr := range ct.Attributes {
		goType, imp := pgTypeToGo(attr.Type, attr.Nullable, nil)
		if imp != "" && !importSet[imp] {
			imports = append(imports, imp)
			importSet[imp] = true
		}
		fields = append(fields, StructField{
			Name: toPascalCase(attr.Name),
			Type: goType,
		})
	}

	if fileExists(filePath) {
		content, err := mergeStructFile(filePath, typeName, fields, imports)
		if err != nil {
			return fmt.Errorf("failed to merge composite type file: %w", err)
		}
		return os.WriteFile(filePath, content, 0o644)
	}

	var sb strings.Builder
	sb.WriteString("package models\n\n")

	if len(imports) > 0 {
		sb.WriteString("import (\n")
		for _, imp := range imports {
			fmt.Fprintf(&sb, "\t%q\n", imp)
		}
		sb.WriteString(")\n\n")
	}

	fmt.Fprintf(&sb, "type %s struct {\n", typeName)
	for _, field := range fields {
		fmt.Fprintf(&sb, "\t%s %s\n", field.Name, field.Type)
	}
	sb.WriteString("}\n")

	return os.WriteFile(filePath, []byte(sb.String()), 0o644)
}

func (g *GoGenerator) generateTable(table parser.Table, outDir string) error {
	typeName := toPascalCase(table.Name)
	extensionTypeName := typeName + "Extension"
	fileName := toSnakeCase(table.Name) + ".go"
	filePath := filepath.Join(outDir, fileName)

	var fields []StructField
	var imports []string
	importSet := make(map[string]bool)

	for _, col := range table.Columns {
		goType, imp := pgTypeToGo(col.Type, col.Nullable, nil)
		if imp != "" && !importSet[imp] {
			imports = append(imports, imp)
			importSet[imp] = true
		}
		fieldName := toPascalCase(col.Name)
		jsonTag := toSnakeCase(col.Name)
		if col.Nullable {
			jsonTag += ",omitempty"
		}
		jsonTag = fmt.Sprintf(`json:"%s"`, jsonTag)
		fields = append(fields, StructField{
			Name: fieldName,
			Type: goType,
			Tag:  jsonTag,
		})
	}

	fields = append(fields, StructField{
		Name: extensionTypeName,
		Type: "",
		Tag:  "",
	})

	if fileExists(filePath) {
		content, err := mergeTableFile(filePath, typeName, extensionTypeName, fields, imports)
		if err != nil {
			return fmt.Errorf("failed to merge table file: %w", err)
		}
		return os.WriteFile(filePath, content, 0o644)
	}

	var sb strings.Builder
	sb.WriteString("package models\n\n")

	if len(imports) > 0 {
		sb.WriteString("import (\n")
		for _, imp := range imports {
			fmt.Fprintf(&sb, "\t%q\n", imp)
		}
		sb.WriteString(")\n\n")
	}

	fmt.Fprintf(&sb, "type %s struct {}\n\n", extensionTypeName)

	fmt.Fprintf(&sb, "type %s struct {\n", typeName)
	for _, field := range fields[:len(fields)-1] {
		if field.Tag != "" {
			fmt.Fprintf(&sb, "\t%s %s `%s`\n", field.Name, field.Type, field.Tag)
		} else {
			fmt.Fprintf(&sb, "\t%s %s\n", field.Name, field.Type)
		}
	}
	fmt.Fprintf(&sb, "\t%s\n", extensionTypeName)
	sb.WriteString("}\n")

	return os.WriteFile(filePath, []byte(sb.String()), 0o644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func pgTypeToGo(pgType string, nullable bool, schema *parser.Schema) (goType string, importPath string) {
	pgType = strings.ToLower(strings.TrimSpace(pgType))

	isArray := strings.HasSuffix(pgType, "[]") || strings.HasPrefix(pgType, "array")
	if isArray {
		pgType = strings.TrimSuffix(pgType, "[]")
		pgType = strings.TrimPrefix(pgType, "array[")
		pgType = strings.TrimSuffix(pgType, "]")
	}

	if strings.HasPrefix(pgType, "character varying") || strings.HasPrefix(pgType, "varchar") {
		pgType = "varchar"
	}
	if strings.HasPrefix(pgType, "character(") || strings.HasPrefix(pgType, "char(") {
		pgType = "char"
	}
	if strings.HasPrefix(pgType, "numeric") || strings.HasPrefix(pgType, "decimal") {
		pgType = "numeric"
	}
	if strings.HasPrefix(pgType, "timestamp") {
		if strings.Contains(pgType, "with time zone") {
			pgType = "timestamptz"
		} else {
			pgType = "timestamp"
		}
	}
	if strings.HasPrefix(pgType, "time ") {
		if strings.Contains(pgType, "with time zone") {
			pgType = "timetz"
		} else {
			pgType = "time"
		}
	}

	var baseType string
	var imp string

	switch pgType {
	case "integer", "int", "int4":
		baseType = "int32"
	case "bigint", "int8":
		baseType = "int64"
	case "smallint", "int2":
		baseType = "int16"
	case "real", "float4":
		baseType = "float32"
	case "double precision", "float8":
		baseType = "float64"
	case "boolean", "bool":
		baseType = "bool"
	case "text", "varchar", "char", "name":
		baseType = "string"
	case "bytea":
		baseType = "[]byte"
	case "uuid":
		baseType = "string"
	case "json", "jsonb":
		baseType = "json.RawMessage"
		imp = "encoding/json"
	case "timestamp", "timestamptz", "date", "time", "timetz":
		baseType = "time.Time"
		imp = "time"
	case "interval":
		baseType = "string"
	case "numeric", "money":
		baseType = "string"
	case "inet", "cidr", "macaddr", "macaddr8":
		baseType = "string"
	case "bit", "bit varying", "varbit":
		baseType = "string"
	case "xml":
		baseType = "string"
	case "point", "line", "lseg", "box", "path", "polygon", "circle":
		baseType = "string"
	case "tsquery", "tsvector":
		baseType = "string"
	case "oid":
		baseType = "uint32"
	default:
		if schema != nil {
			for _, e := range schema.Enums {
				if strings.ToLower(e.Name) == pgType {
					baseType = toPascalCase(e.Name)
					goto done
				}
			}
			for _, c := range schema.CompositeTypes {
				if strings.ToLower(c.Name) == pgType {
					baseType = toPascalCase(c.Name)
					goto done
				}
			}
		}
		baseType = toPascalCase(pgType)
	}

done:
	if isArray {
		if baseType == "[]byte" {
			goType = "[][]byte"
		} else {
			goType = "[]" + baseType
		}
	} else if nullable && baseType != "[]byte" && !strings.HasPrefix(baseType, "json.") {
		goType = "*" + baseType
	} else {
		goType = baseType
	}

	return goType, imp
}
