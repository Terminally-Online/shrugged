package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeStructFile_UpdatesFields(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "users.go")

	original := `package models

import "time"

type Users struct {
	ID    int64     ` + "`json:\"id\"`" + `
	Email string    ` + "`json:\"email\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	newFields := []StructField{
		{Name: "ID", Type: "int64", Tag: `json:"id"`},
		{Name: "Email", Type: "string", Tag: `json:"email"`},
		{Name: "Name", Type: "string", Tag: `json:"name"`},
		{Name: "CreatedAt", Type: "time.Time", Tag: `json:"created_at"`},
	}

	result, err := mergeStructFile(filePath, "Users", newFields, []string{"time"})
	if err != nil {
		t.Fatalf("mergeStructFile() error = %v", err)
	}

	content := string(result)

	if !strings.Contains(content, "Name") || !strings.Contains(content, `json:"name"`) {
		t.Error("merged file should contain new Name field with json tag")
	}
}

func TestMergeStructFile_PreservesMethods(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "users.go")

	original := `package models

type Users struct {
	ID    int64  ` + "`json:\"id\"`" + `
	Email string ` + "`json:\"email\"`" + `
}

func (u *Users) DisplayName() string {
	return u.Email
}

func (u *Users) IsValid() bool {
	return u.Email != ""
}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	newFields := []StructField{
		{Name: "ID", Type: "int64", Tag: `json:"id"`},
		{Name: "Email", Type: "string", Tag: `json:"email"`},
		{Name: "Name", Type: "string", Tag: `json:"name"`},
	}

	result, err := mergeStructFile(filePath, "Users", newFields, nil)
	if err != nil {
		t.Fatalf("mergeStructFile() error = %v", err)
	}

	content := string(result)

	if !strings.Contains(content, "func (u *Users) DisplayName()") {
		t.Error("merged file should preserve DisplayName method")
	}
	if !strings.Contains(content, "func (u *Users) IsValid()") {
		t.Error("merged file should preserve IsValid method")
	}
	if !strings.Contains(content, "Name") || !strings.Contains(content, `json:"name"`) {
		t.Error("merged file should contain new Name field with json tag")
	}
}

func TestMergeStructFile_PreservesImports(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "users.go")

	original := `package models

import (
	"strings"
	"time"
)

type Users struct {
	ID        int64     ` + "`json:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}

func (u *Users) LowerEmail() string {
	return strings.ToLower(u.Email)
}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	newFields := []StructField{
		{Name: "ID", Type: "int64", Tag: `json:"id"`},
		{Name: "Email", Type: "string", Tag: `json:"email"`},
		{Name: "CreatedAt", Type: "time.Time", Tag: `json:"created_at"`},
	}

	result, err := mergeStructFile(filePath, "Users", newFields, []string{"time"})
	if err != nil {
		t.Fatalf("mergeStructFile() error = %v", err)
	}

	content := string(result)

	if !strings.Contains(content, `"strings"`) {
		t.Error("merged file should preserve strings import")
	}
	if !strings.Contains(content, `"time"`) {
		t.Error("merged file should preserve time import")
	}
}

func TestMergeStructFile_AddsNewImports(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "users.go")

	original := `package models

type Users struct {
	ID   int64  ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	newFields := []StructField{
		{Name: "ID", Type: "int64", Tag: `json:"id"`},
		{Name: "Name", Type: "string", Tag: `json:"name"`},
		{Name: "CreatedAt", Type: "time.Time", Tag: `json:"created_at"`},
	}

	result, err := mergeStructFile(filePath, "Users", newFields, []string{"time"})
	if err != nil {
		t.Fatalf("mergeStructFile() error = %v", err)
	}

	content := string(result)

	if !strings.Contains(content, `"time"`) {
		t.Error("merged file should add time import")
	}
}

func TestMergeStructFile_PreservesOtherDeclarations(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "users.go")

	original := `package models

