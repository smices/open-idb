// SPDX-License-Identifier: MIT

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

const API_TARGET = import.meta.env.PUBLIC_API_TARGET || '';

type JsonLike = Record<string, unknown>;

async function handleResponse<T>(response: Response): Promise<T> {
  if (response.status === 401 || response.status === 403) {
    const err = new Error('unauthorized') as Error & { status?: number };
    err.status = response.status;
    throw err;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    const payload = (await response.json()) as JsonLike;
    if (!response.ok) {
      const msg = (typeof payload?.error_description === 'string'
        ? payload.error_description
        : typeof payload?.message === 'string'
          ? payload.message
          : 'Request failed') as string;
      const err = new Error(msg) as Error & { status?: number };
      err.status = response.status;
      throw err;
    }
    return payload as T;
  }

  const text = await response.text();
  if (!response.ok) {
    const err = new Error(text || 'Request failed') as Error & { status?: number };
    err.status = response.status;
    throw err;
  }
  return text as T;
}

export async function apiRequest<T>(
  path: string,
  options: {
    method?: HttpMethod;
    body?: unknown;
    headers?: Record<string, string>;
    signal?: AbortSignal;
    skipJson?: boolean;
  } = {},
): Promise<T> {
  const headers: Record<string, string> = {
    ...(options.headers || {}),
  };

  const hasFormBody = options.body instanceof FormData;
  const body = hasFormBody || options.skipJson ? options.body : options.body ? JSON.stringify(options.body) : undefined;

  if (!hasFormBody && options.body && !options.headers?.['Content-Type']) {
    headers['Content-Type'] = 'application/json';
  }

  const response = await fetch(`${API_TARGET}${path}`, {
    method: options.method ?? 'GET',
    headers,
    credentials: 'include',
    body: body as BodyInit | undefined,
    signal: options.signal,
    redirect: 'follow',
  });

  return handleResponse<T>(response);
}

function queryString(params?: Record<string, string | number | undefined>): string {
  if (!params) return '';
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue;
    search.set(key, String(value));
  }
  return search.toString() ? `?${search.toString()}` : '';
}

export interface CurrentUser {
  id: string;
  entity_id: string;
  username: string;
  display_name: string;
  email?: string;
  phone?: string;
  avatar_url?: string;
  locale: string;
  weak_password?: boolean;
  console_scope?: 'user' | 'enterprise_admin';
  capabilities?: Array<'user' | 'enterprise' | 'system'>;
}

export interface Entity {
  id: string;
  name: string;
  slug: string;
  status: string;
  default_locale: string;
  brand_name: string;
  logo_url: string;
  login_message: string;
  created_at: string;
}

export interface EntityListResponse {
  items: Entity[];
  total: number;
  limit: number;
  offset: number;
}

export interface DashboardSummary {
  users: number;
  active_users: number;
  new_users: number;
  application_activity: number;
  pending_authorization: number;
  sync_health: string;
}

export interface AuditLogEntry {
  id: string;
  entity_id: string;
  actor_user_id?: string;
  actor_type: string;
  action: string;
  resource_type: string;
  resource_id: string;
  before?: unknown;
  after?: unknown;
  ip?: string;
  user_agent?: string;
  trace_id?: string;
  created_at: string;
}

export interface AuditLogListResponse {
  items: AuditLogEntry[];
  total: number;
  limit: number;
  offset: number;
}

export interface SyncJob {
  id: string;
  entity_id: string;
  source_id: string;
  type: string;
  provider: string;
  status: string;
  trace_id: string;
  started_at: string;
  finished_at?: string;
  error_message?: string;
  stats?: unknown;
}

export interface SyncJobListResponse {
  items: SyncJob[];
  total: number;
  limit: number;
  offset: number;
}

export interface DirectoryUser {
  id: string;
  entity_id: string;
  source_id: string;
  external_user_id: string;
  external_union_id?: string;
  external_open_id?: string;
  name: string;
  email?: string;
  phone?: string;
  avatar_url?: string;
  status: string;
  raw_profile?: unknown;
  last_synced_at: string;
  created_at: string;
  updated_at: string;
}

