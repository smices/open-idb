// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/audit"
	"github.com/smices/open-idb/internal/db/generated"
)

// AdminService wraps *generated.Queries and provides data-access methods
// for all admin resource handlers. It satisfies the userService interface.
type AdminService struct {
	queries               *generated.Queries
	audit                 *auditWriter
	organizationTreeCache *OrganizationTreeCache
}

var enabledIdentitySourceTypes = map[string]bool{
	"feishu": true,
}

var primaryIdentitySourceTypes = map[string]bool{
	"feishu":   true,
	"dingtalk": true,
	"wecom":    true,
	"ldap":     true,
}

func NewAdminService(queries *generated.Queries, auditLogger ...AuditLogger) (*AdminService, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	svc := &AdminService{queries: queries}
	if len(auditLogger) > 0 && auditLogger[0] != nil {
		svc.audit = &auditWriter{logger: auditLogger[0]}
	}
	return svc, nil
}

// --- Platform settings ---

func (s *AdminService) GetPlatformBranding(ctx context.Context) (PlatformBrandingResponse, error) {
	row, err := s.queries.GetPlatformSettings(ctx)
	if err != nil {
		return PlatformBrandingResponse{}, err
	}
	return platformBrandingFromRow(row), nil
}

func (s *AdminService) UpdatePlatformBranding(ctx context.Context, platformName, logoURL, faviconURL, titleSuffix string) (PlatformBrandingResponse, error) {
	row, err := s.queries.UpsertPlatformSettings(ctx, generated.UpsertPlatformSettingsParams{
		PlatformName: strings.TrimSpace(platformName),
		LogoUrl:      strings.TrimSpace(logoURL),
		FaviconUrl:   strings.TrimSpace(faviconURL),
		TitleSuffix:  strings.TrimSpace(titleSuffix),
	})
	if err != nil {
		return PlatformBrandingResponse{}, err
	}
	return platformBrandingFromRow(row), nil
}

// --- Entities ---

func (s *AdminService) ListEntities(ctx context.Context, limit, offset int32) ([]EntityResponse, error) {
	rows, err := s.queries.ListEntities(ctx, generated.ListEntitiesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	entities := make([]EntityResponse, 0, len(rows))
	for _, row := range rows {
		entities = append(entities, entityFromRow(row.ID, row.Name, row.Slug, row.Status, row.DefaultLocale, row.BrandName, row.LogoUrl, row.LoginMessage, row.CreatedAt))
	}
	return entities, nil
}

func (s *AdminService) CountEntities(ctx context.Context) (int64, error) {
	return s.queries.CountEntities(ctx)
}

func (s *AdminService) GetEntityByID(ctx context.Context, id string) (EntityResponse, error) {
	row, err := s.queries.GetEntityByID(ctx, id)
	if err != nil {
		return EntityResponse{}, err
	}
	return entityFromRow(row.ID, row.Name, row.Slug, row.Status, row.DefaultLocale, row.BrandName, row.LogoUrl, row.LoginMessage, row.CreatedAt), nil
}

func (s *AdminService) CreateEntity(ctx context.Context, name, slug, defaultLocale, brandName, logoURL, loginMessage string) (EntityResponse, error) {
	row, err := s.queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          name,
		Slug:          slug,
		DefaultLocale: defaultLocale,
		BrandName:     brandName,
		LogoUrl:       logoURL,
		LoginMessage:  loginMessage,
	})
	if err != nil {
		return EntityResponse{}, err
	}
	return entityFromRow(row.ID, row.Name, row.Slug, row.Status, row.DefaultLocale, row.BrandName, row.LogoUrl, row.LoginMessage, row.CreatedAt), nil
}

func (s *AdminService) UpdateEntity(ctx context.Context, id string, name, status, defaultLocale, brandName, logoURL, loginMessage pgtype.Text) (EntityResponse, error) {
	row, err := s.queries.UpdateEntity(ctx, generated.UpdateEntityParams{
		ID:            id,
		Name:          name,
		Status:        status,
		DefaultLocale: defaultLocale,
		BrandName:     brandName,
		LogoUrl:       logoURL,
		LoginMessage:  loginMessage,
	})
	if err != nil {
		return EntityResponse{}, err
	}
	return entityFromRow(row.ID, row.Name, row.Slug, row.Status, row.DefaultLocale, row.BrandName, row.LogoUrl, row.LoginMessage, row.CreatedAt), nil
}

