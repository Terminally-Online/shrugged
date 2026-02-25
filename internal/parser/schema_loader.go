package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// schemaFile represents a single SQL file with its path, content, and
// extracted dependency metadata used for topological ordering.
type schemaFile struct {
	Path      string
	Content   string
	Defines   []string
	DependsOn []string
}

// LoadSchema loads schema SQL from a file or directory. When given a directory,
// it recursively collects all .sql files, builds a dependency graph from DDL
// statements, and concatenates them in topological order so that definitions
// precede their dependents.
func LoadSchema(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if !strings.HasSuffix(path, ".sql") {
			sqlPath := path + ".sql"
			if _, sqlErr := os.Stat(sqlPath); sqlErr == nil {
				return LoadFile(sqlPath)
			}
		}
		return "", fmt.Errorf("schema path not found: %s", path)
	}

	if !info.IsDir() {
		return LoadFile(path)
	}

	return loadSchemaDirectory(path)
}

func loadSchemaDirectory(dirPath string) (string, error) {
	var files []*schemaFile

	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".sql") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", path, err)
		}

		files = append(files, &schemaFile{
			Path:    path,
			Content: string(content),
		})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk schema directory %s: %w", dirPath, err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no .sql files found in %s", dirPath)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	extractDependencies(files)
	sorted := toposortSchemaFiles(files)

	var sb strings.Builder
	for i, sf := range sorted {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(sf.Content)
	}

	return sb.String(), nil
}

// stripSQLCommentsAndStrings removes single-line comments (-- ...), block
// comments (/* ... */), and string literals ('...') from SQL text. This
// prevents the dependency extractor from matching identifiers that appear
// only inside comments or string constants.
func stripSQLCommentsAndStrings(sql string) string {
	var sb strings.Builder
	i := 0
	for i < len(sql) {
		// Single-line comment
		if i+1 < len(sql) && sql[i] == '-' && sql[i+1] == '-' {
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			continue
		}
		// Block comment
		if i+1 < len(sql) && sql[i] == '/' && sql[i+1] == '*' {
			i += 2
			for i+1 < len(sql) {
				if sql[i] == '*' && sql[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}
		// String literal (handles '' escapes)
		if sql[i] == '\'' {
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					i++
					if i < len(sql) && sql[i] == '\'' {
						i++ // escaped quote
						continue
					}
					break
				}
				i++
			}
			continue
		}
		// Dollar-quoted string ($tag$...$tag$)
		if sql[i] == '$' {
			tagStart := i
			i++
			for i < len(sql) && (sql[i] == '_' || (sql[i] >= 'a' && sql[i] <= 'z') || (sql[i] >= 'A' && sql[i] <= 'Z') || (sql[i] >= '0' && sql[i] <= '9')) {
				i++
			}
			if i < len(sql) && sql[i] == '$' {
				tag := sql[tagStart : i+1]
				i++
				endIdx := strings.Index(sql[i:], tag)
				if endIdx >= 0 {
					i += endIdx + len(tag)
				}
				continue
			}
			// Not a dollar-quote, write the $ we consumed
			sb.WriteString(sql[tagStart:i])
			continue
		}
		sb.WriteByte(sql[i])
		i++
	}
	return sb.String()
}

// identifierPattern matches a potentially schema-qualified, optionally quoted SQL identifier.
// Capture groups: (1) quoted schema, (2) quoted name, (3) unquoted name (may include schema.name via dot)
const identifierPattern = `(?:"([^"]+)"\.)?(?:"([^"]+)"|(\w+(?:\.\w+)?))`

var (
	createRegex      = regexp.MustCompile(`(?im)CREATE\s+(?:OR\s+REPLACE\s+)?(TABLE|TYPE|EXTENSION|SCHEMA|SEQUENCE|VIEW|MATERIALIZED\s+VIEW|FUNCTION|PROCEDURE|DOMAIN|AGGREGATE|FOREIGN\s+TABLE)\s+(?:IF\s+NOT\s+EXISTS\s+)?` + identifierPattern)
	referencesRegex  = regexp.MustCompile(`(?im)REFERENCES\s+` + identifierPattern)
	partitionOfRegex = regexp.MustCompile(`(?im)PARTITION\s+OF\s+` + identifierPattern)
	inheritsRegex    = regexp.MustCompile(`(?im)INHERITS\s*\(\s*` + identifierPattern)
	indexOnRegex     = regexp.MustCompile(`(?im)CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?\S+\s+ON\s+(?:ONLY\s+)?` + identifierPattern)
	triggerOnRegex   = regexp.MustCompile(`(?im)ON\s+` + identifierPattern + `\s+(?:FOR|EXECUTE)`)
	policyOnRegex    = regexp.MustCompile(`(?im)CREATE\s+POLICY\s+\S+\s+ON\s+` + identifierPattern)
)