export interface DirectoryUserListResponse {
  items: DirectoryUser[];
  total: number;
  limit: number;
  offset: number;
}

export interface Organization {
  id: string;
  entity_id: string;
  name: string;
  parent_id?: string;
  created_at: string;
  updated_at: string;
}

export interface Department {
  id: string;
  entity_id: string;
  organization_id: string;
  name: string;
  parent_id?: string;
  source_id?: string;
  external_department_id?: string;
  created_at: string;
  updated_at: string;
}

export interface Group {
  id: string;
  entity_id: string;
  name: string;
  type: string;
  created_at: string;
  updated_at: string;
}

export interface GroupMember {
  user_id: string;
  username: string;
  display_name: string;
  email?: string;
  lifecycle_status: string;
}

export interface ResourceScope {
  id: string;
  entity_id: string;
  type: string;
  key: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface PagedResponse<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

export interface User {
  id: string;
  entity_id: string;
  username: string;
  display_name: string;
  email?: string;
  phone?: string;
  avatar_url?: string;
  locale: string;
  lifecycle_status: string;
  user_type: string;
  primary_source_id?: string;
  created_at?: string;
  updated_at?: string;
}

export interface UserListResponse {
  items: User[];
  total: number;
  limit: number;
  offset: number;
}

export interface UserSession {
  id: string;
  entity_id: string;
  user_id: string;
  device_id?: string;
  ip?: string;
  user_agent?: string;
  login_method: string;
  status: string;
  created_at: string;
  expires_at?: string;
}

export interface UserSessionListResponse {
  items: UserSession[];
}

export interface AccountBinding {
  id: string;
  user_id: string;
  source_id: string;
  source_type: string;
  source_name: string;
  directory_user_id: string;
  provider_uid: string;
  provider_union_id?: string;
  is_primary: boolean;
  bound_at: string;
}

export interface UpdateUserRequest {
  display_name?: string;
  email?: string;
  phone?: string;
  locale?: string;
}

export interface Application {
  id: string;
  entity_id: string;
  name: string;
  type: string;
  status: string;
  created_at?: string;
  updated_at?: string;
}

export interface ApplicationListResponse {
  applications: Application[];
  total: number;
  limit?: number;
  offset?: number;
}

export interface IdentitySource {
  id: string;
  entity_id: string;
  type: string;
  name: string;
  status: string;
  sync_enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface IdentitySourceListResponse {
  items?: IdentitySource[];
  sources?: IdentitySource[];
  total: number;
  limit?: number;
  offset?: number;
}

export type IdentitySourceSyncMode = 'full' | 'incremental';

export interface Role {
  id: string;
  entity_id: string;
  name: string;
  code: string;
  description?: string;
  created_at?: string;
}

export interface Permission {
  id: string;
  entity_id: string;
  code: string;
  name: string;
  type: string;
}

export interface PermissionCheckResponse {
  allowed: boolean;
}

export interface OIDCClient {
  id: string;
  entity_id: string;
  application_id: string;
  client_id: string;
  client_secret?: string;
  redirect_uris: string[];
  allowed_scopes?: string[];
  grant_types?: string[];
  response_types?: string[];
  pkce_required?: boolean;
  status?: string;
  created_at?: string;
  updated_at?: string;
}

export interface OIDCClientCreateResponse {
  client: OIDCClient;
  client_secret: string;
}

export interface LegacyAppUser {
  id: string;
  entity_id: string;
  application_id: string;
  user_id: string;
  username: string;
  legacy_user_identifier: string;
  auth_scheme: string;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface LegacyUsersListResponse {
  items: LegacyAppUser[];
  total: number;
  limit: number;
  offset: number;
}

export interface ApplicationAssignment {
  id: string;
  entity_id: string;
  application_id: string;
  subject_type: 'user' | 'group' | 'department' | 'role';
  subject_id: string;
  effect: string;
}

export interface AssignmentListResponse {
  assignments: ApplicationAssignment[];
  total: number;
}

export interface IMProviderConfig {
  id?: string;
  provider: string;
  display_name: string;
  status: string;
  oauth_configured: boolean;
  sync_enabled: boolean;
  config: Record<string, string>;
}

export interface MCPConnector {
  id?: string;
  name: string;
  endpoint_url: string;
  auth_type: 'none' | 'api_key' | 'bearer' | 'basic';
  status: 'active' | 'disabled';
  description: string;
}

export interface LoginProvider {
  provider: string;
  display_name: string;
  oauth_url?: string;
}

export type LoginMode = 'app' | 'user' | 'entity_admin';

export interface LoginContext {
  mode: LoginMode;
  entity: { id?: string; slug?: string; name?: string; brand_name?: string; logo_url?: string; login_message?: string } | null;
  application: { id?: string; name?: string } | null;
  methods: string[];
  allow_entity_selection: boolean;
  reason?: string;
  return_to: string;
  preferred_provider?: string;
  auto_redirect_url?: string;
}

export interface LegacyLoginResponse {
  code: string;
  user_id: string;
  entity_id: string;
  application_id: string;
  username: string;
  display_name: string;
  session: string;
}

export const api = {
  me: (): Promise<CurrentUser> => apiRequest<CurrentUser>('/api/admin/v1/me'),
  listEntities: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<EntityListResponse>(`/api/admin/v1/entities${suffix}`);
  },
  getEntity: (id: string) => apiRequest<Entity>(`/api/admin/v1/entities/${encodeURIComponent(id)}`),
  createEntity: (payload: { name: string; slug: string; default_locale?: string; brand_name?: string; logo_url?: string; login_message?: string }) =>
    apiRequest<Entity>('/api/admin/v1/entities', { method: 'POST', body: payload }),
  updateEntity: (id: string, payload: { name?: string; status?: string; default_locale?: string; brand_name?: string; logo_url?: string; login_message?: string }) =>
    apiRequest<Entity>(`/api/admin/v1/entities/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  listLoginProviders: (entityId?: string) => {
    const suffix = queryString({ entity_id: entityId });
    return apiRequest<LoginProvider[]>(`/api/admin/v1/auth/providers${suffix}`);
  },
  getLoginContext: (params?: { path?: string; return_to?: string }) => {
    const suffix = queryString({ path: params?.path, return_to: params?.return_to });
    return apiRequest<LoginContext>(`/api/admin/v1/auth/context${suffix}`);
  },
  legacyLogin: (payload: { entity_id: string; application_id: string; username: string; password: string }) =>
    apiRequest<LegacyLoginResponse>('/api/admin/v1/login/legacy', {
      method: 'POST',
      body: payload,
    }),
  dashboardSummary: (): Promise<DashboardSummary> =>
    apiRequest<DashboardSummary>('/api/admin/v1/dashboard/summary'),
  listAuditLogs: (params?: {
    action?: string;
    resource_type?: string;
    actor_type?: string;
    limit?: number;
    offset?: number;
  }) => {
    const suffix = queryString({
      action: params?.action,
      resource_type: params?.resource_type,
      actor_type: params?.actor_type,
      limit: params?.limit,
      offset: params?.offset,
    });
    return apiRequest<AuditLogListResponse>(`/api/admin/v1/audit-logs${suffix}`);
  },
  listSyncJobs: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<SyncJobListResponse>(`/api/admin/v1/sync-jobs${suffix}`);
  },
  listDirectoryUsers: (params?: { source_id?: string; limit?: number; offset?: number }) => {
    const suffix = queryString({
      source_id: params?.source_id,
      limit: params?.limit,
      offset: params?.offset,
    });
    return apiRequest<DirectoryUserListResponse>(`/api/admin/v1/directory-users${suffix}`);
  },
  getDirectoryUser: (id: string) =>
    apiRequest<DirectoryUser>(`/api/admin/v1/directory-users/${encodeURIComponent(id)}`),
  listOrganizations: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<PagedResponse<Organization>>(`/api/admin/v1/organizations${suffix}`);
  },
  getOrganization: (id: string) =>
    apiRequest<Organization>(`/api/admin/v1/organizations/${encodeURIComponent(id)}`),
  createOrganization: (payload: { name: string; parent_id?: string }) =>
    apiRequest<Organization>('/api/admin/v1/organizations', { method: 'POST', body: payload }),
  updateOrganization: (id: string, payload: { name?: string; parent_id?: string }) =>
    apiRequest<Organization>(`/api/admin/v1/organizations/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteOrganization: (id: string) =>
    apiRequest<void>(`/api/admin/v1/organizations/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  listDepartments: (params?: { organization_id?: string; limit?: number; offset?: number }) => {
    const suffix = queryString({
      organization_id: params?.organization_id,
      limit: params?.limit,
      offset: params?.offset,
    });
    return apiRequest<PagedResponse<Department>>(`/api/admin/v1/departments${suffix}`);
  },
  getDepartment: (id: string) =>
    apiRequest<Department>(`/api/admin/v1/departments/${encodeURIComponent(id)}`),
  createDepartment: (payload: {
    organization_id: string;
    name: string;
    parent_id?: string;
    source_id?: string;
    external_department_id?: string;
  }) => apiRequest<Department>('/api/admin/v1/departments', { method: 'POST', body: payload }),
  updateDepartment: (id: string, payload: { name?: string; parent_id?: string }) =>
    apiRequest<Department>(`/api/admin/v1/departments/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteDepartment: (id: string) =>
    apiRequest<void>(`/api/admin/v1/departments/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  listGroups: (params?: { type?: string; limit?: number; offset?: number }) => {
    const suffix = queryString({ type: params?.type, limit: params?.limit, offset: params?.offset });
    return apiRequest<PagedResponse<Group>>(`/api/admin/v1/groups${suffix}`);
  },
  getGroup: (id: string) => apiRequest<Group>(`/api/admin/v1/groups/${encodeURIComponent(id)}`),
  createGroup: (payload: { name: string; type?: string }) =>
    apiRequest<Group>('/api/admin/v1/groups', { method: 'POST', body: payload }),
  updateGroup: (id: string, payload: { name?: string }) =>
    apiRequest<Group>(`/api/admin/v1/groups/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteGroup: (id: string) =>
    apiRequest<void>(`/api/admin/v1/groups/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  listGroupMembers: (groupId: string, params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<PagedResponse<GroupMember>>(`/api/admin/v1/groups/${encodeURIComponent(groupId)}/members${suffix}`);
  },
  addGroupMember: (groupId: string, userId: string) =>
    apiRequest<void>(`/api/admin/v1/groups/${encodeURIComponent(groupId)}/members`, {
      method: 'POST',
      body: { user_id: userId },
    }),
  removeGroupMember: (groupId: string, userId: string) =>
    apiRequest<void>(
      `/api/admin/v1/groups/${encodeURIComponent(groupId)}/members/${encodeURIComponent(userId)}`,
      { method: 'DELETE' },
    ),
  listResourceScopes: (params?: { type?: string; limit?: number; offset?: number }) => {
    const suffix = queryString({ type: params?.type, limit: params?.limit, offset: params?.offset });
    return apiRequest<PagedResponse<ResourceScope>>(`/api/admin/v1/resource-scopes${suffix}`);
  },
  getResourceScope: (id: string) =>
    apiRequest<ResourceScope>(`/api/admin/v1/resource-scopes/${encodeURIComponent(id)}`),
  createResourceScope: (payload: { type: string; key: string; name: string }) =>
    apiRequest<ResourceScope>('/api/admin/v1/resource-scopes', { method: 'POST', body: payload }),
  updateResourceScope: (id: string, payload: { name?: string }) =>
    apiRequest<ResourceScope>(`/api/admin/v1/resource-scopes/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteResourceScope: (id: string) =>
    apiRequest<void>(`/api/admin/v1/resource-scopes/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  updatePassword: (payload: { current_password: string; new_password: string }) =>
    apiRequest<void>('/api/admin/v1/me/password', { method: 'POST', body: payload }),

  listUsers: (params?: { status?: string; limit?: number; offset?: number }) => {
    const suffix = queryString({
      status: params?.status,
      limit: params?.limit,
      offset: params?.offset,
    });
    return apiRequest<UserListResponse>(`/api/admin/v1/users${suffix}`);
  },
  getUser: (id: string) => apiRequest<User>(`/api/admin/v1/users/${encodeURIComponent(id)}`),
  updateUser: (id: string, payload: UpdateUserRequest) =>
    apiRequest<User>(`/api/admin/v1/users/${encodeURIComponent(id)}`, { method: 'PUT', body: payload }),
  disableUser: (id: string) =>
    apiRequest<User>(`/api/admin/v1/users/${encodeURIComponent(id)}/disable`, { method: 'POST' }),
  enableUser: (id: string) =>
    apiRequest<User>(`/api/admin/v1/users/${encodeURIComponent(id)}/enable`, { method: 'POST' }),
  listUserSessions: (userId: string, params?: { limit?: number }) => {
    const suffix = queryString({ limit: params?.limit });
    return apiRequest<UserSessionListResponse>(`/api/admin/v1/users/${encodeURIComponent(userId)}/sessions${suffix}`);
  },
  revokeSession: (sessionId: string) =>
    apiRequest<{ status: string }>(`/api/admin/v1/sessions/${encodeURIComponent(sessionId)}/revoke`, {
      method: 'POST',
    }),
  listUserBindings: (userId: string) =>
    apiRequest<AccountBinding[]>(`/api/admin/v1/users/${encodeURIComponent(userId)}/bindings`),
  createUserBinding: (
    userId: string,
    payload: {
      source_id: string;
      directory_user_id: string;
      provider_uid: string;
      provider_union_id?: string;
      is_primary: boolean;
    },
  ) =>
    apiRequest<AccountBinding>(`/api/admin/v1/users/${encodeURIComponent(userId)}/bindings`, {
      method: 'POST',
      body: payload,
    }),
  deleteUserBinding: (userId: string, bindingId: string) =>
    apiRequest<void>(
      `/api/admin/v1/users/${encodeURIComponent(userId)}/bindings/${encodeURIComponent(bindingId)}`,
      { method: 'DELETE' },
    ),
  getUserRoles: (id: string) => apiRequest<Role[]>(`/api/admin/v1/users/${encodeURIComponent(id)}/roles`),
  assignRoleToUser: (userId: string, roleId: string) =>
    apiRequest<void>(`/api/admin/v1/users/${encodeURIComponent(userId)}/roles`, {
      method: 'POST',
      body: { role_id: roleId },
    }),
  removeRoleFromUser: (userId: string, roleId: string) =>
    apiRequest<void>(
      `/api/admin/v1/users/${encodeURIComponent(userId)}/roles/${encodeURIComponent(roleId)}`,
      { method: 'DELETE' },
    ),

  listApplications: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<ApplicationListResponse>(`/api/admin/v1/applications${suffix}`);
  },
  getApplication: (id: string) => apiRequest<Application>(`/api/admin/v1/applications/${encodeURIComponent(id)}`),
  createApplication: (payload: { name: string; type: string }) =>
    apiRequest<Application>('/api/admin/v1/applications', { method: 'POST', body: payload }),
  updateApplication: (id: string, payload: { name?: string; status?: string }) =>
    apiRequest<Application>(`/api/admin/v1/applications/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteApplication: (id: string) =>
    apiRequest<void>(`/api/admin/v1/applications/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  listOIDCClients: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<{ clients: OIDCClient[]; total: number }>(`/api/admin/v1/oidc-clients${suffix}`);
  },
  getOIDCClient: (id: string) => apiRequest<OIDCClient>(`/api/admin/v1/oidc-clients/${encodeURIComponent(id)}`),
  createOIDCClient: (payload: {
    application_id: string;
    client_id: string;
    redirect_uris: string[];
    allowed_scopes?: string[];
    grant_types?: string[];
    response_types?: string[];
    pkce_required?: boolean;
  }) =>
    apiRequest<OIDCClientCreateResponse>('/api/admin/v1/oidc-clients', {
      method: 'POST',
      body: payload,
    }),
  updateOIDCClient: (
    id: string,
    payload: {
      redirect_uris?: string[];
      allowed_scopes?: string[];
      grant_types?: string[];
      response_types?: string[];
      pkce_required?: boolean;
      status?: string;
    },
  ) =>
    apiRequest<OIDCClient>(`/api/admin/v1/oidc-clients/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteOIDCClient: (id: string) =>
    apiRequest<void>(`/api/admin/v1/oidc-clients/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  rotateOIDCClientSecret: (id: string) =>
    apiRequest<OIDCClientCreateResponse>(`/api/admin/v1/oidc-clients/${encodeURIComponent(id)}/rotate-secret`, {
      method: 'POST',
    }),

  listLegacyUsers: (applicationId: string, params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<LegacyUsersListResponse>(
      `/api/admin/v1/applications/${encodeURIComponent(applicationId)}/legacy-users${suffix}`,
    );
  },
  getLegacyUser: (applicationId: string, username: string) =>
    apiRequest<LegacyAppUser>(
      `/api/admin/v1/applications/${encodeURIComponent(applicationId)}/legacy-users/${encodeURIComponent(username)}`,
    ),
  createLegacyUser: (
    applicationId: string,
    payload: {
      username: string;
      user_id: string;
      password: string;
      legacy_user_identifier?: string;
      is_active?: boolean;
    },
  ) =>
    apiRequest<LegacyAppUser>(
      `/api/admin/v1/applications/${encodeURIComponent(applicationId)}/legacy-users`,
      { method: 'POST', body: payload },
    ),
  updateLegacyUser: (
    applicationId: string,
    username: string,
    payload: {
      user_id?: string;
      password?: string;
      legacy_user_identifier?: string;
      is_active?: boolean;
    },
  ) =>
    apiRequest<LegacyAppUser>(
      `/api/admin/v1/applications/${encodeURIComponent(applicationId)}/legacy-users/${encodeURIComponent(username)}`,
      { method: 'PUT', body: payload },
    ),
  setLegacyUserStatus: (applicationId: string, username: string, isActive: boolean) => {
    const action = isActive ? 'enable' : 'disable';
    return apiRequest<LegacyAppUser>(
      `/api/admin/v1/applications/${encodeURIComponent(applicationId)}/legacy-users/${encodeURIComponent(username)}/${action}`,
      { method: 'POST' },
    );
  },
  deleteLegacyUser: (applicationId: string, username: string) =>
    apiRequest<void>(
      `/api/admin/v1/applications/${encodeURIComponent(applicationId)}/legacy-users/${encodeURIComponent(username)}`,
      { method: 'DELETE' },
    ),

  listIdentitySources: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<IdentitySourceListResponse>(`/api/admin/v1/identity-sources${suffix}`);
  },
  getIdentitySource: (id: string) =>
    apiRequest<IdentitySource>(`/api/admin/v1/identity-sources/${encodeURIComponent(id)}`),
  createIdentitySource: (payload: { type: string; name: string; sync_enabled?: boolean }) =>
    apiRequest<IdentitySource>('/api/admin/v1/identity-sources', { method: 'POST', body: payload }),
  updateIdentitySource: (
    id: string,
    payload: { name?: string; status?: string; sync_enabled?: boolean },
  ) =>
    apiRequest<IdentitySource>(`/api/admin/v1/identity-sources/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteIdentitySource: (id: string) =>
    apiRequest<void>(`/api/admin/v1/identity-sources/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  triggerSourceSync: (sourceId: string, mode: IdentitySourceSyncMode) =>
    apiRequest<void>(
      `/api/admin/v1/identity-sources/${encodeURIComponent(sourceId)}/sync/${mode}`,
      { method: 'POST' },
    ),

  listIMProviderConfigs: () => apiRequest<IMProviderConfig[]>('/api/admin/v1/integrations/im'),
  upsertIMProviderConfig: (provider: string, payload: Partial<IMProviderConfig>) =>
    apiRequest<IMProviderConfig>(`/api/admin/v1/integrations/im/${encodeURIComponent(provider)}`, {
      method: 'PUT',
      body: {
        provider,
        ...payload,
        config: payload.config || {},
      },
    }),
  listMCPConnectors: () => apiRequest<MCPConnector[]>('/api/admin/v1/mcp/connectors'),
  createMCPConnector: (payload: Omit<MCPConnector, 'id'>) =>
    apiRequest<MCPConnector>('/api/admin/v1/mcp/connectors', {
      method: 'POST',
      body: payload,
    }),

  listRoles: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<{ items: Role[]; total: number }>(`/api/admin/v1/roles${suffix}`);
  },
  getRole: (id: string) => apiRequest<Role>(`/api/admin/v1/roles/${encodeURIComponent(id)}`),
  createRole: (payload: { name: string; code: string; description?: string }) =>
    apiRequest<Role>('/api/admin/v1/roles', { method: 'POST', body: payload }),
  updateRole: (id: string, payload: { name?: string; description?: string }) =>
    apiRequest<Role>(`/api/admin/v1/roles/${encodeURIComponent(id)}`, { method: 'PUT', body: payload }),
  deleteRole: (id: string) =>
    apiRequest<void>(`/api/admin/v1/roles/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  listPermissions: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<{ items: Permission[]; total: number }>(`/api/admin/v1/permissions${suffix}`);
  },
  getPermission: (id: string) => apiRequest<Permission>(`/api/admin/v1/permissions/${encodeURIComponent(id)}`),
  createPermission: (payload: { code: string; name: string; type: string }) =>
    apiRequest<Permission>('/api/admin/v1/permissions', { method: 'POST', body: payload }),
  updatePermission: (id: string, payload: { name: string }) =>
    apiRequest<Permission>(`/api/admin/v1/permissions/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deletePermission: (id: string) =>
    apiRequest<void>(`/api/admin/v1/permissions/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  checkPermission: (payload: { user_id: string; permission: string }) =>
    apiRequest<PermissionCheckResponse>('/api/admin/v1/permissions/check', {
      method: 'POST',
      body: payload,
    }),
  assignPermissionToRole: (roleId: string, permissionId: string) =>
    apiRequest<void>(`/api/admin/v1/roles/${encodeURIComponent(roleId)}/permissions`, {
      method: 'POST',
      body: { permission_id: permissionId },
    }),
  removePermissionFromRole: (roleId: string, permissionId: string) =>
    apiRequest<void>(
      `/api/admin/v1/roles/${encodeURIComponent(roleId)}/permissions/${encodeURIComponent(permissionId)}`,
      { method: 'DELETE' },
    ),
  listRolePermissions: (roleId: string) =>
    apiRequest<Permission[]>(`/api/admin/v1/roles/${encodeURIComponent(roleId)}/permissions`),
  assignResourceScopeToRole: (roleId: string, resourceScopeId: string, effect: 'allow' | 'deny') =>
    apiRequest<{ status: string }>('/api/admin/v1/roles/scopes', {
      method: 'POST',
      body: { role_id: roleId, resource_scope_id: resourceScopeId, effect },
    }),
  removeResourceScopeFromRole: (roleId: string, resourceScopeId: string) =>
    apiRequest<void>(
      `/api/admin/v1/roles/${encodeURIComponent(roleId)}/scopes/${encodeURIComponent(resourceScopeId)}`,
      { method: 'DELETE' },
    ),

  listAssignments: (appId: string, params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<AssignmentListResponse>(
      `/api/admin/v1/applications/${encodeURIComponent(appId)}/assignments${suffix}`,
    );
  },
  createAssignment: (
    appId: string,
    payload: { subject_type: string; subject_id: string; effect: string },
  ) =>
    apiRequest<ApplicationAssignment>(
      `/api/admin/v1/applications/${encodeURIComponent(appId)}/assignments`,
      { method: 'POST', body: payload },
    ),
  deleteAssignment: (assignmentId: string) =>
    apiRequest<void>(`/api/admin/v1/applications/assignments/${encodeURIComponent(assignmentId)}`, {
      method: 'DELETE',
    }),
};
