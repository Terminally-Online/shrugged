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
			query.Parameters[i].Type = pgType
		}
	}

	jsonAggColumns := detectJSONAggColumns(query.SQL, schema)

	query.Columns = make([]parser.QueryColumn, len(sd.Fields))
	for i, field := range sd.Fields {
		pgType := resolveTypeName(field.DataTypeOID, typeMap)
		nullable := true

		if jsonAggInfo, ok := jsonAggColumns[field.Name]; ok {
			query.Columns[i] = parser.QueryColumn{
				Name:         field.Name,
				Type:         pgType,
				Nullable:     nullable,
				IsJSONAgg:    true,
				JSONElemType: jsonAggInfo.tableName,
			}
		} else {
			query.Columns[i] = parser.QueryColumn{
				Name:     field.Name,
				Type:     pgType,
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
			query.Parameters[i].Type = pgType
		}
	}

	return query, nil
}

func resolveTypeName(oid uint32, typeMap map[uint32]string) string {
	if name := oidToTypeName(oid); name != "unknown" {
		return name
	}
	if name, ok := typeMap[oid]; ok {
		// PostgreSQL represents array types with a "_" prefix in pg_type
		// (e.g., "_uuid" for uuid[], "_int4" for integer[]). Convert this
		// to the "basetype[]" format that pgTypeToGo expects.
		if strings.HasPrefix(name, "_") {
			return name[1:] + "[]"
		}
		return name
	}
	return "unknown"
}

type jsonAggInfo struct {
	tableName string
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

				tableExists := false
				for _, t := range schema.Tables {
					if t.Name == tableName {
						tableExists = true
						break
					}
				}

				columnName := extractColumnAlias(line, match[0])
				if columnName != "" && tableExists {
					result[columnName] = jsonAggInfo{
						tableName: tableName,
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
	case 1115:
		return "timestamp[]"
	case 1182:
		return "date[]"
	case 1185:
		return "timestamp with time zone[]"
	case 1231:
		return "numeric[]"
	case 1700:
		return "numeric"
	case 2950:
		return "uuid"
	case 2951:
		return "uuid[]"
	case 3802:
		return "jsonb"
	case 3807:
		return "jsonb[]"
	default:
		return "unknown"
	}
}
