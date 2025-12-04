package migrate

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SumFile = "shrugged.sum"

type SumEntry struct {
	Name string
	Hash string
}

func GenerateSum(migrationsDir string) ([]SumEntry, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		sqlFiles = append(sqlFiles, name)
	}

	sort.Strings(sqlFiles)

	var sumEntries []SumEntry
	for _, name := range sqlFiles {
		content, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", name, err)
		}

		hash := hashContent(content)
		sumEntries = append(sumEntries, SumEntry{Name: name, Hash: hash})
	}

	return sumEntries, nil
}

func WriteSum(migrationsDir string, entries []SumEntry) error {
	if len(entries) == 0 {
		return nil
	}

	sumPath := filepath.Join(migrationsDir, SumFile)
	f, err := os.Create(sumPath)
	if err != nil {
		return fmt.Errorf("failed to create sum file: %w", err)
	}
	defer f.Close()

	var allHashes []byte
	for _, entry := range entries {
		allHashes = append(allHashes, []byte(entry.Hash)...)
	}
	totalHash := hashContent(allHashes)

	if _, err := fmt.Fprintf(f, "h1:%s\n", totalHash); err != nil {
		return err
	}

	for _, entry := range entries {
		if _, err := fmt.Fprintf(f, "%s h1:%s\n", entry.Name, entry.Hash); err != nil {
			return err
		}
	}

	return nil
}

func ReadSum(migrationsDir string) (string, []SumEntry, error) {
	sumPath := filepath.Join(migrationsDir, SumFile)
	f, err := os.Open(sumPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("failed to open sum file: %w", err)
	}
	defer f.Close()

	var totalHash string
	var entries []SumEntry

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineNum++

		if lineNum == 1 {
			if !strings.HasPrefix(line, "h1:") {
				return "", nil, fmt.Errorf("invalid sum file: first line must be total hash")
			}
			totalHash = strings.TrimPrefix(line, "h1:")
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid sum file line %d: %s", lineNum, line)
		}

		name := parts[0]
		hash := strings.TrimPrefix(parts[1], "h1:")
		entries = append(entries, SumEntry{Name: name, Hash: hash})
	}

	if err := scanner.Err(); err != nil {
		return "", nil, fmt.Errorf("failed to read sum file: %w", err)
	}

	return totalHash, entries, nil
}

func ValidateSum(migrationsDir string) error {
	storedTotal, storedEntries, err := ReadSum(migrationsDir)
	if err != nil {
		return err
	}

	if storedTotal == "" && len(storedEntries) == 0 {
		return nil
	}

	currentEntries, err := GenerateSum(migrationsDir)
	if err != nil {
		return err
	}

	storedMap := make(map[string]string)
	for _, e := range storedEntries {
		storedMap[e.Name] = e.Hash
	}

	for _, current := range currentEntries {
		stored, exists := storedMap[current.Name]
		if !exists {
			if !strings.HasSuffix(current.Name, ".down.sql") {
				upName := current.Name
				if _, hasUp := storedMap[upName]; !hasUp {
					continue
				}
			}
			continue
		}

		if stored != current.Hash {
			return fmt.Errorf("migration %s has been modified (hash mismatch)", current.Name)
		}
	}

	var allHashes []byte
	for _, entry := range storedEntries {
		allHashes = append(allHashes, []byte(entry.Hash)...)
	}
	expectedTotal := hashContent(allHashes)

	if storedTotal != expectedTotal {
		return fmt.Errorf("sum file has been tampered with (total hash mismatch)")
	}

	return nil
}

func hashContent(content []byte) string {
	h := sha256.Sum256(content)
	return base64.StdEncoding.EncodeToString(h[:])
}

func UpdateSum(migrationsDir string) error {
	entries, err := GenerateSum(migrationsDir)
	if err != nil {
		return err
	}
	return WriteSum(migrationsDir, entries)
}
