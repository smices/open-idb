// SPDX-License-Identifier: MIT

package generated

import (
	"strings"
	"testing"
)

func TestOrganizationTreeDirectoryUserQueriesExcludeDeletedUsers(t *testing.T) {
	queries := map[string]string{
		"list by department":  listDirectoryUsersByDepartmentExternalID,
		"count by department": countDirectoryUsersByDepartmentExternalID,
		"list root":           listRootDirectoryUsers,
		"count root":          countRootDirectoryUsers,
		"search":              searchOrganizationTreeUsers,
	}
	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(query, "status <> 'deleted'") {
				t.Fatalf("organization tree query must exclude deleted directory users")
			}
		})
	}
}

func TestDepartmentMemberQueriesMatchCurrentProviderAliases(t *testing.T) {
	queries := map[string]string{
		"list":  listDirectoryUsersByDepartmentExternalID,
		"count": countDirectoryUsersByDepartmentExternalID,
	}
	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(query, "FROM directory_departments target_department") ||
				!strings.Contains(query, "target_department.raw_profile->>'department_id'") ||
				!strings.Contains(query, "target_department.raw_profile->>'open_department_id'") {
				t.Fatalf("%s department member query does not match provider ID aliases", name)
			}
		})
	}
}