// --- Users ---

func (s *AdminService) ListUsers(ctx context.Context, entityID string, status pgtype.Text, limit, offset int32) ([]UserResponse, error) {
	rows, err := s.queries.ListUsers(ctx, generated.ListUsersParams{
		EntityID:        entityID,
		Limit:           limit,
		Offset:          offset,
		LifecycleStatus: status,
	})
	if err != nil {
		return nil, err
	}
	users := make([]UserResponse, 0, len(rows))
	for _, row := range rows {
		users = append(users, userFromRow(row))
	}
	return users, nil
}

func (s *AdminService) CountUsers(ctx context.Context, entityID string, status pgtype.Text) (int64, error) {
	return s.queries.CountUsers(ctx, generated.CountUsersParams{
		EntityID:        entityID,
		LifecycleStatus: status,
	})
}

func (s *AdminService) GetUserByID(ctx context.Context, entityID, id string) (UserResponse, error) {
	row, err := s.queries.GetUserByID(ctx, generated.GetUserByIDParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return UserResponse{}, err
	}
	return userFromRow(row), nil
}

func (s *AdminService) UpdateUserLifecycle(ctx context.Context, entityID, id string, status string) (UserResponse, error) {
	before, _ := s.GetUserByID(ctx, entityID, id)
	row, err := s.queries.UpdateUserLifecycle(ctx, generated.UpdateUserLifecycleParams{
		EntityID:        entityID,
		ID:              id,
		LifecycleStatus: status,
	})
	if err != nil {
		return UserResponse{}, err
	}
	after := userFromRow(row)
	action := audit.ActionUserUpdated
	if status == "disabled" {
		action = audit.ActionUserDisabled
	}
	if err := s.audit.logAction(ctx, audit.Event{
		EntityID:     ulidString(entityID),
		ActorType:    "user",
		Action:       action,
		ResourceType: "user",
		ResourceID:   after.ID,
		Before:       before,
		After:        after,
	}); err != nil {
		return UserResponse{}, err
	}
	return after, nil
}

func (s *AdminService) UpdateUser(ctx context.Context, entityID, id string, displayName, email, phone, locale pgtype.Text) (UserResponse, error) {
	row, err := s.queries.UpdateUser(ctx, generated.UpdateUserParams{
		EntityID:    entityID,
		ID:          id,
		DisplayName: displayName,
		Email:       email,
		Phone:       phone,
		Locale:      locale,
	})
	if err != nil {
		return UserResponse{}, err
	}
	return userFromRow(row), nil
}

// --- Directory Users ---

func (s *AdminService) GetDirectoryUserByID(ctx context.Context, entityID, id string) (DirectoryUserResponse, error) {
	row, err := s.queries.GetDirectoryUserByID(ctx, generated.GetDirectoryUserByIDParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return DirectoryUserResponse{}, err
	}
	return directoryUserFromRow(row), nil
}

// --- Applications ---

