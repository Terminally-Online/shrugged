package parser

type QueryResultType string

const (
	QueryResultRow      QueryResultType = "row"
	QueryResultRows     QueryResultType = "rows"
	QueryResultExec     QueryResultType = "exec"
	QueryResultExecRows QueryResultType = "execrows"
)

type Query struct {
	Name         string
	SQL          string
	PreparedSQL  string
	ResultType   QueryResultType
	Parameters   []QueryParameter
	Columns      []QueryColumn
	NestMappings []NestMapping
	SourceFile   string
	LineNumber   int
}

type QueryParameter struct {
	Name     string
	Position int
	Type     string
	Nullable bool
}

type QueryColumn struct {
	Name         string
	Type         string
	Nullable     bool
	IsJSONAgg    bool
	JSONElemType string
}

type NestMapping struct {
	StructName string
	Prefix     string
	Columns    []string
}

type QueryFile struct {
	Path    string
	Queries []Query
}
