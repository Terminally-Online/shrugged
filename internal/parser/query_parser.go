package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	nameAnnotationRegex = regexp.MustCompile(`^--\s*name:\s*(\w+)\s*:(row|rows|exec|execrows)\s*$`)
	nestAnnotationRegex = regexp.MustCompile(`^--\s*nest:\s*(.+)$`)
	nestMappingRegex    = regexp.MustCompile(`(\w+)\(([^)]+)\)`)
	paramRegex          = regexp.MustCompile(`@(\w+)`)
	jsonAggRegex        = regexp.MustCompile(`(?i)(json_agg|jsonb_agg)\s*\(`)
)

func ParseQueryFile(path string) (*QueryFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read query file %s: %w", path, err)
	}

	queries, err := parseQueryContent(string(content), path)
	if err != nil {
		return nil, err
	}

	return &QueryFile{
		Path:    path,
		Queries: queries,
	}, nil
}

func ParseQueryDirectory(dirPath string) ([]*QueryFile, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read query directory %s: %w", dirPath, err)
	}

	var files []*QueryFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		qf, err := ParseQueryFile(filePath)
		if err != nil {
			return nil, err
		}
		if len(qf.Queries) > 0 {
			files = append(files, qf)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func ParseQueries(path string) ([]*QueryFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		if !strings.HasSuffix(path, ".sql") {
			sqlPath := path + ".sql"
			if _, err := os.Stat(sqlPath); err == nil {
				qf, err := ParseQueryFile(sqlPath)
				if err != nil {
					return nil, err
				}
				return []*QueryFile{qf}, nil
			}
		}
		return nil, fmt.Errorf("query path not found: %s", path)
	}

	if info.IsDir() {
		return ParseQueryDirectory(path)
	}

	qf, err := ParseQueryFile(path)
	if err != nil {
		return nil, err
	}
	return []*QueryFile{qf}, nil
}

func parseQueryContent(content string, sourcePath string) ([]Query, error) {
	var queries []Query
	var currentQuery *Query
	var sqlBuilder strings.Builder
	var nestMappings []NestMapping

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if matches := nameAnnotationRegex.FindStringSubmatch(trimmed); matches != nil {
			if currentQuery != nil {
				currentQuery.SQL = strings.TrimSpace(sqlBuilder.String())
				currentQuery.NestMappings = nestMappings
				if currentQuery.SQL != "" {
					preparedSQL, params := extractParameters(currentQuery.SQL)
					currentQuery.PreparedSQL = preparedSQL
					currentQuery.Parameters = params
					queries = append(queries, *currentQuery)
				}
			}

			currentQuery = &Query{
				Name:       matches[1],
				ResultType: QueryResultType(matches[2]),
				SourceFile: sourcePath,
				LineNumber: lineNum,
			}
			sqlBuilder.Reset()
			nestMappings = nil
			continue
		}

		if matches := nestAnnotationRegex.FindStringSubmatch(trimmed); matches != nil {
			mappings := parseNestMappings(matches[1])
			nestMappings = append(nestMappings, mappings...)
			continue
		}

		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		if currentQuery != nil {
			if sqlBuilder.Len() > 0 {
				sqlBuilder.WriteString("\n")
			}
			sqlBuilder.WriteString(line)
		}
	}

	if currentQuery != nil {
		currentQuery.SQL = strings.TrimSpace(sqlBuilder.String())
		currentQuery.NestMappings = nestMappings
		if currentQuery.SQL != "" {
			preparedSQL, params := extractParameters(currentQuery.SQL)
			currentQuery.PreparedSQL = preparedSQL
			currentQuery.Parameters = params
			queries = append(queries, *currentQuery)
		}
	}

	return queries, nil
}

func parseNestMappings(annotation string) []NestMapping {
	var mappings []NestMapping

	matches := nestMappingRegex.FindAllStringSubmatch(annotation, -1)
	for _, match := range matches {
		structName := match[1]
		columnsSpec := strings.TrimSpace(match[2])

		var columns []string
		var prefix string

		if strings.HasSuffix(columnsSpec, ".*") {
			prefix = strings.TrimSuffix(columnsSpec, ".*")
		} else {
			parts := strings.Split(columnsSpec, ",")
			for _, p := range parts {
				columns = append(columns, strings.TrimSpace(p))
			}
		}

		mappings = append(mappings, NestMapping{
			StructName: structName,
			Prefix:     prefix,
			Columns:    columns,
		})
	}

	return mappings
}

func extractParameters(sql string) (string, []QueryParameter) {
	var params []QueryParameter
	paramPositions := make(map[string]int)
	position := 0

	result := paramRegex.ReplaceAllStringFunc(sql, func(match string) string {
		paramName := match[1:]

		if pos, exists := paramPositions[paramName]; exists {
			return fmt.Sprintf("$%d", pos)
		}

		position++
		paramPositions[paramName] = position
		params = append(params, QueryParameter{
			Name:     paramName,
			Position: position,
		})
		return fmt.Sprintf("$%d", position)
	})

	markOptionalParameters(sql, params)

	return result, params
}

var isNullParamRegex = regexp.MustCompile(`@(\w+)\s+IS\s+NULL`)

func markOptionalParameters(sql string, params []QueryParameter) {
	matches := isNullParamRegex.FindAllStringSubmatch(sql, -1)
	nullableParams := make(map[string]bool)
	for _, m := range matches {
		nullableParams[m[1]] = true
	}

	for i := range params {
		if nullableParams[params[i].Name] {
			params[i].Nullable = true
		}
	}
}

func DetectJSONAggregation(sql string) bool {
	return jsonAggRegex.MatchString(sql)
}

func GetAllQueries(files []*QueryFile) []Query {
	var all []Query
	for _, f := range files {
		all = append(all, f.Queries...)
	}
	return all
}