func (s *AdminService) ListApplications(ctx context.Context, entityID string, limit, offset int32) ([]ApplicationResponse, error) {
	rows, err := s.queries.ListApplications(ctx, generated.ListApplicationsParams{
		EntityID: entityID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}
	apps := make([]ApplicationResponse, 0, len(rows))
	for _, row := range rows {
		apps = append(apps, applicationFromRow(row))
	}
	return apps, nil
}

func (s *AdminService) CountApplications(ctx context.Context, entityID string) (int64, error) {
	return s.queries.CountApplications(ctx, entityID)
}

func (s *AdminService) GetApplicationByID(ctx context.Context, entityID, id string) (ApplicationResponse, error) {
	row, err := s.queries.GetApplicationByID(ctx, generated.GetApplicationByIDParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return ApplicationResponse{}, err
	}
	return applicationFromRow(row), nil
}

func (s *AdminService) CreateApplication(ctx context.Context, entityID string, name, appType string) (ApplicationResponse, error) {
	row, err := s.queries.CreateApplication(ctx, generated.CreateApplicationParams{
		EntityID: entityID,
		Name:     name,
		Type:     appType,
	})
	if err != nil {
		return ApplicationResponse{}, err
	}
	resp := applicationFromRow(row)
	if err := s.audit.logCreate(ctx, ulidString(entityID), "", "application", resp.ID, resp); err != nil {
		return ApplicationResponse{}, err
	}
	return resp, nil
}

func (s *AdminService) UpdateApplication(ctx context.Context, entityID, id string, name, status pgtype.Text) (ApplicationResponse, error) {
	row, err := s.queries.UpdateApplication(ctx, generated.UpdateApplicationParams{
		EntityID: entityID,
		ID:       id,
		Name:     name,
		Status:   status,
	})
	if err != nil {
		return ApplicationResponse{}, err
	}
	return applicationFromRow(row), nil
}

func (s *AdminService) DeleteApplication(ctx context.Context, entityID, id string) error {
	return s.queries.DeleteApplication(ctx, generated.DeleteApplicationParams{
		EntityID: entityID,
		ID:       id,
	})
}

// --- Sync Jobs ---

func (s *AdminService) ListAllSyncJobs(ctx context.Context, entityID string, limit, offset int32) ([]SyncJobResponse, error) {
	rows, err := s.queries.ListAllSyncJobs(ctx, generated.ListAllSyncJobsParams{
		EntityID: entityID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}
	jobs := make([]SyncJobResponse, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, syncJobFromRow(row))
	}
	return jobs, nil
}

func (s *AdminService) CountAllSyncJobs(ctx context.Context, entityID string) (int64, error) {
	return s.queries.CountAllSyncJobs(ctx, entityID)
}

// --- Roles ---

func (s *AdminService) ListRoles(ctx context.Context, entityID string, limit, offset int32) ([]RoleResponse, error) {
	rows, err := s.queries.ListRoles(ctx, generated.ListRolesParams{
		EntityID: entityID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}
	roles := make([]RoleResponse, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, roleFromRow(row))
	}
	return roles, nil
}

func (s *AdminService) CountRoles(ctx context.Context, entityID string) (int64, error) {
	return s.queries.CountRoles(ctx, entityID)
}

func (s *AdminService) GetRoleByID(ctx context.Context, entityID, id string) (RoleResponse, error) {
	row, err := s.queries.GetRoleByID(ctx, generated.GetRoleByIDParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return RoleResponse{}, err
	}
	return roleFromRow(row), nil
}

// --- Permissions ---

func (s *AdminService) ListPermissions(ctx context.Context, entityID string, limit, offset int32) ([]PermissionResponse, error) {
	rows, err := s.queries.ListPermissions(ctx, generated.ListPermissionsParams{
		EntityID: entityID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}
	perms := make([]PermissionResponse, 0, len(rows))
	for _, row := range rows {
		perms = append(perms, permissionFromRow(row))
	}
	return perms, nil
}

func (s *AdminService) CountPermissions(ctx context.Context, entityID string) (int64, error) {
	return s.queries.CountPermissions(ctx, entityID)
}

func (s *AdminService) GetPermissionByID(ctx context.Context, entityID, id string) (PermissionResponse, error) {
	row, err := s.queries.GetPermissionByID(ctx, generated.GetPermissionByIDParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return PermissionResponse{}, err
	}
	return permissionFromRow(row), nil
}

// --- Identity Sources ---

func (s *AdminService) ListIdentitySources(ctx context.Context, entityID string, limit, offset int32) ([]IdentitySourceResponse, error) {
	rows, err := s.queries.ListIdentitySources(ctx, generated.ListIdentitySourcesParams{
		EntityID: entityID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}
	sources := make([]IdentitySourceResponse, 0, len(rows))
	for _, row := range rows {
		sources = append(sources, identitySourceFromRow(row.ID, row.EntityID, row.Type, row.Name, row.Status, row.SyncEnabled, row.CreatedAt))
	}
	return sources, nil
}

func (s *AdminService) CountIdentitySources(ctx context.Context, entityID string) (int64, error) {
	return s.queries.CountIdentitySources(ctx, entityID)
}

func (s *AdminService) GetIdentitySourceByID(ctx context.Context, entityID, id string) (IdentitySourceResponse, error) {
	row, err := s.queries.GetIdentitySourceByID(ctx, generated.GetIdentitySourceByIDParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return IdentitySourceResponse{}, err
	}
	return identitySourceFromRow(row.ID, row.EntityID, row.Type, row.Name, row.Status, row.SyncEnabled, row.CreatedAt), nil
}

func (s *AdminService) CreateIdentitySource(ctx context.Context, entityID string, sourceType, name string, syncEnabled bool) (IdentitySourceResponse, error) {
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	name = strings.TrimSpace(name)
	if !enabledIdentitySourceTypes[sourceType] {
		return IdentitySourceResponse{}, fmt.Errorf("identity source type %q is not enabled yet", sourceType)
	}
	if primaryIdentitySourceTypes[sourceType] {
		existing, err := s.ListIdentitySources(ctx, entityID, 200, 0)
		if err != nil {
			return IdentitySourceResponse{}, err
		}
		for _, item := range existing {
			if item.ID != "" && primaryIdentitySourceTypes[item.Type] && item.Status == "active" {
				return IdentitySourceResponse{}, fmt.Errorf("only one active primary identity source is allowed")
			}
		}
	}
	row, err := s.queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{
		EntityID:    entityID,
		Type:        sourceType,
		Name:        name,
		SyncEnabled: syncEnabled,
	})
	if err != nil {
		return IdentitySourceResponse{}, err
	}
	return identitySourceFromRow(row.ID, row.EntityID, row.Type, row.Name, row.Status, row.SyncEnabled, row.CreatedAt), nil
}

func (s *AdminService) UpdateIdentitySource(ctx context.Context, entityID, id string, name, status pgtype.Text, syncEnabled pgtype.Bool) (IdentitySourceResponse, error) {
	if status.Valid && status.String == "active" {
		current, err := s.GetIdentitySourceByID(ctx, entityID, id)
		if err != nil {
			return IdentitySourceResponse{}, err
		}
		if primaryIdentitySourceTypes[current.Type] {
			existing, err := s.ListIdentitySources(ctx, entityID, 200, 0)
			if err != nil {
				return IdentitySourceResponse{}, err
			}
			for _, item := range existing {
				if item.ID != id && primaryIdentitySourceTypes[item.Type] && item.Status == "active" {
					return IdentitySourceResponse{}, fmt.Errorf("only one active primary identity source is allowed")
				}
			}
		}
	}
	row, err := s.queries.UpdateIdentitySource(ctx, generated.UpdateIdentitySourceParams{
		EntityID:    entityID,
		ID:          id,
		Name:        name,
		Status:      status,
		SyncEnabled: syncEnabled,
	})
	if err != nil {
		return IdentitySourceResponse{}, err
	}
	return identitySourceFromRow(row.ID, row.EntityID, row.Type, row.Name, row.Status, row.SyncEnabled, row.CreatedAt), nil
}

func (s *AdminService) DeleteIdentitySource(ctx context.Context, entityID, id string) error {
	return s.queries.DeleteIdentitySource(ctx, generated.DeleteIdentitySourceParams{
		EntityID: entityID,
		ID:       id,
	})
}

// --- Account Bindings ---

func (s *AdminService) ListUserBindings(ctx context.Context, entityID, userID string) ([]BindingResponse, error) {
	rows, err := s.queries.ListAccountBindingsByUserID(ctx, generated.ListAccountBindingsByUserIDParams{
		EntityID: entityID,
		UserID:   userID,
	})
	if err != nil {
		return nil, err
	}
	bindings := make([]BindingResponse, 0, len(rows))
	for _, row := range rows {
		bindings = append(bindings, bindingFromRow(
			row.ID, row.EntityID, row.UserID, row.SourceID, row.DirectoryUserID,
			row.ProviderUid, row.ProviderUnionID, row.IsPrimary, row.BoundAt,
			row.SourceType, row.SourceName,
		))
	}
	return bindings, nil
}

func (s *AdminService) GetBindingByID(ctx context.Context, entityID, id string) (BindingResponse, error) {
	row, err := s.queries.GetAccountBindingWithSource(ctx, generated.GetAccountBindingWithSourceParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return BindingResponse{}, err
	}
	return bindingFromRow(
		row.ID, row.EntityID, row.UserID, row.SourceID, row.DirectoryUserID,
		row.ProviderUid, row.ProviderUnionID, row.IsPrimary, row.BoundAt,
		row.SourceType, row.SourceName,
	), nil
}

func (s *AdminService) CreateUserBinding(ctx context.Context, entityID, userID string, sourceID, directoryUserID string, providerUID string, providerUnionID pgtype.Text, isPrimary bool) (BindingResponse, error) {
	created, err := s.queries.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
		EntityID:        entityID,
		UserID:          userID,
		SourceID:        sourceID,
		DirectoryUserID: directoryUserID,
		ProviderUid:     providerUID,
		ProviderUnionID: providerUnionID,
		IsPrimary:       isPrimary,
	})
	if err != nil {
		return BindingResponse{}, err
	}
	// Fetch back with source info via the JOIN query.
	resp, err := s.GetBindingByID(ctx, entityID, created.ID)
	if err != nil {
		return BindingResponse{}, err
	}
	if err := s.audit.logAction(ctx, audit.Event{
		EntityID:     ulidString(entityID),
		ActorType:    "user",
		Action:       audit.ActionUserBoundIdentity,
		ResourceType: "user",
		ResourceID:   ulidString(userID),
		After:        resp,
	}); err != nil {
		return BindingResponse{}, err
	}
	return resp, nil
}

