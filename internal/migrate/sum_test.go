package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateSum(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	files := []struct {
		name    string
		content string
	}{
		{"001_first.sql", "CREATE TABLE first (id INT);"},
		{"001_first.down.sql", "DROP TABLE first;"},
		{"002_second.sql", "CREATE TABLE second (id INT);"},
		{"002_second.down.sql", "DROP TABLE second;"},
	}

	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f.name), []byte(f.content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	entries, err := GenerateSum(tmpDir)
	if err != nil {
		t.Fatalf("GenerateSum() error = %v", err)
	}

	if len(entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.Hash == "" {
			t.Errorf("entry %s has empty hash", e.Name)
		}
	}
}

func TestGenerateSum_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	entries, err := GenerateSum(tmpDir)
	if err != nil {
		t.Fatalf("GenerateSum() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty dir, got %d", len(entries))
	}
}

func TestGenerateSum_NonExistentDir(t *testing.T) {
	entries, err := GenerateSum("/nonexistent/path")
	if err != nil {
		t.Fatalf("GenerateSum() error = %v", err)
	}

	if entries != nil {
		t.Errorf("expected nil entries for non-existent dir, got %v", entries)
	}
}

func TestWriteAndReadSum(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	entries := []SumEntry{
		{Name: "001_first.sql", Hash: "abc123"},
		{Name: "002_second.sql", Hash: "def456"},
	}

	if err := WriteSum(tmpDir, entries); err != nil {
		t.Fatalf("WriteSum() error = %v", err)
	}

	sumPath := filepath.Join(tmpDir, SumFile)
	if _, err := os.Stat(sumPath); os.IsNotExist(err) {
		t.Fatal("sum file was not created")
	}

	totalHash, readEntries, err := ReadSum(tmpDir)
	if err != nil {
		t.Fatalf("ReadSum() error = %v", err)
	}

	if totalHash == "" {
		t.Error("total hash should not be empty")
	}

	if len(readEntries) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(readEntries))
	}

	for i, e := range readEntries {
		if e.Name != entries[i].Name {
			t.Errorf("entry %d name = %q, want %q", i, e.Name, entries[i].Name)
		}
		if e.Hash != entries[i].Hash {
			t.Errorf("entry %d hash = %q, want %q", i, e.Hash, entries[i].Hash)
		}
	}
}

func TestReadSum_NonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	totalHash, entries, err := ReadSum(tmpDir)
	if err != nil {
		t.Fatalf("ReadSum() error = %v", err)
	}

	if totalHash != "" {
		t.Errorf("expected empty total hash, got %q", totalHash)
	}

	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}
}

func TestValidateSum_Valid(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	files := []struct {
		name    string
		content string
	}{
		{"001_first.sql", "CREATE TABLE first (id INT);"},
		{"002_second.sql", "CREATE TABLE second (id INT);"},
	}

	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f.name), []byte(f.content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() error = %v", err)
	}

	if err := ValidateSum(tmpDir); err != nil {
		t.Errorf("ValidateSum() error = %v, want nil", err)
	}
}

func TestValidateSum_ModifiedFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "001_first.sql"), []byte("CREATE TABLE first (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "001_first.sql"), []byte("CREATE TABLE first (id INT, name TEXT);"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	err = ValidateSum(tmpDir)
	if err == nil {
		t.Error("ValidateSum() should return error for modified file")
	}
}

func TestValidateSum_NoSumFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "001_first.sql"), []byte("CREATE TABLE first (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := ValidateSum(tmpDir); err != nil {
		t.Errorf("ValidateSum() should not error when no sum file exists, got: %v", err)
	}
}

func TestUpdateSum(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "001_first.sql"), []byte("CREATE TABLE first (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() error = %v", err)
	}

	sumPath := filepath.Join(tmpDir, SumFile)
	if _, err := os.Stat(sumPath); os.IsNotExist(err) {
		t.Error("sum file was not created")
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "002_second.sql"), []byte("CREATE TABLE second (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write second file: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() second call error = %v", err)
	}

	_, entries, err := ReadSum(tmpDir)
	if err != nil {
		t.Fatalf("ReadSum() error = %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries after update, got %d", len(entries))
	}
}

