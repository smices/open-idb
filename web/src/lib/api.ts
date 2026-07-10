// SPDX-License-Identifier: MIT

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

const API_TARGET = import.meta.env.VITE_API_TARGET || import.meta.env.PUBLIC_API_TARGET || '';

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

export interface AdminCurrentUser {
  id: string;
  admin_id: string;
  entity_id?: string;
  username: string;
  display_name: string;
  role: 'platform_admin' | 'enterprise_admin' | string;
}

export interface AdminUser {
  id: string;
  entity_id?: string;
  entity_name?: string;
  username: string;
  display_name: string;
  email?: string;
  status: string;
  role: 'platform_admin' | 'enterprise_admin' | string;
  protected: boolean;
  created_at: string;
  updated_at: string;
}

export interface AdminRoleOption {
  value: 'platform_admin' | 'enterprise_admin' | string;
  label: string;
  description: string;
  requires_entity: boolean;
}

export interface AdminUserListResponse {
  items: AdminUser[];
  total: number;
}

export interface PlatformBranding {
  platform_name: string;
  logo_url: string;
  favicon_url: string;
  title_suffix: string;
  updated_at?: string;
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
  admin_users: number;
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

export type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue };

export type ArchivedUser = {
  id: string;
  entity_id: string;
  original_user_id: string;
  username: string;
  display_name: string;
  email: string;
  phone: string;
  user_type: string;
  archived_at: string;
  archived_by_user_id: string;
  archive_reason: string;
  user_snapshot: JsonValue;
  bindings_snapshot: JsonValue;
  roles_snapshot: JsonValue;
};

export type ArchivedUserListResponse = {
  items: ArchivedUser[];
  total: number;
  limit: number;
  offset: number;
};

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
  english_name?: string;
  employee_no?: string;
  job_title?: string;
  email?: string;
  phone?: string;
  avatar_url?: string;
  status: string;
  raw_profile?: unknown;
  last_synced_at: string;
  created_at: string;
  updated_at: string;
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

export type OrganizationTreeNodeKind = 'company' | 'organization' | 'department' | 'user';

export interface OrganizationTreeNode {
  id: string;
  kind: OrganizationTreeNodeKind;
  name: string;
  parent_id?: string;
  organization_id?: string;
  source_id?: string;
  external_department_id?: string;
  english_name?: string;
  employee_no?: string;
  job_title?: string;
  email?: string;
  phone?: string;
  status?: string;
  has_children: boolean;
  updated_at?: string;
}

export interface OrganizationTreeRootResponse {
  root: OrganizationTreeNode;
  children: OrganizationTreeNode[];
  limit: number;
  offset: number;
}