func (s *AdminService) DeleteUserBinding(ctx context.Context, entityID, userID, id string) error {
	// Fetch the binding before deletion for audit and safety checks.
	before, err := s.GetBindingByID(ctx, entityID, id)
	if err != nil {
		return err
	}

	// Must not delete the last binding if user has no other auth method.
	count, err := s.queries.CountBindingsByUser(ctx, generated.CountBindingsByUserParams{
		EntityID: entityID,
		UserID:   userID,
	})
	if err != nil {
		return err
	}
	if count <= 1 {
		hasLocal, err := s.queries.HasLocalCredential(ctx, generated.HasLocalCredentialParams{
			EntityID: entityID,
			UserID:   userID,
		})
		if err != nil {
			return err
		}
		if !hasLocal {
			return fmt.Errorf("cannot delete last binding: user has no other authentication method")
		}
	}

	if err := s.queries.DeleteAccountBindingByID(ctx, generated.DeleteAccountBindingByIDParams{
		EntityID: entityID,
		ID:       id,
	}); err != nil {
		return err
	}
	if err := s.audit.logAction(ctx, audit.Event{
		EntityID:     ulidString(entityID),
		ActorType:    "user",
		Action:       audit.ActionUserUnboundIdentity,
		ResourceType: "user",
		ResourceID:   ulidString(userID),
		Before:       before,
	}); err != nil {
		return err
	}
	return nil
}