func TestUpdateSum_ReconcilesStaleSumFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "001_first.sql"), []byte("CREATE TABLE first (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "002_second.sql"), []byte("CREATE TABLE second (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() error = %v", err)
	}

	_, entriesBefore, err := ReadSum(tmpDir)
	if err != nil {
		t.Fatalf("ReadSum() error = %v", err)
	}
	if len(entriesBefore) != 2 {
		t.Fatalf("expected 2 entries before, got %d", len(entriesBefore))
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "003_third.sql"), []byte("CREATE TABLE third (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "003_third.down.sql"), []byte("DROP TABLE third;"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := ValidateSum(tmpDir); err != nil {
		t.Errorf("ValidateSum() should not error when new files are added (only detects modifications), got: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() after adding files error = %v", err)
	}

	totalHash, entriesAfter, err := ReadSum(tmpDir)
	if err != nil {
		t.Fatalf("ReadSum() error = %v", err)
	}

	if len(entriesAfter) != 4 {
		t.Errorf("expected 4 entries after reconciliation, got %d", len(entriesAfter))
	}

	if totalHash == "" {
		t.Error("total hash should not be empty after reconciliation")
	}

	if err := ValidateSum(tmpDir); err != nil {
		t.Errorf("ValidateSum() should pass after reconciliation, got: %v", err)
	}
}

func TestUpdateSum_OverwritesCorruptSumFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "001_first.sql"), []byte("CREATE TABLE first (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, SumFile), []byte("garbage\nmore garbage\n"), 0644); err != nil {
		t.Fatalf("failed to write corrupt sum file: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() should overwrite corrupt sum file, got: %v", err)
	}

	totalHash, entries, err := ReadSum(tmpDir)
	if err != nil {
		t.Fatalf("ReadSum() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	if totalHash == "" {
		t.Error("total hash should not be empty")
	}

	if err := ValidateSum(tmpDir); err != nil {
		t.Errorf("ValidateSum() should pass after overwrite, got: %v", err)
	}
}

func TestUpdateSum_Idempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "001_first.sql"), []byte("CREATE TABLE first (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("first UpdateSum() error = %v", err)
	}

	firstContent, err := os.ReadFile(filepath.Join(tmpDir, SumFile))
	if err != nil {
		t.Fatalf("failed to read sum file: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("second UpdateSum() error = %v", err)
	}

	secondContent, err := os.ReadFile(filepath.Join(tmpDir, SumFile))
	if err != nil {
		t.Fatalf("failed to read sum file: %v", err)
	}

	if string(firstContent) != string(secondContent) {
		t.Error("UpdateSum() should be idempotent, but sum file changed between runs")
	}
}

func TestValidateSum_DeletedFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	for _, f := range []struct{ name, content string }{
		{"001_first.sql", "CREATE TABLE first (id INT);"},
		{"002_second.sql", "CREATE TABLE second (id INT);"},
	} {
		if err := os.WriteFile(filepath.Join(tmpDir, f.name), []byte(f.content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() error = %v", err)
	}

	if err := os.Remove(filepath.Join(tmpDir, "002_second.sql")); err != nil {
		t.Fatalf("failed to delete migration file: %v", err)
	}

	err = ValidateSum(tmpDir)
	if err == nil {
		t.Error("ValidateSum() should return error when a tracked migration file is deleted")
	}
}

func TestUpdateSum_ClearsSumFileWhenMigrationsDeleted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "001_first.sql"), []byte("CREATE TABLE first (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() error = %v", err)
	}

	sumPath := filepath.Join(tmpDir, SumFile)
	if _, err := os.Stat(sumPath); os.IsNotExist(err) {
		t.Fatal("sum file should exist after UpdateSum with migrations")
	}

	if err := os.Remove(filepath.Join(tmpDir, "001_first.sql")); err != nil {
		t.Fatalf("failed to delete migration: %v", err)
	}

	if err := UpdateSum(tmpDir); err != nil {
		t.Fatalf("UpdateSum() with empty dir error = %v", err)
	}

	if _, err := os.Stat(sumPath); !os.IsNotExist(err) {
		t.Error("sum file should be removed when all migrations are deleted")
	}

	if err := ValidateSum(tmpDir); err != nil {
		t.Errorf("ValidateSum() should pass on empty dir with no sum file, got: %v", err)
	}
}

func TestGenerateSum_SortsFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	files := []string{"003_third.sql", "001_first.sql", "002_second.sql"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("SELECT 1;"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	entries, err := GenerateSum(tmpDir)
	if err != nil {
		t.Fatalf("GenerateSum() error = %v", err)
	}

	expected := []string{"001_first.sql", "002_second.sql", "003_third.sql"}
	for i, e := range entries {
		if e.Name != expected[i] {
			t.Errorf("entry %d = %q, want %q", i, e.Name, expected[i])
		}
	}
}