export interface OrganizationTreeSearchResponse {
  items: OrganizationTreeNode[];
  total: number;
  limit: number;
  offset: number;
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
  english_name?: string;
  employee_no?: string;
  job_title?: string;
  email?: string;
  lifecycle_status: string;
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

export type ApplicationType = 'oidc_client' | 'api_client' | 'internal_app';

export interface Application {
  id: string;
  entity_id: string;
  name: string;
  type: ApplicationType;
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

export interface ApplicationRoleAssignment {
  id: string;
  entity_id: string;
  application_id: string;
  role_id: string;
  role_code: string;
  role_name: string;
  effect: string;
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
  workplace_provider?: string;
  workplace_app_id?: string;
  workplace_app_secret?: string;
  status?: string;
  created_at?: string;
  updated_at?: string;
}

export interface OIDCClientCreateResponse {
  client: OIDCClient;
  client_secret: string;
}

export interface FeishuIdentitySourceConfig {
  id?: string;
  provider: string;
  display_name: string;
  status: string;
  oauth_configured: boolean;
  sync_enabled: boolean;
  redirect_uri?: string;
  config: Record<string, string>;
}

export interface LoginProvider {
  provider: string;
  display_name: string;
  oauth_url?: string;
  app_id?: string;
  workplace_exchange_url?: string;
}

export interface FeishuExchangeResponse {
  session: string;
  entity_id: string;
  user_id: string;
  username: string;
  display_name: string;
}

export type LoginMode = 'app' | 'user' | 'admin' | 'entity_admin';

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

export interface UserApplicationAccess {
  application_id: string;
  application_name: string;
  application_type: string;
  has_access: boolean;
  roles: Array<{
    role_id: string;
    role_code: string;
    permissions: string[];
    resource_scopes: Array<{ type: string; key: string; effect: string }>;
  }>;
}

export interface UserAccessSummary {
  user_id: string;
  entity_id: string;
  lifecycle_status: string;
  applications: UserApplicationAccess[];
}

export const api = {
  getPlatformBranding: (): Promise<PlatformBranding> => apiRequest<PlatformBranding>('/api/platform/branding'),
  getAdminPlatformBranding: (): Promise<PlatformBranding> => apiRequest<PlatformBranding>('/sapi/platform/branding'),
  updatePlatformBranding: (payload: { platform_name: string; logo_url?: string; favicon_url?: string; title_suffix?: string }) =>
    apiRequest<PlatformBranding>('/sapi/platform/branding', { method: 'PUT', body: payload }),
  me: (): Promise<CurrentUser> => apiRequest<CurrentUser>('/api/me'),
  adminMe: (): Promise<AdminCurrentUser> => apiRequest<AdminCurrentUser>('/sapi/me'),
  updateMe: (payload: { display_name: string }): Promise<CurrentUser> =>
    apiRequest<CurrentUser>('/api/me', { method: 'PATCH', body: payload }),
  updateAdminMe: (payload: { display_name: string }): Promise<AdminCurrentUser> =>
    apiRequest<AdminCurrentUser>('/sapi/me', { method: 'PATCH', body: payload }),
  myAccess: (): Promise<UserAccessSummary> => apiRequest<UserAccessSummary>('/api/me/access'),
  listEntities: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<EntityListResponse>(`/sapi/entities${suffix}`);
  },
  getEntity: (id: string) => apiRequest<Entity>(`/sapi/entities/${encodeURIComponent(id)}`),
  createEntity: (payload: { name: string; slug: string; default_locale?: string; brand_name?: string; logo_url?: string; login_message?: string }) =>
    apiRequest<Entity>('/sapi/entities', { method: 'POST', body: payload }),
  updateEntity: (id: string, payload: { name?: string; status?: string; default_locale?: string; brand_name?: string; logo_url?: string; login_message?: string }) =>
    apiRequest<Entity>(`/sapi/entities/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  listLoginProviders: (entityId?: string, clientId?: string) => {
    const suffix = queryString({ entity_id: entityId, client_id: clientId });
    return apiRequest<LoginProvider[]>(`/api/auth/providers${suffix}`);
  },
  exchangeFeishuAppCode: (payload: { auth_code: string; entity_id: string; client_id?: string }) =>
    apiRequest<FeishuExchangeResponse>('/api/auth/feishu/exchange', {
      method: 'POST',
      body: payload,
    }),
  getLoginContext: (params?: { path?: string; return_to?: string }) => {
    const suffix = queryString({ path: params?.path, return_to: params?.return_to });
    return apiRequest<LoginContext>(`/api/auth/context${suffix}`);
  },
  getAdminLoginContext: (params?: { path?: string; return_to?: string }) => {
    const suffix = queryString({ path: params?.path, return_to: params?.return_to });
    return apiRequest<LoginContext>(`/sapi/auth/context${suffix}`);
  },
  dashboardSummary: (): Promise<DashboardSummary> =>
    apiRequest<DashboardSummary>('/sapi/dashboard/summary'),
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
    return apiRequest<AuditLogListResponse>(`/sapi/audit-logs${suffix}`);
  },
  listSyncJobs: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<SyncJobListResponse>(`/sapi/sync-jobs${suffix}`);
  },
  getDirectoryUser: (id: string) =>
    apiRequest<DirectoryUser>(`/sapi/directory-users/${encodeURIComponent(id)}`),
  getOrganizationTreeRoot: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<OrganizationTreeRootResponse>(`/sapi/organization-tree/root${suffix}`);
  },
  listOrganizationTreeChildren: (params: {
    kind: OrganizationTreeNodeKind;
    id: string;
    limit?: number;
    offset?: number;
  }) => {
    const suffix = queryString({
      kind: params.kind,
      id: params.id,
      limit: params.limit,
      offset: params.offset,
    });
    return apiRequest<PagedResponse<OrganizationTreeNode>>(`/sapi/organization-tree/children${suffix}`);
  },
  searchOrganizationTree: (params: { q: string; limit?: number; offset?: number }) => {
    const suffix = queryString({
      q: params.q,
      limit: params.limit,
      offset: params.offset,
    });
    return apiRequest<OrganizationTreeSearchResponse>(`/sapi/organization-tree/search${suffix}`);
  },
  updatePassword: (payload: { current_password: string; new_password: string }) =>
    apiRequest<void>('/api/me/password', { method: 'POST', body: payload }),
  updateAdminPassword: (payload: { current_password: string; new_password: string }) =>
    apiRequest<void>('/sapi/me/password', { method: 'POST', body: payload }),
  listAdminUsers: () => apiRequest<AdminUserListResponse>('/sapi/admin-users'),
  listAdminRoles: () => apiRequest<AdminRoleOption[]>('/sapi/admin-users/roles'),
  createAdminUser: (payload: {
    username: string;
    display_name: string;
    email?: string;
    role: string;
    entity_id?: string;
    password: string;
  }) => apiRequest<AdminUser>('/sapi/admin-users', { method: 'POST', body: payload }),
  updateAdminUser: (
    id: string,
    payload: {
      display_name: string;
      email?: string;
      role: string;
      entity_id?: string;
      status: string;
    },
  ) => apiRequest<AdminUser>(`/sapi/admin-users/${encodeURIComponent(id)}`, { method: 'PUT', body: payload }),
  deleteAdminUser: (id: string) =>
    apiRequest<void>(`/sapi/admin-users/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  setAdminUserPassword: (id: string, password: string) =>
    apiRequest<void>(`/sapi/admin-users/${encodeURIComponent(id)}/password`, {
      method: 'POST',
      body: { password },
    }),

  listUsers: (params?: { status?: string; limit?: number; offset?: number }) => {
    const suffix = queryString({
      status: params?.status,
      limit: params?.limit,
      offset: params?.offset,
    });
    return apiRequest<UserListResponse>(`/sapi/users${suffix}`);
  },
  listArchivedUsers: (params?: { username?: string; limit?: number; offset?: number }) => {
    const suffix = queryString({
      username: params?.username,
      limit: params?.limit,
      offset: params?.offset,
    });
    return apiRequest<ArchivedUserListResponse>(`/sapi/archived-users${suffix}`);
  },
  getArchivedUser: (id: string) =>
    apiRequest<ArchivedUser>(`/sapi/archived-users/${encodeURIComponent(id)}`),
  getUser: (id: string) => apiRequest<User>(`/sapi/users/${encodeURIComponent(id)}`),
  updateUser: (id: string, payload: UpdateUserRequest) =>
    apiRequest<User>(`/sapi/users/${encodeURIComponent(id)}`, { method: 'PUT', body: payload }),
  disableUser: (id: string) =>
    apiRequest<User>(`/sapi/users/${encodeURIComponent(id)}/disable`, { method: 'POST' }),
  enableUser: (id: string) =>
    apiRequest<User>(`/sapi/users/${encodeURIComponent(id)}/enable`, { method: 'POST' }),
  listUserSessions: (userId: string, params?: { limit?: number }) => {
    const suffix = queryString({ limit: params?.limit });
    return apiRequest<UserSessionListResponse>(`/sapi/users/${encodeURIComponent(userId)}/sessions${suffix}`);
  },
  revokeSession: (sessionId: string) =>
    apiRequest<{ status: string }>(`/sapi/sessions/${encodeURIComponent(sessionId)}/revoke`, {
      method: 'POST',
    }),
  listUserBindings: (userId: string) =>
    apiRequest<AccountBinding[]>(`/sapi/users/${encodeURIComponent(userId)}/bindings`),
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
    apiRequest<AccountBinding>(`/sapi/users/${encodeURIComponent(userId)}/bindings`, {
      method: 'POST',
      body: payload,
    }),
  deleteUserBinding: (userId: string, bindingId: string) =>
    apiRequest<void>(
      `/sapi/users/${encodeURIComponent(userId)}/bindings/${encodeURIComponent(bindingId)}`,
      { method: 'DELETE' },
    ),
  getUserRoles: (id: string) => apiRequest<Role[]>(`/sapi/users/${encodeURIComponent(id)}/roles`),
  assignRoleToUser: (userId: string, roleId: string) =>
    apiRequest<void>(`/sapi/users/${encodeURIComponent(userId)}/roles`, {
      method: 'POST',
      body: { role_id: roleId },
    }),
  removeRoleFromUser: (userId: string, roleId: string) =>
    apiRequest<void>(
      `/sapi/users/${encodeURIComponent(userId)}/roles/${encodeURIComponent(roleId)}`,
      { method: 'DELETE' },
    ),

  listApplications: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<ApplicationListResponse>(`/sapi/applications${suffix}`);
  },
  getApplication: (id: string) => apiRequest<Application>(`/sapi/applications/${encodeURIComponent(id)}`),
  createApplication: (payload: { name: string; type: ApplicationType }) =>
    apiRequest<Application>('/sapi/applications', { method: 'POST', body: payload }),
  updateApplication: (id: string, payload: { name?: string; status?: string }) =>
    apiRequest<Application>(`/sapi/applications/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteApplication: (id: string) =>
    apiRequest<void>(`/sapi/applications/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  listApplicationRoleAssignments: (id: string) =>
    apiRequest<{ items: ApplicationRoleAssignment[]; roles: ApplicationRoleAssignment[] }>(
      `/sapi/applications/${encodeURIComponent(id)}/role-assignments`,
    ),
  setApplicationRoleAssignments: (id: string, roleIds: string[]) =>
    apiRequest<{ items: ApplicationRoleAssignment[]; roles: ApplicationRoleAssignment[] }>(
      `/sapi/applications/${encodeURIComponent(id)}/role-assignments`,
      {
        method: 'PUT',
        body: { role_ids: roleIds },
      },
    ),

  listOIDCClients: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<{ clients: OIDCClient[]; total: number }>(`/sapi/oidc-clients${suffix}`);
  },
  getOIDCClient: (id: string) => apiRequest<OIDCClient>(`/sapi/oidc-clients/${encodeURIComponent(id)}`),
  createOIDCClient: (payload: {
    application_id: string;
    client_id?: string;
    redirect_uris: string[];
    allowed_scopes?: string[];
    grant_types?: string[];
    response_types?: string[];
    pkce_required?: boolean;
    workplace_provider?: string;
    workplace_app_id?: string;
    workplace_app_secret?: string;
  }) =>
    apiRequest<OIDCClientCreateResponse>('/sapi/oidc-clients', {
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
      workplace_provider?: string;
      workplace_app_id?: string;
      workplace_app_secret?: string;
      status?: string;
    },
  ) =>
    apiRequest<OIDCClient>(`/sapi/oidc-clients/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteOIDCClient: (id: string) =>
    apiRequest<void>(`/sapi/oidc-clients/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  rotateOIDCClientSecret: (id: string) =>
    apiRequest<OIDCClientCreateResponse>(`/sapi/oidc-clients/${encodeURIComponent(id)}/rotate-secret`, {
      method: 'POST',
    }),

  listIdentitySources: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<IdentitySourceListResponse>(`/sapi/identity-sources${suffix}`);
  },
  getIdentitySource: (id: string) =>
    apiRequest<IdentitySource>(`/sapi/identity-sources/${encodeURIComponent(id)}`),
  createIdentitySource: (payload: { type: string; name: string; sync_enabled?: boolean }) =>
    apiRequest<IdentitySource>('/sapi/identity-sources', { method: 'POST', body: payload }),
  updateIdentitySource: (
    id: string,
    payload: { name?: string; status?: string; sync_enabled?: boolean },
  ) =>
    apiRequest<IdentitySource>(`/sapi/identity-sources/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deleteIdentitySource: (id: string) =>
    apiRequest<void>(`/sapi/identity-sources/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  triggerSourceSync: (sourceId: string, mode: IdentitySourceSyncMode) =>
    apiRequest<void>(
      `/sapi/identity-sources/${encodeURIComponent(sourceId)}/sync/${mode}`,
      { method: 'POST' },
    ),

  getFeishuIdentitySourceConfig: () =>
    apiRequest<FeishuIdentitySourceConfig>('/sapi/identity-sources/feishu/config'),
  upsertFeishuIdentitySourceConfig: (payload: Partial<FeishuIdentitySourceConfig>) =>
    apiRequest<FeishuIdentitySourceConfig>('/sapi/identity-sources/feishu/config', {
      method: 'PUT',
      body: {
        provider: 'feishu',
        ...payload,
        config: payload.config || {},
      },
    }),
  listRoles: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<{ items: Role[]; total: number }>(`/sapi/roles${suffix}`);
  },
  getRole: (id: string) => apiRequest<Role>(`/sapi/roles/${encodeURIComponent(id)}`),
  createRole: (payload: { name: string; code: string; description?: string }) =>
    apiRequest<Role>('/sapi/roles', { method: 'POST', body: payload }),
  updateRole: (id: string, payload: { name?: string; description?: string }) =>
    apiRequest<Role>(`/sapi/roles/${encodeURIComponent(id)}`, { method: 'PUT', body: payload }),
  deleteRole: (id: string) =>
    apiRequest<void>(`/sapi/roles/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  listPermissions: (params?: { limit?: number; offset?: number }) => {
    const suffix = queryString({ limit: params?.limit, offset: params?.offset });
    return apiRequest<{ items: Permission[]; total: number }>(`/sapi/permissions${suffix}`);
  },
  getPermission: (id: string) => apiRequest<Permission>(`/sapi/permissions/${encodeURIComponent(id)}`),
  createPermission: (payload: { code: string; name: string; type: string }) =>
    apiRequest<Permission>('/sapi/permissions', { method: 'POST', body: payload }),
  updatePermission: (id: string, payload: { name: string }) =>
    apiRequest<Permission>(`/sapi/permissions/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    }),
  deletePermission: (id: string) =>
    apiRequest<void>(`/sapi/permissions/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  checkPermission: (payload: { user_id: string; permission: string }) =>
    apiRequest<PermissionCheckResponse>('/sapi/permissions/check', {
      method: 'POST',
      body: payload,
    }),
  assignPermissionToRole: (roleId: string, permissionId: string) =>
    apiRequest<void>(`/sapi/roles/${encodeURIComponent(roleId)}/permissions`, {
      method: 'POST',
      body: { permission_id: permissionId },
    }),
  removePermissionFromRole: (roleId: string, permissionId: string) =>
    apiRequest<void>(
      `/sapi/roles/${encodeURIComponent(roleId)}/permissions/${encodeURIComponent(permissionId)}`,
      { method: 'DELETE' },
    ),
  listRolePermissions: (roleId: string) =>
    apiRequest<Permission[]>(`/sapi/roles/${encodeURIComponent(roleId)}/permissions`),
};
