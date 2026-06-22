// SPDX-License-Identifier: MIT

package adminapi

import "testing"

func TestConsoleAccessForEnterpriseAdminIncludesSystemCapability(t *testing.T) {
	scope, capabilities := consoleAccessForUser("manager", []string{"enterprise_admin"})

	if scope != "enterprise_admin" {
		t.Fatalf("scope = %q, want enterprise_admin", scope)
	}
	if !containsCapability(capabilities, "system") {
		t.Fatalf("capabilities = %#v, want system capability", capabilities)
	}
}

func TestConsoleAccessDoesNotExposeSystemAdminScope(t *testing.T) {
	scope, capabilities := consoleAccessForUser("admin", nil)

	if scope != "enterprise_admin" {
		t.Fatalf("scope = %q, want enterprise_admin", scope)
	}
	if !containsCapability(capabilities, "system") {
		t.Fatalf("capabilities = %#v, want system capability", capabilities)
	}
}

func containsCapability(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
