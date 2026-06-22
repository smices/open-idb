// SPDX-License-Identifier: MIT

package id

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGlobalIDsUseULID(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	forbidden := []string{
		"gen_random_uuid()",
		" UUID",
		"::uuid",
		"pgtype.UUID",
		"github.com/google/uuid",
		"github.com/google/ulid",
	}
	allowed := map[string]bool{
		filepath.Clean("internal/id/ulid_policy_test.go"): true,
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.Clean(rel)
		if allowed[rel] || !isPolicyCheckedFile(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, pattern := range forbidden {
			if strings.Contains(content, pattern) {
				t.Errorf("%s contains forbidden non-ULID identifier pattern %q", rel, pattern)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestULIDImplementationUsesOklogPackage(t *testing.T) {
	data, err := os.ReadFile("ulid.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `"github.com/oklog/ulid/v2"`) {
		t.Fatalf("internal ULID implementation must use github.com/oklog/ulid/v2")
	}
}

func TestDatabaseIDsUseChar26(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "migrations", "000001_schema_baseline.sql"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, column := range []string{
		"id",
		"entity_id",
		"user_id",
		"source_id",
		"application_id",
		"role_id",
		"permission_id",
		"resource_scope_id",
		"subject_id",
		"parent_id",
		"organization_id",
		"group_id",
		"actor_user_id",
	} {
		pattern := regexp.MustCompile(`(?m)^\s+` + regexp.QuoteMeta(column) + `\s+TEXT\b`)
		if pattern.MatchString(content) {
			t.Errorf("schema still uses loose TEXT for ULID column: %s", column)
		}
	}
	if !strings.Contains(content, "id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid()") {
		t.Errorf("schema must use CHAR(26) primary keys with idb_generate_ulid default")
	}
}

func isPolicyCheckedFile(path string) bool {
	if strings.HasPrefix(path, "docs/superpowers/plans/") {
		return false
	}
	if strings.HasPrefix(path, "go.sum") {
		return false
	}
	return strings.HasSuffix(path, ".go") ||
		strings.HasSuffix(path, ".sql") ||
		strings.HasSuffix(path, ".yaml") ||
		strings.HasSuffix(path, ".md")
}
