// SPDX-License-Identifier: MIT

package id

import (
	"os"
	"strings"
	"testing"
)

func TestBaselineSchemaUsesBusinessEntityBoundary(t *testing.T) {
	schema, err := os.ReadFile("../../migrations/000001_schema_baseline.sql")
	if err != nil {
		t.Fatalf("read baseline schema: %v", err)
	}
	text := string(schema)
	if !strings.Contains(text, "CREATE TABLE business_entities") {
		t.Fatal("baseline schema must define business_entities")
	}
	if !strings.Contains(text, "entity_id CHAR(26)") {
		t.Fatal("baseline schema must use entity_id CHAR(26) for scoped tables")
	}
	if strings.Contains(text, "CREATE TABLE tenants") {
		t.Fatal("baseline schema must not define SaaS-style tenants")
	}
	if strings.Contains(text, "tenant_id") {
		t.Fatal("baseline schema must not use tenant_id")
	}
}
