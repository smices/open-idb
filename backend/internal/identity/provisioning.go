// SPDX-License-Identifier: MIT

package identity

func ProvisionManagedUser(dir DirectoryUser, policy ProvisionPolicy) ManagedUserDraft {
	if !policy.AutoCreateManagedUsers {
		return ManagedUserDraft{}
	}

	username := dir.Email
	if username == "" {
		username = dir.ExternalUserID
	}

	sourceID := dir.SourceID

	return ManagedUserDraft{
		EntityID:        dir.EntityID,
		Username:        username,
		DisplayName:     dir.Name,
		Email:           dir.Email,
		Phone:           dir.Phone,
		AvatarURL:       dir.AvatarURL,
		LifecycleStatus: lifecycleStatusForDirectoryUser(dir.Status, policy),
		UserType:        UserTypeEmployee,
		PrimarySourceID: &sourceID,
		Locale:          policy.DefaultLocale,
	}
}

func lifecycleStatusForDirectoryUser(status DirectoryUserStatus, policy ProvisionPolicy) UserLifecycleStatus {
	switch status {
	case DirectoryUserStatusActive:
		return policy.DefaultLifecycleStatus
	case DirectoryUserStatusDisabled, DirectoryUserStatusDeleted:
		return UserLifecycleDisabled
	case DirectoryUserStatusUnknown:
		return UserLifecycleLocked
	default:
		return UserLifecycleLocked
	}
}
