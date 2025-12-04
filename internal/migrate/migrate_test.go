package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"shrugged/internal/docker"
)

func TestComputeChecksum(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"simple", "CREATE TABLE users (id INT);"},
		{"multiline", "CREATE TABLE users (\n\tid INT,\n\tname TEXT\n);"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum := ComputeChecksum(tt.content)
			if checksum == "" {
				t.Error("checksum should not be empty")
			}
			if len(checksum) != 64 {
				t.Errorf("checksum length = %d, want 64 (SHA256 hex)", len(checksum))
			}
		})
	}
}

func TestComputeChecksum_Deterministic(t *testing.T) {
	content := "CREATE TABLE users (id INT);"

	checksum1 := ComputeChecksum(content)
	checksum2 := ComputeChecksum(content)

	if checksum1 != checksum2 {
		t.Errorf("checksums should be equal: %q != %q", checksum1, checksum2)
	}
}

func TestComputeChecksum_DifferentContent(t *testing.T) {
	checksum1 := ComputeChecksum("CREATE TABLE users (id INT);")
	checksum2 := ComputeChecksum("CREATE TABLE posts (id INT);")

	if checksum1 == checksum2 {
		t.Error("different content should produce different checksums")
	}
}

func TestEnsureMigrationsTable_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	applied, err := GetApplied(ctx, container.ConnectionString())
	if err != nil {
		t.Fatalf("GetApplied() error = %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("expected 0 applied migrations, got %d", len(applied))
	}
}

func TestApplyAndGetApplied_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	migration := Migration{
		Name:    "001_create_users.sql",
		Content: "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT);",
	}

	if err := Apply(ctx, dbURL, migration); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	applied, err := GetApplied(ctx, dbURL)
	if err != nil {
		t.Fatalf("GetApplied() error = %v", err)
	}

	if len(applied) != 1 {
		t.Fatalf("expected 1 applied migration, got %d", len(applied))
	}

	if applied[0].Name != migration.Name {
		t.Errorf("migration name = %q, want %q", applied[0].Name, migration.Name)
	}

	if applied[0].Checksum == "" {
		t.Error("migration checksum should not be empty")
	}
}

func TestGetPending_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	tmpDir, err := os.MkdirTemp("", "migrate_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	migrations := []struct {
		name    string
		content string
	}{
		{"001_create_users.sql", "CREATE TABLE users (id SERIAL PRIMARY KEY);"},
		{"002_create_posts.sql", "CREATE TABLE posts (id SERIAL PRIMARY KEY);"},
		{"002_create_posts.down.sql", "DROP TABLE posts;"},
	}

	for _, m := range migrations {
		if err := os.WriteFile(filepath.Join(tmpDir, m.name), []byte(m.content), 0644); err != nil {
			t.Fatalf("failed to write migration file: %v", err)
		}
	}

	pending, err := GetPending(ctx, dbURL, tmpDir)
	if err != nil {
		t.Fatalf("GetPending() error = %v", err)
	}

	if len(pending) != 2 {
		t.Fatalf("expected 2 pending migrations, got %d", len(pending))
	}

	if pending[0].Name != "001_create_users.sql" {
		t.Errorf("first pending = %q, want %q", pending[0].Name, "001_create_users.sql")
	}
	if pending[1].Name != "002_create_posts.sql" {
		t.Errorf("second pending = %q, want %q", pending[1].Name, "002_create_posts.sql")
	}

	if err := Apply(ctx, dbURL, pending[0]); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	pending, err = GetPending(ctx, dbURL, tmpDir)
	if err != nil {
		t.Fatalf("GetPending() after apply error = %v", err)
	}

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending migration after apply, got %d", len(pending))
	}
}

func TestGetPending_SkipsDownMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	tmpDir, err := os.MkdirTemp("", "migrate_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	files := []struct {
		name    string
		content string
	}{
		{"001_test.sql", "CREATE TABLE test (id INT);"},
		{"001_test.down.sql", "DROP TABLE test;"},
	}

	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f.name), []byte(f.content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	pending, err := GetPending(ctx, dbURL, tmpDir)
	if err != nil {
		t.Fatalf("GetPending() error = %v", err)
	}

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending migration (excluding .down.sql), got %d", len(pending))
	}

	if pending[0].Name != "001_test.sql" {
		t.Errorf("pending name = %q, want %q", pending[0].Name, "001_test.sql")
	}
}

func TestGetLastApplied_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	last, err := GetLastApplied(ctx, dbURL)
	if err != nil {
		t.Fatalf("GetLastApplied() error = %v", err)
	}
	if last != nil {
		t.Error("expected nil for empty migrations")
	}

	migrations := []Migration{
		{Name: "001_first.sql", Content: "CREATE TABLE first (id INT);"},
		{Name: "002_second.sql", Content: "CREATE TABLE second (id INT);"},
	}

	for _, m := range migrations {
		if err := Apply(ctx, dbURL, m); err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
	}

	last, err = GetLastApplied(ctx, dbURL)
	if err != nil {
		t.Fatalf("GetLastApplied() error = %v", err)
	}

	if last == nil {
		t.Fatal("expected non-nil last migration")
	}

	if last.Name != "002_second.sql" {
		t.Errorf("last migration = %q, want %q", last.Name, "002_second.sql")
	}
}