type Users struct {
	ID int64 ` + "`json:\"id\"`" + `
}

var DefaultUser = Users{ID: 1}

const MaxUsers = 100
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	newFields := []StructField{
		{Name: "ID", Type: "int64", Tag: `json:"id"`},
		{Name: "Email", Type: "string", Tag: `json:"email"`},
	}

	result, err := mergeStructFile(filePath, "Users", newFields, nil)
	if err != nil {
		t.Fatalf("mergeStructFile() error = %v", err)
	}

	content := string(result)

	if !strings.Contains(content, "var DefaultUser") {
		t.Error("merged file should preserve var declaration")
	}
	if !strings.Contains(content, "const MaxUsers") {
		t.Error("merged file should preserve const declaration")
	}
}

func TestMergeEnumFile_UpdatesValues(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "status.go")

	original := `package models

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	newValues := []EnumValue{
		{Name: "StatusActive", Value: "active"},
		{Name: "StatusInactive", Value: "inactive"},
		{Name: "StatusPending", Value: "pending"},
	}

	result, err := mergeEnumFile(filePath, "Status", newValues)
	if err != nil {
		t.Fatalf("mergeEnumFile() error = %v", err)
	}

	content := string(result)

	if !strings.Contains(content, "StatusPending") {
		t.Error("merged file should contain new StatusPending value")
	}
	if !strings.Contains(content, `"pending"`) {
		t.Error("merged file should contain pending value string")
	}
}

func TestMergeEnumFile_PreservesMethods(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "status.go")

	original := `package models

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

func (s Status) IsActive() bool {
	return s == StatusActive
}

func (s Status) String() string {
	return string(s)
}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	newValues := []EnumValue{
		{Name: "StatusActive", Value: "active"},
		{Name: "StatusInactive", Value: "inactive"},
		{Name: "StatusPending", Value: "pending"},
	}

	result, err := mergeEnumFile(filePath, "Status", newValues)
	if err != nil {
		t.Fatalf("mergeEnumFile() error = %v", err)
	}

	content := string(result)

	if !strings.Contains(content, "func (s Status) IsActive()") {
		t.Error("merged file should preserve IsActive method")
	}
	if !strings.Contains(content, "func (s Status) String()") {
		t.Error("merged file should preserve String method")
	}
	if !strings.Contains(content, "StatusPending") {
		t.Error("merged file should contain new StatusPending value")
	}
}

func TestMergeEnumFile_PreservesVars(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "status.go")

	original := `package models

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

var AllStatuses = []Status{StatusActive, StatusInactive}
`
	if err := os.WriteFile(filePath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	newValues := []EnumValue{
		{Name: "StatusActive", Value: "active"},
		{Name: "StatusInactive", Value: "inactive"},
		{Name: "StatusPending", Value: "pending"},
	}

	result, err := mergeEnumFile(filePath, "Status", newValues)
	if err != nil {
		t.Fatalf("mergeEnumFile() error = %v", err)
	}

	content := string(result)

	if !strings.Contains(content, "var AllStatuses") {
		t.Error("merged file should preserve AllStatuses var")
	}
}

func TestBuildFieldList(t *testing.T) {
	fields := []StructField{
		{Name: "ID", Type: "int64", Tag: `json:"id"`},
		{Name: "Name", Type: "string", Tag: `json:"name"`},
		{Name: "Bio", Type: "*string", Tag: ""},
	}

	result := buildFieldList(fields)

	if len(result.List) != 3 {
		t.Errorf("buildFieldList() returned %d fields, want 3", len(result.List))
	}

	if result.List[0].Names[0].Name != "ID" {
		t.Error("first field should be ID")
	}
	if result.List[2].Tag != nil {
		t.Error("Bio field should not have a tag")
	}
}

func TestParseTypeExpr(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"int64"},
		{"string"},
		{"*string"},
		{"[]string"},
		{"time.Time"},
		{"*time.Time"},
		{"json.RawMessage"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseTypeExpr(tt.input)
			if result == nil {
				t.Errorf("parseTypeExpr(%q) returned nil", tt.input)
			}
		})
	}
}

func TestBuildEnumSpecs(t *testing.T) {
	values := []EnumValue{
		{Name: "StatusActive", Value: "active"},
		{Name: "StatusPending", Value: "pending"},
	}

	specs := buildEnumSpecs("Status", values)

	if len(specs) != 2 {
		t.Errorf("buildEnumSpecs() returned %d specs, want 2", len(specs))
	}
}
