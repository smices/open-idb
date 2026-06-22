// SPDX-License-Identifier: MIT

package idp

import "testing"

func TestLifecycleForDirectoryStatus(t *testing.T) {
	tests := map[string]string{
		"active":   "active",
		"disabled": "disabled",
		"deleted":  "disabled",
		"unknown":  "locked",
		"other":    "locked",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := lifecycleForDirectoryStatus(input); got != want {
				t.Fatalf("lifecycleForDirectoryStatus(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestUsernameForDirectoryUser(t *testing.T) {
	if got := usernameForDirectoryUser(DirectoryUser{Email: "ada@example.test", ExternalUserID: "ou_1"}); got != "ada@example.test" {
		t.Fatalf("username = %q", got)
	}
	if got := usernameForDirectoryUser(DirectoryUser{ExternalUserID: "ou_1"}); got != "ou_1" {
		t.Fatalf("username = %q", got)
	}
}