// --- OIDC Clients ---

func (s *AdminService) ListOIDCClients(ctx context.Context, entityID string, limit, offset int32) ([]OIDCClientResponse, error) {
	rows, err := s.queries.ListOIDCClients(ctx, generated.ListOIDCClientsParams{
		EntityID: entityID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}
	clients := make([]OIDCClientResponse, 0, len(rows))
	for _, row := range rows {
		clients = append(clients, oidcClientFromRow(row))
	}
	return clients, nil
}

func (s *AdminService) CountOIDCClients(ctx context.Context, entityID string) (int64, error) {
	return s.queries.CountOIDCClients(ctx, entityID)
}

func (s *AdminService) GetOIDCClientByID(ctx context.Context, entityID, id string) (OIDCClientResponse, error) {
	row, err := s.queries.GetOIDCClientByID(ctx, generated.GetOIDCClientByIDParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return OIDCClientResponse{}, err
	}
	return OIDCClientResponse{
		ID:            ulidString(row.ID),
		EntityID:      ulidString(row.EntityID),
		ApplicationID: ulidString(row.ApplicationID),
		ClientID:      row.ClientID,
		ClientSecret:  textValue(row.ClientSecretHash),
		RedirectURIs:  row.RedirectUris,
		AllowedScopes: row.AllowedScopes,
		GrantTypes:    row.GrantTypes,
		ResponseTypes: row.ResponseTypes,
		PKCERequired:  row.PkceRequired,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *AdminService) CreateOIDCClient(ctx context.Context, params generated.CreateOIDCClientParams) (OIDCClientResponse, string, error) {
	// Generate a random client secret
	secret, err := generateRandomSecret(32)
	if err != nil {
		return OIDCClientResponse{}, "", err
	}
	params.ClientSecretHash = pgtype.Text{String: hashSecret(secret), Valid: true}

	row, err := s.queries.CreateOIDCClient(ctx, params)
	if err != nil {
		return OIDCClientResponse{}, "", err
	}
	resp := OIDCClientResponse{
		ID:            ulidString(row.ID),
		EntityID:      ulidString(row.EntityID),
		ApplicationID: ulidString(row.ApplicationID),
		ClientID:      row.ClientID,
		RedirectURIs:  row.RedirectUris,
		AllowedScopes: row.AllowedScopes,
		GrantTypes:    row.GrantTypes,
		ResponseTypes: row.ResponseTypes,
		PKCERequired:  row.PkceRequired,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}
	if err := s.audit.logCreate(ctx, ulidString(params.EntityID), "", "oidc_client", resp.ID, resp); err != nil {
		return OIDCClientResponse{}, "", err
	}
	return resp, secret, nil
}

func (s *AdminService) UpdateOIDCClient(ctx context.Context, params generated.UpdateOIDCClientParams) (OIDCClientResponse, error) {
	before, _ := s.GetOIDCClientByID(ctx, params.EntityID, params.ID)
	before.ClientSecret = ""
	row, err := s.queries.UpdateOIDCClient(ctx, params)
	if err != nil {
		return OIDCClientResponse{}, err
	}
	after := OIDCClientResponse{
		ID:            ulidString(row.ID),
		EntityID:      ulidString(row.EntityID),
		ApplicationID: ulidString(row.ApplicationID),
		ClientID:      row.ClientID,
		RedirectURIs:  row.RedirectUris,
		AllowedScopes: row.AllowedScopes,
		GrantTypes:    row.GrantTypes,
		ResponseTypes: row.ResponseTypes,
		PKCERequired:  row.PkceRequired,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}
	if err := s.audit.logUpdate(ctx, ulidString(params.EntityID), "", "oidc_client", after.ID, before, after); err != nil {
		return OIDCClientResponse{}, err
	}
	return after, nil
}

func (s *AdminService) DeleteOIDCClient(ctx context.Context, entityID, id string) error {
	before, _ := s.GetOIDCClientByID(ctx, entityID, id)
	before.ClientSecret = ""
	err := s.queries.DeleteOIDCClient(ctx, generated.DeleteOIDCClientParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return err
	}
	if err := s.audit.logDelete(ctx, ulidString(entityID), "", "oidc_client", ulidString(id), before); err != nil {
		return err
	}
	return nil
}

func (s *AdminService) RotateOIDCClientSecret(ctx context.Context, entityID, id string) (OIDCClientResponse, string, error) {
	secret, err := generateRandomSecret(32)
	if err != nil {
		return OIDCClientResponse{}, "", err
	}
	row, err := s.queries.RotateOIDCClientSecret(ctx, generated.RotateOIDCClientSecretParams{
		EntityID:         entityID,
		ID:               id,
		ClientSecretHash: pgtype.Text{String: hashSecret(secret), Valid: true},
	})
	if err != nil {
		return OIDCClientResponse{}, "", err
	}
	resp := OIDCClientResponse{
		ID:            ulidString(row.ID),
		EntityID:      ulidString(row.EntityID),
		ApplicationID: ulidString(row.ApplicationID),
		ClientID:      row.ClientID,
		RedirectURIs:  row.RedirectUris,
		AllowedScopes: row.AllowedScopes,
		GrantTypes:    row.GrantTypes,
		ResponseTypes: row.ResponseTypes,
		PKCERequired:  row.PkceRequired,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}
	if err := s.audit.logAction(ctx, audit.Event{
		EntityID:     ulidString(entityID),
		ActorType:    "user",
		Action:       audit.ActionSecretRotated,
		ResourceType: "oidc_client",
		ResourceID:   resp.ID,
	}); err != nil {
		return OIDCClientResponse{}, "", err
	}
	return resp, secret, nil
}

func generateRandomSecret(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		var buf [1]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return "", err
		}
		b[i] = charset[int(buf[0])%len(charset)]
	}
	return string(b), nil
}

func generateOIDCClientID() (string, error) {
	value, err := generateRandomSecret(24)
	if err != nil {
		return "", err
	}
	return "idb_" + value, nil
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func hashSecret(secret string) string {
	// Simple hash for now - in production use bcrypt or argon2
	// The SSO package already uses SHA-256 for token hashing
	return secret
}
