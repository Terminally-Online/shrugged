package introspect

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/jackc/pgx/v5"

	"github.com/terminally-online/shrugged/internal/parser"
)

var stmtCounter atomic.Uint64

var jsonAggTableRegex = regexp.MustCompile(`(?i)(json_agg|jsonb_agg)\s*\(\s*(\w+)\s*\.\s*\*\s*\)`)

func Queries(ctx context.Context, databaseURL string, queries []parser.Query, schema *parser.Schema) ([]parser.Query, error) {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	typeMap, err := buildTypeMap(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to build type map: %w", err)
	}

	result := make([]parser.Query, len(queries))
	for i, q := range queries {
		introspected, err := introspectQuery(ctx, conn, q, schema, typeMap)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect query %s: %w", q.Name, err)
		}
		result[i] = introspected
	}

	return result, nil
}

func buildTypeMap(ctx context.Context, conn *pgx.Conn) (map[uint32]string, error) {
	typeMap := make(map[uint32]string)

	rows, err := conn.Query(ctx, `
		SELECT t.oid, t.typname
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname = 'public'
		   OR t.typtype IN ('b', 'e', 'c')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var oid uint32
		var typname string
		if err := rows.Scan(&oid, &typname); err != nil {
			return nil, err
		}
		typeMap[oid] = typname
	}

	return typeMap, rows.Err()
}

func introspectQuery(ctx context.Context, conn *pgx.Conn, query parser.Query, schema *parser.Schema, typeMap map[uint32]string) (parser.Query, error) {
	if query.ResultType == parser.QueryResultExec || query.ResultType == parser.QueryResultExecRows {
		return introspectExecQuery(ctx, conn, query, schema, typeMap)
	}

	stmtName := fmt.Sprintf("shrugged_introspect_%d_%s", stmtCounter.Add(1), query.Name)

	sd, err := conn.Prepare(ctx, stmtName, query.PreparedSQL)
	if err != nil {
		return query, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(ctx, fmt.Sprintf("DEALLOCATE %s", stmtName))
	}()

	for i := range query.Parameters {
		if i < len(sd.ParamOIDs) {
			oid := sd.ParamOIDs[i]
			pgType := resolveTypeName(oid, typeMap)
			goType, imp := pgTypeToGo(pgType, false, schema)
			query.Parameters[i].Type = pgType
			query.Parameters[i].GoType = goType
			if imp != "" {
				query.Parameters[i].Import = imp
			}
		}
	}

	jsonAggColumns := detectJSONAggColumns(query.SQL, schema)

	query.Columns = make([]parser.QueryColumn, len(sd.Fields))
	for i, field := range sd.Fields {
		pgType := resolveTypeName(field.DataTypeOID, typeMap)
		nullable := true

		if jsonAggInfo, ok := jsonAggColumns[field.Name]; ok {
			goType, imp := pgTypeToGo(pgType, false, schema)
			query.Columns[i] = parser.QueryColumn{
				Name:           field.Name,
				Type:           pgType,
				GoType:         goType,
				Import:         imp,
				Nullable:       nullable,
				IsJSONAgg:      true,
				JSONElemType:   jsonAggInfo.tableName,
				JSONElemGoType: jsonAggInfo.goType,
			}
		} else {
			goType, imp := pgTypeToGo(pgType, nullable, schema)
			query.Columns[i] = parser.QueryColumn{
				Name:     field.Name,
				Type:     pgType,
				GoType:   goType,
				Import:   imp,
				Nullable: nullable,
			}
		}
	}

	return query, nil
}

func introspectExecQuery(ctx context.Context, conn *pgx.Conn, query parser.Query, schema *parser.Schema, typeMap map[uint32]string) (parser.Query, error) {
	stmtName := fmt.Sprintf("shrugged_introspect_%d_%s", stmtCounter.Add(1), query.Name)

	sd, err := conn.Prepare(ctx, stmtName, query.PreparedSQL)
	if err != nil {
		return query, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(ctx, fmt.Sprintf("DEALLOCATE %s", stmtName))
	}()

	for i := range query.Parameters {
		if i < len(sd.ParamOIDs) {
			oid := sd.ParamOIDs[i]
			pgType := resolveTypeName(oid, typeMap)
			goType, imp := pgTypeToGo(pgType, false, schema)
			query.Parameters[i].Type = pgType
			query.Parameters[i].GoType = goType
			if imp != "" {
				query.Parameters[i].Import = imp
			}
		}
	}

	return query, nil
}

func resolveTypeName(oid uint32, typeMap map[uint32]string) string {
	if name := oidToTypeName(oid); name != "unknown" {
		return name
	}
	if name, ok := typeMap[oid]; ok {
		return name
	}
	return "unknown"
}

type jsonAggInfo struct {
	tableName string
	goType    string
}

func detectJSONAggColumns(sql string, schema *parser.Schema) map[string]jsonAggInfo {
	result := make(map[string]jsonAggInfo)

	tableAliases := extractTableAliases(sql)

	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		matches := jsonAggTableRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				alias := match[2]

				tableName := alias
				if actual, ok := tableAliases[alias]; ok {
					tableName = actual
				}

				var goType string
				for _, t := range schema.Tables {
					if t.Name == tableName {
						goType = toPascalCase(tableName)
						break
					}
				}

				columnName := extractColumnAlias(line, match[0])
				if columnName != "" && goType != "" {
					result[columnName] = jsonAggInfo{
						tableName: tableName,
						goType:    goType,
					}
				}
			}
		}
	}

	return result
}

func extractTableAliases(sql string) map[string]string {
	aliases := make(map[string]string)

	fromRegex := regexp.MustCompile(`(?i)\bFROM\s+(\w+)\s+(?:AS\s+)?(\w+)`)
	joinRegex := regexp.MustCompile(`(?i)\bJOIN\s+(\w+)\s+(?:AS\s+)?(\w+)`)

	for _, match := range fromRegex.FindAllStringSubmatch(sql, -1) {
		if len(match) >= 3 {
			aliases[match[2]] = match[1]
		}
	}
	for _, match := range joinRegex.FindAllStringSubmatch(sql, -1) {
		if len(match) >= 3 {
			aliases[match[2]] = match[1]
		}
	}

	return aliases
}

func extractColumnAlias(line string, jsonAggExpr string) string {
	asRegex := regexp.MustCompile(`(?i)\)\s*(?:AS\s+)?(\w+)\s*(?:,|$|\))`)

	idx := strings.Index(line, jsonAggExpr)
	if idx == -1 {
		return ""
	}

	remainder := line[idx+len(jsonAggExpr):]
	matches := asRegex.FindStringSubmatch(remainder)
	if len(matches) >= 2 {
		return matches[1]
	}

	return ""
}

func oidToTypeName(oid uint32) string {
	switch oid {
	case 16:
		return "boolean"
	case 17:
		return "bytea"
	case 18:
		return "char"
	case 19:
		return "name"
	case 20:
		return "bigint"
	case 21:
		return "smallint"
	case 23:
		return "integer"
	case 24:
		return "regproc"
	case 25:
		return "text"
	case 26:
		return "oid"
	case 114:
		return "json"
	case 142:
		return "xml"
	case 600:
		return "point"
	case 700:
		return "real"
	case 701:
		return "double precision"
	case 790:
		return "money"
	case 829:
		return "macaddr"
	case 869:
		return "inet"
	case 650:
		return "cidr"
	case 1000:
		return "boolean[]"
	case 1001:
		return "bytea[]"
	case 1005:
		return "smallint[]"
	case 1007:
		return "integer[]"
	case 1009:
		return "text[]"
	case 1014:
		return "character[]"
	case 1015:
		return "character varying[]"
	case 1016:
		return "bigint[]"
	case 1021:
		return "real[]"
	case 1022:
		return "double precision[]"
	case 1028:
		return "oid[]"
	case 1042:
		return "character"
	case 1043:
		return "character varying"
	case 1082:
		return "date"
	case 1083:
		return "time"
	case 1114:
		return "timestamp"
	case 1184:
		return "timestamp with time zone"
	case 1186:
		return "interval"
	case 1266:
		return "time with time zone"
	case 1560:
		return "bit"
	case 1562:
		return "bit varying"
	case 1700:
		return "numeric"
	case 2950:
		return "uuid"
	case 3802:
		return "jsonb"
	case 3807:
		return "jsonb[]"
	default:
		return "unknown"
	}
}

func pgTypeToGo(pgType string, nullable bool, schema *parser.Schema) (goType string, importPath string) {
	pgType = strings.ToLower(pgType)

	if strings.HasSuffix(pgType, "[]") {
		elemType := strings.TrimSuffix(pgType, "[]")
		elemGoType, imp := pgTypeToGo(elemType, false, schema)
		return "[]" + elemGoType, imp
	}

	if strings.HasPrefix(pgType, "character varying") {
		pgType = "character varying"
	}
	if strings.HasPrefix(pgType, "numeric") {
		pgType = "numeric"
	}

	var baseType string
	switch pgType {
	case "boolean", "bool":
		baseType = "bool"
	case "smallint", "int2":
		baseType = "int16"
	case "integer", "int", "int4":
		baseType = "int32"
	case "bigint", "int8":
		baseType = "int64"
	case "real", "float4":
		baseType = "float32"
	case "double precision", "float8":
		baseType = "float64"
	case "text", "character varying", "varchar", "character", "char", "name":
		baseType = "string"
	case "bytea":
		return "[]byte", ""
	case "uuid":
		baseType = "string"
	case "json", "jsonb":
		return "json.RawMessage", "encoding/json"
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz", "date", "time", "time with time zone":
		if nullable {
			return "*time.Time", "time"
		}
		return "time.Time", "time"
	case "interval":
		baseType = "string"
	case "numeric", "decimal", "money":
		baseType = "string"
	case "inet", "cidr", "macaddr":
		baseType = "string"
	case "oid":
		baseType = "uint32"
	case "unknown":
		baseType = "interface{}"
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
	if nullable && baseType != "interface{}" && !strings.HasPrefix(baseType, "[]") {
		return "*" + baseType, importPath
	}
	return baseType, importPath
}

func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		upper := strings.ToUpper(part)
		if isCommonInitialism(upper) {
			result.WriteString(upper)
		} else {
			result.WriteString(strings.ToUpper(string(part[0])))
			result.WriteString(strings.ToLower(part[1:]))
		}
	}
	return result.String()
}

func isCommonInitialism(s string) bool {
	initialisms := map[string]bool{
		"ID": true, "URL": true, "API": true, "HTTP": true, "HTTPS": true,
		"JSON": true, "XML": true, "UUID": true, "SQL": true, "SSH": true,
		"TCP": true, "UDP": true, "IP": true, "HTML": true, "CSS": true,
		"DNS": true, "RPC": true, "TLS": true, "SSL": true, "EOF": true,
		"ASCII": true, "CPU": true, "RAM": true, "OS": true,
	}
	return initialisms[s]
}