// resolveIdentifier extracts a lowercased, optionally schema-qualified name
// from the three capture groups produced by identifierPattern.
func resolveIdentifier(quotedSchema, quotedName, unquoted string) string {
	if quotedName != "" {
		name := strings.ToLower(quotedName)
		if quotedSchema != "" {
			schema := strings.ToLower(quotedSchema)
			if schema != "public" {
				return schema + "." + name
			}
		}
		return name
	}
	if unquoted == "" {
		return ""
	}
	lower := strings.ToLower(unquoted)
	if parts := strings.SplitN(lower, ".", 2); len(parts) == 2 {
		if parts[0] != "public" {
			return lower
		}
		return parts[1]
	}
	return lower
}

func extractDependencies(files []*schemaFile) {
	allDefined := make(map[string]bool)

	// First pass: extract what each file defines
	for _, sf := range files {
		stripped := stripSQLCommentsAndStrings(sf.Content)
		for _, m := range createRegex.FindAllStringSubmatch(stripped, -1) {
			// m[1]=object type, m[2]=quoted schema, m[3]=quoted name, m[4]=unquoted
			name := resolveIdentifier(m[2], m[3], m[4])
			if name != "" {
				sf.Defines = append(sf.Defines, name)
				allDefined[name] = true
			}
		}
	}

	// Pre-compile word boundary patterns for all defined names so we don't
	// compile inside the per-file loop.
	typePatterns := make(map[string]*regexp.Regexp, len(allDefined))
	for name := range allDefined {
		typePatterns[name] = regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(name) + `\b`)
	}

	// Second pass: extract what each file references
	for _, sf := range files {
		selfSet := make(map[string]bool, len(sf.Defines))
		for _, d := range sf.Defines {
			selfSet[d] = true
		}

		depSet := make(map[string]bool)
		addDep := func(name string) {
			if name != "" && allDefined[name] && !selfSet[name] {
				depSet[name] = true
			}
		}

		stripped := stripSQLCommentsAndStrings(sf.Content)

		// Structural dependency regexes — each uses identifierPattern so
		// capture groups are at consistent offsets within the match.
		structuralRegexes := []*regexp.Regexp{
			referencesRegex,
			partitionOfRegex,
			inheritsRegex,
			indexOnRegex,
			triggerOnRegex,
			policyOnRegex,
		}
		for _, re := range structuralRegexes {
			for _, m := range re.FindAllStringSubmatch(stripped, -1) {
				// The identifier groups are always the last 3 captures
				gi := len(m) - 3
				addDep(resolveIdentifier(m[gi], m[gi+1], m[gi+2]))
			}
		}

		// Type/name references: check if any defined name from another file
		// appears as a word-bounded identifier in the stripped SQL.
		strippedLower := strings.ToLower(stripped)
		for name := range allDefined {
			if selfSet[name] {
				continue
			}
			if typePatterns[name].MatchString(strippedLower) {
				depSet[name] = true
			}
		}

		for dep := range depSet {
			sf.DependsOn = append(sf.DependsOn, dep)
		}
		sort.Strings(sf.DependsOn)
	}
}

// toposortSchemaFiles performs a topological sort on schema files using Kahn's
// algorithm. Files whose definitions are depended upon by other files appear
// first. Ties are broken by path order. Cycles fall back to path ordering.
func toposortSchemaFiles(files []*schemaFile) []*schemaFile {
	nameToFileIdx := make(map[string]int)
	for i, sf := range files {
		for _, name := range sf.Defines {
			nameToFileIdx[name] = i
		}
	}

	// Build file-level adjacency: file i depends on file j
	fileDeps := make([]map[int]bool, len(files))
	for i, sf := range files {
		fileDeps[i] = make(map[int]bool)
		for _, dep := range sf.DependsOn {
			if j, ok := nameToFileIdx[dep]; ok && j != i {
				fileDeps[i][j] = true
			}
		}
	}

	// unresolved tracks how many of file i's dependencies haven't been
	// emitted yet. When it hits 0, the file is ready.
	unresolved := make([]int, len(files))
	for i := range files {
		unresolved[i] = len(fileDeps[i])
	}

	// Reverse adjacency: which files depend on file j?
	dependents := make(map[int][]int)
	for i, deps := range fileDeps {
		for j := range deps {
			dependents[j] = append(dependents[j], i)
		}
	}

	var queue []int
	for i := range files {
		if unresolved[i] == 0 {
			queue = append(queue, i)
		}
	}

	var sorted []*schemaFile
	visited := make([]bool, len(files))

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		if visited[idx] {
			continue
		}
		visited[idx] = true
		sorted = append(sorted, files[idx])

		var ready []int
		for _, dep := range dependents[idx] {
			if visited[dep] {
				continue
			}
			unresolved[dep]--
			if unresolved[dep] == 0 {
				ready = append(ready, dep)
			}
		}
		sort.Slice(ready, func(a, b int) bool {
			return files[ready[a]].Path < files[ready[b]].Path
		})
		queue = append(queue, ready...)
	}

	// Cycles: append remaining files in path order
	if len(sorted) < len(files) {
		for i := range files {
			if !visited[i] {
				sorted = append(sorted, files[i])
			}
		}
	}

	return sorted
}
