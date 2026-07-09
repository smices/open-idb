// SPDX-License-Identifier: MIT

package generated

import (
	"strings"
	"testing"
)

func TestUpdateManagedUserFromDirectoryPreservesUsernameWhenNull(t *testing.T) {
	want := "username = COALESCE($1, username)"
	if !strings.Contains(updateManagedUserFromDirectory, want) {
		t.Fatalf("UpdateManagedUserFromDirectory must preserve username when arg.Username is null")
	}
}