func TestRollback_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	migration := Migration{
		Name:    "001_create_rollback_test.sql",
		Content: "CREATE TABLE rollback_test (id SERIAL PRIMARY KEY);",
	}

	if err := Apply(ctx, dbURL, migration); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	applied, err := GetApplied(ctx, dbURL)
	if err != nil {
		t.Fatalf("GetApplied() error = %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected 1 applied migration, got %d", len(applied))
	}

	rollbackMigration := Migration{
		Name:    migration.Name,
		Content: "DROP TABLE rollback_test;",
	}

	if err := Rollback(ctx, dbURL, rollbackMigration); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	applied, err = GetApplied(ctx, dbURL)
	if err != nil {
		t.Fatalf("GetApplied() after rollback error = %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("expected 0 applied migrations after rollback, got %d", len(applied))
	}
}

func TestGetRollbackable_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	tmpDir, err := os.MkdirTemp("", "migrate_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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

	migrations := []Migration{
		{Name: "001_first.sql", Content: "CREATE TABLE first (id INT);"},
		{Name: "002_second.sql", Content: "CREATE TABLE second (id INT);"},
	}

	for _, m := range migrations {
		if err := Apply(ctx, dbURL, m); err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
	}

	rollbackable, err := GetRollbackable(ctx, dbURL, tmpDir, 1)
	if err != nil {
		t.Fatalf("GetRollbackable() error = %v", err)
	}

	if len(rollbackable) != 1 {
		t.Fatalf("expected 1 rollbackable, got %d", len(rollbackable))
	}

	if rollbackable[0].Name != "002_second.sql" {
		t.Errorf("rollbackable name = %q, want %q", rollbackable[0].Name, "002_second.sql")
	}

	if rollbackable[0].Content != "DROP TABLE second;" {
		t.Errorf("rollbackable content = %q, want down migration content", rollbackable[0].Content)
	}

	rollbackable, err = GetRollbackable(ctx, dbURL, tmpDir, 2)
	if err != nil {
		t.Fatalf("GetRollbackable(2) error = %v", err)
	}

	if len(rollbackable) != 2 {
		t.Fatalf("expected 2 rollbackable, got %d", len(rollbackable))
	}

	if rollbackable[0].Name != "002_second.sql" {
		t.Errorf("first rollbackable = %q, want %q", rollbackable[0].Name, "002_second.sql")
	}
	if rollbackable[1].Name != "001_first.sql" {
		t.Errorf("second rollbackable = %q, want %q", rollbackable[1].Name, "001_first.sql")
	}
}

func TestGetRollbackable_MissingDownMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	tmpDir, err := os.MkdirTemp("", "migrate_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "001_test.sql"), []byte("CREATE TABLE test (id INT);"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	migration := Migration{
		Name:    "001_test.sql",
		Content: "CREATE TABLE test (id INT);",
	}

	if err := Apply(ctx, dbURL, migration); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	_, err = GetRollbackable(ctx, dbURL, tmpDir, 1)
	if err == nil {
		t.Error("expected error for missing down migration")
	}
}

func TestHasModifiedMigrations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := docker.DefaultPostgresConfig()
	container, err := docker.StartPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("StartPostgres() error = %v", err)
	}
	defer func() { _ = docker.StopContainer(context.Background(), container.ID) }()

	dbURL := container.ConnectionString()

	tmpDir, err := os.MkdirTemp("", "migrate_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalContent := "CREATE TABLE users (id INT);"
	if err := os.WriteFile(filepath.Join(tmpDir, "001_users.sql"), []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	migration := Migration{
		Name:    "001_users.sql",
		Content: originalContent,
	}

	if err := Apply(ctx, dbURL, migration); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	modified, err := HasModifiedMigrations(ctx, dbURL, tmpDir)
	if err != nil {
		t.Fatalf("HasModifiedMigrations() error = %v", err)
	}
	if len(modified) != 0 {
		t.Errorf("expected 0 modified migrations, got %d", len(modified))
	}

	modifiedContent := "CREATE TABLE users (id INT, name TEXT);"
	if err := os.WriteFile(filepath.Join(tmpDir, "001_users.sql"), []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to write modified file: %v", err)
	}

	modified, err = HasModifiedMigrations(ctx, dbURL, tmpDir)
	if err != nil {
		t.Fatalf("HasModifiedMigrations() after modification error = %v", err)
	}
	if len(modified) != 1 {
		t.Errorf("expected 1 modified migration, got %d", len(modified))
	}
}
