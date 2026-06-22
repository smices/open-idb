// SPDX-License-Identifier: MIT

package identity

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/smices/open-idb/internal/entity"
)

func TestProvisionManagedUserFromDirectoryUser(t *testing.T) {
	sourceID := SourceID("src_feishu")
	dir := DirectoryUser{
		ID:              DirectoryUserID("dir_1"),
		EntityID:        entity.ID("entity_1"),
		SourceID:        sourceID,
		ExternalUserID:  "ou_1",
		ExternalUnionID: "union_1",
		Name:            "张三",
		Email:           "zhangsan@example.com",
		Phone:           "13800000000",
		AvatarURL:       "https://example.com/avatar.png",
		Status:          DirectoryUserStatusActive,
	}

	got := ProvisionManagedUser(dir, ProvisionPolicy{
		AutoCreateManagedUsers: true,
		DefaultLifecycleStatus: UserLifecycleActive,
		DefaultLocale:          "zh-CN",
	})

	want := ManagedUserDraft{
		EntityID:        entity.ID("entity_1"),
		Username:        "zhangsan@example.com",
		DisplayName:     "张三",
		Email:           "zhangsan@example.com",
		Phone:           "13800000000",
		AvatarURL:       "https://example.com/avatar.png",
		LifecycleStatus: UserLifecycleActive,
		UserType:        UserTypeEmployee,
		PrimarySourceID: &sourceID,
		Locale:          "zh-CN",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("draft mismatch (-want +got):\n%s", diff)
	}
}

func TestProvisionManagedUserUsesExternalIDWhenEmailMissing(t *testing.T) {
	dir := DirectoryUser{
		EntityID:       entity.ID("entity_1"),
		SourceID:       SourceID("src_feishu"),
		ExternalUserID: "ou_1",
		Name:           "No Email",
		Status:         DirectoryUserStatusActive,
	}

	got := ProvisionManagedUser(dir, ProvisionPolicy{
		AutoCreateManagedUsers: true,
		DefaultLifecycleStatus: UserLifecycleActive,
		DefaultLocale:          "en-US",
	})

	if got.Username != "ou_1" {
		t.Fatalf("expected external user id username, got %q", got.Username)
	}
}

func TestProvisionManagedUserReturnsZeroWhenAutoCreateDisabled(t *testing.T) {
	dir := DirectoryUser{
		EntityID:       entity.ID("entity_1"),
		SourceID:       SourceID("src_feishu"),
		ExternalUserID: "ou_1",
		Status:         DirectoryUserStatusActive,
	}

	got := ProvisionManagedUser(dir, ProvisionPolicy{
		AutoCreateManagedUsers: false,
		DefaultLifecycleStatus: UserLifecycleActive,
		DefaultLocale:          "en-US",
	})

	if got.Username != "" {
		t.Fatalf("expected zero draft, got %#v", got)
	}
}

func TestProvisionManagedUserMapsInactiveDirectoryStatusesToNonActiveLifecycles(t *testing.T) {
	tests := []struct {
		name string
		dir  DirectoryUserStatus
		want UserLifecycleStatus
	}{
		{
			name: "disabled directory user becomes disabled managed user",
			dir:  DirectoryUserStatusDisabled,
			want: UserLifecycleDisabled,
		},
		{
			name: "deleted directory user becomes disabled managed user to preserve sync history",
			dir:  DirectoryUserStatusDeleted,
			want: UserLifecycleDisabled,
		},
		{
			name: "unknown directory user becomes locked managed user",
			dir:  DirectoryUserStatusUnknown,
			want: UserLifecycleLocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := DirectoryUser{
				EntityID:       entity.ID("entity_1"),
				SourceID:       SourceID("src_feishu"),
				ExternalUserID: "ou_1",
				Status:         tt.dir,
			}

			got := ProvisionManagedUser(dir, ProvisionPolicy{
				AutoCreateManagedUsers: true,
				DefaultLifecycleStatus: UserLifecycleActive,
				DefaultLocale:          "en-US",
			})

			if got.LifecycleStatus != tt.want {
				t.Fatalf("expected lifecycle %q, got %q", tt.want, got.LifecycleStatus)
			}
			if got.LifecycleStatus == UserLifecycleActive {
				t.Fatalf("expected non-active lifecycle for directory status %q", tt.dir)
			}
		})
	}
}
