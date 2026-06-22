// SPDX-License-Identifier: MIT

package identity

import "github.com/smices/open-idb/internal/entity"

type SourceID string
type DirectoryUserID string
type UserID string

type SourceType string

const (
	SourceTypeFeishu   SourceType = "feishu"
	SourceTypeDingTalk SourceType = "dingtalk"
	SourceTypeWeCom    SourceType = "wecom"
	SourceTypeLDAP     SourceType = "ldap"
	SourceTypeLocal    SourceType = "local"
)

type DirectoryUserStatus string

const (
	DirectoryUserStatusActive   DirectoryUserStatus = "active"
	DirectoryUserStatusDisabled DirectoryUserStatus = "disabled"
	DirectoryUserStatusDeleted  DirectoryUserStatus = "deleted"
	DirectoryUserStatusUnknown  DirectoryUserStatus = "unknown"
)

type UserLifecycleStatus string

const (
	UserLifecycleActive   UserLifecycleStatus = "active"
	UserLifecycleDisabled UserLifecycleStatus = "disabled"
	UserLifecycleLocked   UserLifecycleStatus = "locked"
	UserLifecycleDeleted  UserLifecycleStatus = "deleted"
)

type UserType string

const (
	UserTypeEmployee       UserType = "employee"
	UserTypeContractor     UserType = "contractor"
	UserTypeServiceAccount UserType = "service_account"
)

type DirectoryUser struct {
	ID              DirectoryUserID
	EntityID        entity.ID
	SourceID        SourceID
	ExternalUserID  string
	ExternalUnionID string
	ExternalOpenID  string
	Name            string
	Email           string
	Phone           string
	AvatarURL       string
	Status          DirectoryUserStatus
}

type ManagedUserDraft struct {
	EntityID        entity.ID
	Username        string
	DisplayName     string
	Email           string
	Phone           string
	AvatarURL       string
	LifecycleStatus UserLifecycleStatus
	UserType        UserType
	PrimarySourceID *SourceID
	Locale          string
}

type AccountBinding struct {
	ID              string
	EntityID        entity.ID
	UserID          UserID
	SourceID        SourceID
	DirectoryUserID DirectoryUserID
	ProviderUID     string
	ProviderUnionID string
	IsPrimary       bool
}

type ProvisionPolicy struct {
	AutoCreateManagedUsers bool
	DefaultLifecycleStatus UserLifecycleStatus
	DefaultLocale          string
}
