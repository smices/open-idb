<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { t } from '$lib/i18n';
  import { api, type User, type Role, type Application, type ApplicationAssignment, type UserSession, type AccountBinding } from '$lib/api';

  let userId = '';
  let user: User | null = null;

  let activeTab: 'info' | 'roles' | 'apps' | 'sessions' | 'bindings' = 'info';
  let roleSearch = '';
  let appAccessSearch = '';
  let sessionSearch = '';
  let bindingSearch = '';
  let loading = true;
  let userRoles: Role[] = [];
  let allRoles: Role[] = [];
  let availableRoles: Role[] = [];
  let roleLoading = false;

  let apps: Application[] = [];
  let assignments: ApplicationAssignment[] = [];
  let appsLoading = false;

  let sessions: UserSession[] = [];
  let sessionsLoading = false;
  let pendingSessionRevokeId = '';

  let bindings: AccountBinding[] = [];
  let bindingsLoading = false;
  let bindingSaving = false;
  let pendingBindingDeleteId = '';
  let bindingSourceId = '';
  let bindingDirectoryUserId = '';
  let bindingProviderUid = '';
  let bindingProviderUnionId = '';
  let bindingIsPrimary = false;

  let selectedRoleId = '';
  let pendingRoleDeleteId = '';

  let error = '';
  let message = '';

  $: userId = $page.params.id ?? '';

  const normalizeLocale = () => {
    return user?.locale || 'en-US';
  };

  const localeLabel = (value: string): string => (value === 'zh-CN' ? t('layout.chinese') : t('layout.english'));
  const userStatusLabel = (value: string): string => t(`users.status.${value}`, value);
  const userTypeLabel = (value: string): string => t(`users.type.${value}`, value);
  const sessionStatusLabel = (value: string): string => t(`users.sessionStatus.${value}`, value);
  const assignmentEffectLabel = (value: string): string => t(`assignments.${value}`, value);
  const formatDateTime = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');

  const refreshUser = async () => {
    if (!userId) return;
    loading = true;
    error = '';

    try {
      user = await api.getUser(userId);
    } catch {
      error = t('users.fetchFailed');
      user = null;
    } finally {
      loading = false;
    }
  };

  const loadRoles = async () => {
    if (!user) return;
    roleLoading = true;
    try {
      const [myRoles, all] = await Promise.all([api.getUserRoles(user.id), api.listRoles({ limit: 200 })]);
      userRoles = myRoles || [];
      allRoles = all.items || [];
    } finally {
      roleLoading = false;
    }
  };

  const loadAppAccess = async () => {
    if (!user) return;
    appsLoading = true;
    try {
      const appData = await api.listApplications({ limit: 200 });
      apps = appData.applications || [];
      const perAppAssignments = await Promise.all(
        apps.map(async (appItem) => {
          try {
            const result = await api.listAssignments(appItem.id, { limit: 100 });
            return (result.assignments || []).filter(
              (item: ApplicationAssignment) => item.subject_type === 'user' && item.subject_id === user?.id,
            );
          } catch {
            return [] as ApplicationAssignment[];
          }
        }),
      );

      assignments = perAppAssignments.flat();
    } finally {
      appsLoading = false;
    }
  };

  const loadSessions = async () => {
    if (!user) return;
    sessionsLoading = true;
    try {
      const data = await api.listUserSessions(user.id, { limit: 100 });
      sessions = data.items || [];
    } finally {
      sessionsLoading = false;
    }
  };

  const loadBindings = async () => {
    if (!user) return;
    bindingsLoading = true;
    try {
      bindings = await api.listUserBindings(user.id);
    } finally {
      bindingsLoading = false;
    }
  };

  const changeStatus = async () => {
    if (!user) return;
    error = '';
    message = '';
    try {
      if (user.lifecycle_status === 'active') {
        user = await api.disableUser(user.id);
        message = t('users.disableSuccess');
      } else {
        user = await api.enableUser(user.id);
        message = t('users.enableSuccess');
      }
      await refreshUser();
    } catch {
      error = t('users.saveFailed');
    }
  };

  const onAssignRole = async () => {
    if (!user || !selectedRoleId) return;
    error = '';
    message = '';

    try {
      await api.assignRoleToUser(user.id, selectedRoleId);
      selectedRoleId = '';
      message = t('users.roleAssigned');
      await loadRoles();
    } catch {
      error = t('users.saveFailed');
    }
  };

  const onRemoveRole = async (roleId: string) => {
    if (!user) return;
    error = '';
    message = '';
    try {
      await api.removeRoleFromUser(user.id, roleId);
      pendingRoleDeleteId = '';
      message = t('users.roleRemoved');
      await loadRoles();
    } catch {
      error = t('users.saveFailed');
    }
  };

  const revokeSession = async (session: UserSession) => {
    if (pendingSessionRevokeId !== session.id) {
      pendingSessionRevokeId = session.id;
      return;
    }
    error = '';
    message = '';
    try {
      await api.revokeSession(session.id);
      pendingSessionRevokeId = '';
      message = t('users.sessionRevoked');
      await loadSessions();
    } catch {
      error = t('users.sessionRevokeFailed');
    }
  };

  const createBinding = async () => {
    if (!user) return;
    bindingSaving = true;
    error = '';
    message = '';
    try {
      await api.createUserBinding(user.id, {
        source_id: bindingSourceId,
        directory_user_id: bindingDirectoryUserId,
        provider_uid: bindingProviderUid,
        provider_union_id: bindingProviderUnionId || undefined,
        is_primary: bindingIsPrimary,
      });
      bindingSourceId = '';
      bindingDirectoryUserId = '';
      bindingProviderUid = '';
      bindingProviderUnionId = '';
      bindingIsPrimary = false;
      message = t('users.bindingCreated');
      await loadBindings();
    } catch {
      error = t('users.bindingCreateFailed');
    } finally {
      bindingSaving = false;
    }
  };

  const deleteBinding = async (binding: AccountBinding) => {
    if (!user) return;
    if (pendingBindingDeleteId !== binding.id) {
      pendingBindingDeleteId = binding.id;
      return;
    }
    error = '';
    message = '';
    try {
      await api.deleteUserBinding(user.id, binding.id);
      pendingBindingDeleteId = '';
      message = t('users.bindingDeleted');
      await loadBindings();
    } catch {
      error = t('users.bindingDeleteFailed');
    }
  };

  const appName = (id: string): string => {
    const target = apps.find((item) => item.id === id);
    return target ? target.name : id;
  };

  const includesQuery = (value: unknown, query: string): boolean => String(value ?? '').toLowerCase().includes(query.trim().toLowerCase());

  const matchesRole = (role: Role, query: string): boolean => {
    if (!query.trim()) return true;
    return [role.id, role.name, role.code, role.description].some((value) => includesQuery(value, query));
  };

  const matchesAssignment = (assignment: ApplicationAssignment, query: string): boolean => {
    if (!query.trim()) return true;
    return [assignment.id, assignment.application_id, appName(assignment.application_id), assignment.effect].some((value) =>
      includesQuery(value, query),
    );
  };

  const matchesSession = (session: UserSession, query: string): boolean => {
    if (!query.trim()) return true;
    return [session.id, session.device_id, session.ip, session.user_agent, session.login_method, session.status].some((value) =>
      includesQuery(value, query),
    );
  };

  const matchesBinding = (binding: AccountBinding, query: string): boolean => {
    if (!query.trim()) return true;
    return [
      binding.id,
      binding.source_name,
      binding.source_type,
      binding.source_id,
      binding.directory_user_id,
      binding.provider_uid,
      binding.provider_union_id,
    ].some((value) => includesQuery(value, query));
  };

  $: availableRoles = allRoles.filter((item) => !userRoles.find((r) => r.id === item.id));
  $: filteredUserRoles = userRoles.filter((role) => matchesRole(role, roleSearch));
  $: filteredAssignments = assignments.filter((assignment) => matchesAssignment(assignment, appAccessSearch));
  $: filteredSessions = sessions.filter((session) => matchesSession(session, sessionSearch));
  $: filteredBindings = bindings.filter((binding) => matchesBinding(binding, bindingSearch));
  $: activeSessionCount = sessions.filter((session) => session.status === 'active').length;
  $: primaryBindingCount = bindings.filter((binding) => binding.is_primary).length;

  onMount(() => {
    void refreshUser();
  });
