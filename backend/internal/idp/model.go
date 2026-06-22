// SPDX-License-Identifier: MIT

package idp

import (
	"context"
	"strings"
)

type DirectoryDepartment struct {
	ExternalDepartmentID       string
	ParentExternalDepartmentID string
	Name                       string
	RawProfile                 []byte
}

type DirectoryUser struct {
	ExternalUserID  string
	ExternalUnionID string
	ExternalOpenID  string
	Name            string
	Email           string
	Phone           string
	AvatarURL       string
	Status          string
	RawProfile      []byte
}

type FullSyncData struct {
	Departments []DirectoryDepartment
	Users       []DirectoryUser
}

type DirectorySyncEvent struct {
	EventType    string
	ObjectType   string
	ObjectID     string
	ObjectIDType string
	EventID      string
	Raw          map[string]interface{}
}

func (e DirectorySyncEvent) IsUserEvent() bool {
	return e.ObjectType == "user"
}

func (e DirectorySyncEvent) IsDepartmentEvent() bool {
	return e.ObjectType == "department"
}

func (e DirectorySyncEvent) IsDeleteEvent() bool {
	eventType := strings.ToLower(strings.TrimSpace(e.EventType))
	return strings.Contains(eventType, "delete")
}

func (e DirectorySyncEvent) IsKnown() bool {
	return e.IsUserEvent() || e.IsDepartmentEvent()
}

type SyncMode string

const (
	SyncModeFull        SyncMode = "full"
	SyncModeIncremental SyncMode = "incremental"
)

type DirectoryProvider interface {
	FullSync(ctx context.Context) (FullSyncData, error)
	IncrementalSync(ctx context.Context, events []DirectorySyncEvent) (FullSyncData, error)
}
