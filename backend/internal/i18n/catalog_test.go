// SPDX-License-Identifier: MIT

package i18n

import "testing"

func TestCatalogReturnsEnglishByDefault(t *testing.T) {
	catalog := NewCatalog()

	got := catalog.Message("en-US", "health.ok")
	if got != "OK" {
		t.Fatalf("expected OK, got %q", got)
	}
}

func TestCatalogReturnsChineseMessage(t *testing.T) {
	catalog := NewCatalog()

	got := catalog.Message("zh-CN", "health.ok")
	if got != "正常" {
		t.Fatalf("expected Chinese message, got %q", got)
	}
}

func TestCatalogFallsBackToEnglish(t *testing.T) {
	catalog := NewCatalog()

	got := catalog.Message("fr-FR", "health.ok")
	if got != "OK" {
		t.Fatalf("expected English fallback, got %q", got)
	}
}

func TestCatalogReturnsCodeForMissingMessage(t *testing.T) {
	catalog := NewCatalog()

	got := catalog.Message("zh-CN", "missing.code")
	if got != "missing.code" {
		t.Fatalf("expected missing code fallback, got %q", got)
	}
}