</script>

<svelte:head>
  <title>{user ? `${user.display_name || user.username} - ${t('users.info')}` : t('users.info')}</title>
</svelte:head>

{#if loading}
  <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
{:else if !user}
  <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.fetchFailed')}</div>
{:else}
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-2xl font-semibold">{user.display_name || user.username}</h1>
      <div class="flex flex-wrap gap-2">
        <a class="btn btn-sm preset-outlined-surface-500 inline-flex" href="/users">{t('common.back')}</a>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void changeStatus()}>
          {user.lifecycle_status === 'active' ? t('users.disable') : t('users.enable')}
        </button>
      </div>
    </header>

    {#if error}
      <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
    {/if}
    {#if message}
      <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
    {/if}

    <div class="flex flex-wrap gap-2" aria-label={t('users.info')}>
      <button
        class={`btn btn-sm ${activeTab === 'info' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`}
        type="button"
        aria-pressed={activeTab === 'info'}
        on:click={() => (activeTab = 'info')}
      >
        {t('users.info')}
      </button>
      <button
        class={`btn btn-sm ${activeTab === 'roles' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`}
        type="button"
        aria-pressed={activeTab === 'roles'}
        on:click={() => {
          activeTab = 'roles';
          void loadRoles();
        }}
      >
        {t('users.roles')}
      </button>
      <button
        class={`btn btn-sm ${activeTab === 'apps' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`}
        type="button"
        aria-pressed={activeTab === 'apps'}
        on:click={() => {
          activeTab = 'apps';
          void loadAppAccess();
        }}
      >
        {t('users.appAccess')}
      </button>
      <button
        class={`btn btn-sm ${activeTab === 'sessions' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`}
        type="button"
        aria-pressed={activeTab === 'sessions'}
        on:click={() => {
          activeTab = 'sessions';
          void loadSessions();
        }}
      >
        {t('users.sessions')}
      </button>
      <button
        class={`btn btn-sm ${activeTab === 'bindings' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`}
        type="button"
        aria-pressed={activeTab === 'bindings'}
        on:click={() => {
          activeTab = 'bindings';
          void loadBindings();
        }}
      >
        {t('users.bindings')}
      </button>
    </div>

    {#if activeTab === 'info'}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-2">
        <dl class="grid gap-3 text-sm md:grid-cols-2">
          <div>
            <dt class="text-surface-500">{t('users.username')}</dt>
            <dd class="font-medium">{user.username}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.displayName')}</dt>
            <dd class="font-medium">{user.display_name || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.email')}</dt>
            <dd class="font-medium">{user.email || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.phone')}</dt>
            <dd class="font-medium">{user.phone || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.type')}</dt>
            <dd class="font-medium">{userTypeLabel(user.user_type)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.status')}</dt>
            <dd class="font-medium">{userStatusLabel(user.lifecycle_status)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.createdAt')}</dt>
            <dd class="font-medium">{user.created_at ? new Date(user.created_at).toLocaleString() : '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.locale')}</dt>
            <dd class="font-medium">{localeLabel(normalizeLocale())}</dd>
          </div>
        </dl>
      </div>
    {:else if activeTab === 'roles'}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-4">
        <form class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto]" on:submit|preventDefault={onAssignRole}>
          <select class="input w-full" aria-label={t('users.assignRole')} bind:value={selectedRoleId}>
            <option value="">{t('users.assignRole')}</option>
            {#each availableRoles as role}
              <option value={role.id}>{role.name}</option>
            {/each}
          </select>
          <button class="btn preset-filled-primary-500" type="submit">{t('roles.assign')}</button>
        </form>

        {#if roleLoading}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else}
          {#if userRoles.length === 0}
            <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
          {:else}
            <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-center">
              <label class="block">
                <span class="sr-only">{t('users.searchRoles')}</span>
                <input class="input w-full" type="search" placeholder={t('users.searchRoles')} bind:value={roleSearch} />
              </label>
              <div class="card bg-surface-50-950 border border-surface-200-800 px-3 py-2 text-sm">
                <div class="text-xs text-surface-500">{t('users.visibleRows')}</div>
                <div class="font-semibold">{filteredUserRoles.length} / {userRoles.length}</div>
              </div>
              <div class="card bg-surface-50-950 border border-surface-200-800 px-3 py-2 text-sm">
                <div class="text-xs text-surface-500">{t('users.pageRange')}</div>
                <div class="font-semibold">{userRoles.length}</div>
              </div>
            </div>
            <div class="divide-y divide-surface-200-800">
              {#each filteredUserRoles as role (role.id)}
                <article class="py-3">
                  <div class="flex justify-between items-center">
                    <span>{role.name}</span>
                    <button
                      class="btn preset-tonal-error btn-xs"
                      type="button"
                      on:click={() => (pendingRoleDeleteId === role.id ? void onRemoveRole(role.id) : (pendingRoleDeleteId = role.id))}
                    >
                      {pendingRoleDeleteId === role.id ? t('common.confirmDelete') : t('common.delete')}
                    </button>
                  </div>
                </article>
              {/each}
            </div>
            {#if filteredUserRoles.length === 0}
              <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
            {/if}
          {/if}
        {/if}
      </div>
    {:else if activeTab === 'apps'}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3">
        {#if appsLoading}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else}
          {#if assignments.length === 0}
            <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
          {:else}
            <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-center">
              <label class="block">
                <span class="sr-only">{t('users.searchAppAccess')}</span>
                <input class="input w-full" type="search" placeholder={t('users.searchAppAccess')} bind:value={appAccessSearch} />
              </label>
              <div class="card bg-surface-50-950 border border-surface-200-800 px-3 py-2 text-sm">
                <div class="text-xs text-surface-500">{t('users.visibleRows')}</div>
                <div class="font-semibold">{filteredAssignments.length} / {assignments.length}</div>
              </div>
              <div class="card bg-surface-50-950 border border-surface-200-800 px-3 py-2 text-sm">
                <div class="text-xs text-surface-500">{t('applications.activeApps')}</div>
                <div class="font-semibold">{apps.filter((app) => app.status === 'active').length}</div>
              </div>
            </div>
            <div class="divide-y divide-surface-200-800">
              {#each filteredAssignments as assignment (assignment.id)}
                <article class="py-3">
                  <div class="flex justify-between">
                    <span>{appName(assignment.application_id)}</span>
                    <span class="text-sm">{assignmentEffectLabel(assignment.effect)}</span>
                  </div>
                </article>
              {/each}
            </div>
            {#if filteredAssignments.length === 0}
              <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
            {/if}
          {/if}
        {/if}
      </div>
    {:else if activeTab === 'sessions'}
      <div class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
        {#if sessionsLoading}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else if sessions.length === 0}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.noSessions')}</div>
        {:else}
          <div class="grid gap-3 border-b border-surface-200-800 p-4 md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-center">
            <label class="block">
              <span class="sr-only">{t('users.searchSessions')}</span>
              <input class="input w-full" type="search" placeholder={t('users.searchSessions')} bind:value={sessionSearch} />
            </label>
            <div class="card bg-surface-50-950 border border-surface-200-800 px-3 py-2 text-sm">
              <div class="text-xs text-surface-500">{t('users.visibleRows')}</div>
              <div class="font-semibold">{filteredSessions.length} / {sessions.length}</div>
            </div>
            <div class="card bg-surface-50-950 border border-surface-200-800 px-3 py-2 text-sm">
              <div class="text-xs text-surface-500">{t('users.activeSessions')}</div>
              <div class="font-semibold">{activeSessionCount}</div>
            </div>
          </div>
          <div class="overflow-x-auto">
            <table class="table min-w-full">
              <thead>
                <tr>
                  <th>{t('users.device')}</th>
                  <th>{t('users.ip')}</th>
                  <th>{t('users.loginMethod')}</th>
                  <th>{t('users.status')}</th>
                  <th>{t('users.createdAt')}</th>
                  <th>{t('users.expiresAt')}</th>
                  <th>{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {#each filteredSessions as session (session.id)}
                  <tr>
                    <td>
                      <div class="space-y-1">
                        <div class="font-medium">{session.device_id || '-'}</div>
                        <div class="max-w-72 truncate text-xs text-surface-500">{session.user_agent || '-'}</div>
                      </div>
                    </td>
                    <td>{session.ip || '-'}</td>
                    <td>{session.login_method || '-'}</td>
                    <td>{sessionStatusLabel(session.status)}</td>
                    <td class="whitespace-nowrap">{formatDateTime(session.created_at)}</td>
                    <td class="whitespace-nowrap">{formatDateTime(session.expires_at)}</td>
                    <td>
                      <button class="btn preset-tonal-error btn-xs" type="button" on:click={() => void revokeSession(session)}>
                        {pendingSessionRevokeId === session.id ? t('users.confirmRevokeSession') : t('users.revokeSession')}
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
          {#if filteredSessions.length === 0}
            <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.noSessionSearchResults')}</div>
          {/if}
        {/if}
      </div>
    {:else}
      <div class="space-y-4">
        <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 lg:grid-cols-5" on:submit|preventDefault={createBinding}>
          <label class="block">
            <span class="text-sm text-surface-500">{t('users.sourceId')}</span>
            <input class="input w-full" type="text" bind:value={bindingSourceId} required />
          </label>
          <label class="block">
            <span class="text-sm text-surface-500">{t('users.directoryUserId')}</span>
            <input class="input w-full" type="text" bind:value={bindingDirectoryUserId} required />
          </label>
          <label class="block">
            <span class="text-sm text-surface-500">{t('users.providerUid')}</span>
            <input class="input w-full" type="text" bind:value={bindingProviderUid} required />
          </label>
          <label class="block">
            <span class="text-sm text-surface-500">{t('users.providerUnionId')}</span>
            <input class="input w-full" type="text" bind:value={bindingProviderUnionId} />
          </label>
          <div class="flex items-end gap-3">
            <label class="flex items-center gap-2 pb-2 text-sm">
              <input class="checkbox" type="checkbox" bind:checked={bindingIsPrimary} />
              <span>{t('users.primaryBinding')}</span>
            </label>
            <button class="btn preset-filled-primary-500" type="submit" disabled={bindingSaving}>
              {bindingSaving ? t('common.loading') : t('users.createBinding')}
            </button>
          </div>
        </form>

        <div class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
          {#if bindingsLoading}
            <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
          {:else if bindings.length === 0}
            <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.noBindings')}</div>
          {:else}
            <div class="grid gap-3 border-b border-surface-200-800 p-4 md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-center">
              <label class="block">
                <span class="sr-only">{t('users.searchBindings')}</span>
                <input class="input w-full" type="search" placeholder={t('users.searchBindings')} bind:value={bindingSearch} />
              </label>
              <div class="card bg-surface-50-950 border border-surface-200-800 px-3 py-2 text-sm">
                <div class="text-xs text-surface-500">{t('users.visibleRows')}</div>
                <div class="font-semibold">{filteredBindings.length} / {bindings.length}</div>
              </div>
              <div class="card bg-surface-50-950 border border-surface-200-800 px-3 py-2 text-sm">
                <div class="text-xs text-surface-500">{t('users.primaryBindings')}</div>
                <div class="font-semibold">{primaryBindingCount}</div>
              </div>
            </div>
            <div class="overflow-x-auto">
              <table class="table min-w-full">
                <thead>
                  <tr>
                    <th>{t('users.source')}</th>
                    <th>{t('users.directoryUserId')}</th>
                    <th>{t('users.providerUid')}</th>
                    <th>{t('users.primaryBinding')}</th>
                    <th>{t('users.boundAt')}</th>
                    <th>{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {#each filteredBindings as binding (binding.id)}
                    <tr>
                      <td>
                        <div class="space-y-1">
                          <div class="font-medium">{binding.source_name || binding.source_type || '-'}</div>
                          <div class="text-xs text-surface-500">{binding.source_id}</div>
                        </div>
                      </td>
                      <td class="max-w-64 truncate">{binding.directory_user_id}</td>
                      <td>
                        <div class="space-y-1">
                          <div>{binding.provider_uid}</div>
                          <div class="text-xs text-surface-500">{binding.provider_union_id || '-'}</div>
                        </div>
                      </td>
                      <td>{binding.is_primary ? t('common.yes') : t('common.no')}</td>
                      <td class="whitespace-nowrap">{formatDateTime(binding.bound_at)}</td>
                      <td>
                        <button class="btn preset-tonal-error btn-xs" type="button" on:click={() => void deleteBinding(binding)}>
                          {pendingBindingDeleteId === binding.id ? t('common.confirmDelete') : t('common.delete')}
                        </button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
            {#if filteredBindings.length === 0}
              <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.noBindingSearchResults')}</div>
            {/if}
          {/if}
        </div>
      </div>
    {/if}
  </section>
{/if}
