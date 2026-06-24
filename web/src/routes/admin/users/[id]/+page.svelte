<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { Switch, Tabs } from '@skeletonlabs/skeleton-svelte';
  import { t } from '$lib/i18n';
  import { api, type User, type Role, type UserSession, type AccountBinding } from '$lib/api';
  import { Check, Plus, Power, Search, Trash2, X } from 'lucide-svelte';
  import IdConfirmDialog from '$lib/components/ui/IdConfirmDialog.svelte';
  import { notifySuccess } from '$lib/toast';

  let userId = '';
  let user: User | null = null;
  let entityName = '';

  let activeTab: 'info' | 'roles' | 'sessions' | 'bindings' = 'info';
  let roleSearch = '';
  let sessionSearch = '';
  let bindingSearch = '';
  let loading = true;
  let userRoles: Role[] = [];
  let allRoles: Role[] = [];
  let availableRoles: Role[] = [];
  let roleLoading = false;

  let sessions: UserSession[] = [];
  let sessionsLoading = false;
  let pendingSessionRevokeId = '';

  let bindings: AccountBinding[] = [];
  let bindingsLoading = false;
  let bindingSaving = false;
  let bindingFormOpen = false;
  let pendingBindingDeleteId = '';
  let bindingSourceId = '';
  let bindingDirectoryUserId = '';
  let bindingProviderUid = '';
  let bindingProviderUnionId = '';
  let bindingIsPrimary = false;

  let selectedRoleId = '';
  let pendingRoleDeleteId = '';
  let pendingStatusChange = false;

  let error = '';
  $: userId = $page.params.id ?? '';

  const normalizeLocale = () => {
    return user?.locale || 'en-US';
  };

  const localeLabel = (value: string): string => (value === 'zh-CN' ? t('layout.chinese') : t('layout.english'));
  const userStatusLabel = (value: string): string => t(`users.status.${value}`, value);
  const userTypeLabel = (value: string): string => t(`users.type.${value}`, value);
  const sessionStatusLabel = (value: string): string => t(`users.sessionStatus.${value}`, value);
  const formatDateTime = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');

  const refreshUser = async () => {
    if (!userId) return;
    loading = true;
    error = '';
    entityName = '';

    try {
      user = await api.getUser(userId);
      if (user.entity_id) {
        try {
          const entity = await api.getEntity(user.entity_id);
          entityName = entity.name || '-';
        } catch {
          entityName = '-';
        }
      } else {
        entityName = '-';
      }
    } catch {
      error = t('users.fetchFailed');
      user = null;
      entityName = '';
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
    try {
      if (user.lifecycle_status === 'active') {
        user = await api.disableUser(user.id);
        notifySuccess(t('users.disableSuccess'));
      } else {
        user = await api.enableUser(user.id);
        notifySuccess(t('users.enableSuccess'));
      }
      await refreshUser();
      pendingStatusChange = false;
    } catch {
      error = t('users.saveFailed');
    }
  };

  const onAssignRole = async () => {
    if (!user || !selectedRoleId) return;
    error = '';

    try {
      await api.assignRoleToUser(user.id, selectedRoleId);
      selectedRoleId = '';
      notifySuccess(t('users.roleAssigned'));
      await loadRoles();
    } catch {
      error = t('users.saveFailed');
    }
  };

  const onRemoveRole = async (roleId: string) => {
    if (!user) return;
    error = '';
    try {
      await api.removeRoleFromUser(user.id, roleId);
      pendingRoleDeleteId = '';
      notifySuccess(t('users.roleRemoved'));
      await loadRoles();
    } catch {
      error = t('users.saveFailed');
    }
  };

  const revokeSession = async (session: UserSession) => {
    error = '';
    try {
      await api.revokeSession(session.id);
      pendingSessionRevokeId = '';
      notifySuccess(t('users.sessionRevoked'));
      await loadSessions();
    } catch {
      error = t('users.sessionRevokeFailed');
    }
  };

  const createBinding = async () => {
    if (!user) return;
    bindingSaving = true;
    error = '';
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
      bindingFormOpen = false;
      notifySuccess(t('users.bindingCreated'));
      await loadBindings();
    } catch {
      error = t('users.bindingCreateFailed');
    } finally {
      bindingSaving = false;
    }
  };

  const resetBindingForm = () => {
    bindingSourceId = '';
    bindingDirectoryUserId = '';
    bindingProviderUid = '';
    bindingProviderUnionId = '';
    bindingIsPrimary = false;
    bindingFormOpen = false;
  };

  const deleteBinding = async (binding: AccountBinding) => {
    if (!user) return;
    error = '';
    try {
      await api.deleteUserBinding(user.id, binding.id);
      pendingBindingDeleteId = '';
      notifySuccess(t('users.bindingDeleted'));
      await loadBindings();
    } catch {
      error = t('users.bindingDeleteFailed');
    }
  };

  const includesQuery = (value: unknown, query: string): boolean => String(value ?? '').toLowerCase().includes(query.trim().toLowerCase());

  const matchesRole = (role: Role, query: string): boolean => {
    if (!query.trim()) return true;
    return [role.id, role.name, role.code, role.description].some((value) => includesQuery(value, query));
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

  const changeTab = (value: string) => {
    activeTab = value as typeof activeTab;
    if (activeTab === 'roles') void loadRoles();
    if (activeTab === 'sessions') void loadSessions();
    if (activeTab === 'bindings') void loadBindings();
  };

  $: availableRoles = allRoles.filter((item) => !userRoles.find((r) => r.id === item.id));
  $: filteredUserRoles = userRoles.filter((role) => matchesRole(role, roleSearch));
  $: filteredSessions = sessions.filter((session) => matchesSession(session, sessionSearch));
  $: filteredBindings = bindings.filter((binding) => matchesBinding(binding, bindingSearch));
  $: activeSessionCount = sessions.filter((session) => session.status === 'active').length;

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
        <a class="btn btn-sm preset-outlined-surface-500 inline-flex" href="/admin/users">{t('common.back')}</a>
        <IdConfirmDialog
          open={pendingStatusChange}
          triggerLabel={user.lifecycle_status === 'active' ? t('users.disable') : t('users.enable')}
          triggerClass={`btn btn-xs inline-grid size-8 min-h-0 min-w-0 place-items-center p-0 ${user.lifecycle_status === 'active' ? 'preset-outlined-surface-500' : 'preset-filled-success-500'}`}
          confirmClass={user.lifecycle_status === 'active' ? 'preset-filled-error-500' : 'preset-filled-success-500'}
          onOpenChange={(open) => (pendingStatusChange = open)}
          onConfirm={() => void changeStatus()}
        >
          {#snippet trigger()}
            <Power class="size-4" aria-hidden="true" />
          {/snippet}
        </IdConfirmDialog>
      </div>
    </header>

    {#if error}
      <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
    {/if}
    <Tabs value={activeTab} onValueChange={(details) => changeTab(details.value)} class="w-full">
      <Tabs.List class="inline-flex flex-wrap gap-2" aria-label={t('users.info')}>
        <Tabs.Trigger class={`btn btn-sm ${activeTab === 'info' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} value="info">
          {t('users.info')}
        </Tabs.Trigger>
        <Tabs.Trigger class={`btn btn-sm ${activeTab === 'roles' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} value="roles">
          {t('users.roles')}
        </Tabs.Trigger>
        <Tabs.Trigger class={`btn btn-sm ${activeTab === 'sessions' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} value="sessions">
          {t('users.sessions')}
        </Tabs.Trigger>
        <Tabs.Trigger class={`btn btn-sm ${activeTab === 'bindings' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} value="bindings">
          {t('users.bindings')}
        </Tabs.Trigger>
      </Tabs.List>
    </Tabs>

    {#if activeTab === 'info'}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-4">
        <dl class="grid gap-3 text-sm md:grid-cols-2 xl:grid-cols-3">
          <div>
            <dt class="text-surface-500">{t('users.accountId')}</dt>
            <dd class="break-all font-mono text-xs">{user.id}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.entityId')}</dt>
            <dd class="font-medium">{entityName || '-'}</dd>
          </div>
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
            <dt class="text-surface-500">{t('users.updatedAt')}</dt>
            <dd class="font-medium">{formatDateTime(user.updated_at)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.locale')}</dt>
            <dd class="font-medium">{localeLabel(normalizeLocale())}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.primarySourceId')}</dt>
            <dd class="break-all font-mono text-xs">{user.primary_source_id || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('users.avatar')}</dt>
            <dd class="break-all">{user.avatar_url || '-'}</dd>
          </div>
        </dl>
      </div>
    {:else if activeTab === 'roles'}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-4">
        <p class="text-sm text-surface-600-400">{t('users.rolesDescription')}</p>
        <form class="flex flex-wrap items-center gap-2" on:submit|preventDefault={onAssignRole}>
          <select class="input h-8 min-w-56 flex-1 text-sm" aria-label={t('users.assignRole')} bind:value={selectedRoleId}>
            <option value="">{t('users.assignRole')}</option>
            {#each availableRoles as role}
              <option value={role.id}>{role.name}</option>
            {/each}
          </select>
          <button class="btn btn-sm preset-filled-primary-500" type="submit">{t('roles.assign')}</button>
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
                <input class="input h-8 w-full text-sm" type="search" placeholder={t('users.searchRoles')} bind:value={roleSearch} />
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
                    <div class="min-w-0">
                      <div class="font-medium">{role.name}</div>
                      <div class="truncate text-xs text-surface-500">{role.code} · {role.description || role.id}</div>
                    </div>
                    <IdConfirmDialog
                      open={pendingRoleDeleteId === role.id}
                      triggerLabel={t('common.delete')}
                      triggerClass="btn preset-tonal-error btn-xs"
                      onOpenChange={(open) => (pendingRoleDeleteId = open ? role.id : '')}
                      onConfirm={() => void onRemoveRole(role.id)}
                    >
                      {#snippet trigger()}
                        {t('common.delete')}
                      {/snippet}
                    </IdConfirmDialog>
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
    {:else if activeTab === 'sessions'}
      <div class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
        <div class="border-b border-surface-200-800 p-4 text-sm text-surface-600-400">{t('users.sessionsDescription')}</div>
        {#if sessionsLoading}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else if sessions.length === 0}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.noSessions')}</div>
        {:else}
          <div class="grid gap-3 border-b border-surface-200-800 p-4 md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-center">
            <label class="block">
              <span class="sr-only">{t('users.searchSessions')}</span>
              <input class="input h-8 w-full text-sm" type="search" placeholder={t('users.searchSessions')} bind:value={sessionSearch} />
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
                      <IdConfirmDialog
                        open={pendingSessionRevokeId === session.id}
                        triggerLabel={t('users.revokeSession')}
                        triggerClass="btn preset-tonal-error btn-xs"
                        onOpenChange={(open) => (pendingSessionRevokeId = open ? session.id : '')}
                        onConfirm={() => void revokeSession(session)}
                      >
                        {#snippet trigger()}
                          {t('users.revokeSession')}
                        {/snippet}
                      </IdConfirmDialog>
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
        <div class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-surface-200-800 p-3">
            <p class="max-w-3xl text-sm leading-6 text-surface-600-400">{t('users.bindingsDescription')}</p>
            <button
              class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0"
              type="button"
              on:click={() => (bindingFormOpen ? resetBindingForm() : (bindingFormOpen = true))}
              aria-label={bindingFormOpen ? t('common.cancel') : t('users.createBinding')}
              title={bindingFormOpen ? t('common.cancel') : t('users.createBinding')}
            >
              {#if bindingFormOpen}
                <X class="size-4" aria-hidden="true" />
              {:else}
                <Plus class="size-4" aria-hidden="true" />
              {/if}
            </button>
          </div>

          {#if bindingFormOpen}
            <form class="grid gap-3 border-b border-surface-200-800 p-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)_auto] lg:items-end" on:submit|preventDefault={createBinding}>
              <label class="block">
                <span class="text-xs text-surface-500">{t('users.sourceId')}</span>
                <input class="input h-8 w-full text-sm" type="text" bind:value={bindingSourceId} required />
              </label>
              <label class="block">
                <span class="text-xs text-surface-500">{t('users.directoryUserId')}</span>
                <input class="input h-8 w-full text-sm" type="text" bind:value={bindingDirectoryUserId} required />
              </label>
              <label class="block">
                <span class="text-xs text-surface-500">{t('users.providerUid')}</span>
                <input class="input h-8 w-full text-sm" type="text" bind:value={bindingProviderUid} required />
              </label>
              <label class="block">
                <span class="text-xs text-surface-500">{t('users.providerUnionId')}</span>
                <input class="input h-8 w-full text-sm" type="text" bind:value={bindingProviderUnionId} />
              </label>
              <div class="flex items-center justify-end gap-2">
                <Switch checked={bindingIsPrimary} onCheckedChange={(details) => (bindingIsPrimary = details.checked)} class="inline-flex items-center gap-2 text-sm">
                  <Switch.HiddenInput />
                  <Switch.Control class="relative inline-flex h-5 w-9 items-center rounded-full bg-surface-300-700 transition-colors data-[state=checked]:bg-primary-500">
                    <Switch.Thumb class="block size-4 rounded-full bg-white shadow transition-transform data-[state=checked]:translate-x-4" />
                  </Switch.Control>
                  <Switch.Label>{t('users.primaryBinding')}</Switch.Label>
                </Switch>
                <button
                  class="btn btn-xs preset-filled-primary-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0"
                  type="submit"
                  disabled={bindingSaving}
                  aria-label={bindingSaving ? t('common.loading') : t('users.createBinding')}
                  title={bindingSaving ? t('common.loading') : t('users.createBinding')}
                >
                  <Check class="size-4" aria-hidden="true" />
                </button>
              </div>
            </form>
          {/if}

          {#if bindingsLoading}
            <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
          {:else if bindings.length === 0}
            <div class="p-6 text-center text-sm text-surface-500">{t('users.noBindings')}</div>
          {:else}
            <div class="border-b border-surface-200-800 p-3">
              <label class="relative block max-w-sm">
                <span class="sr-only">{t('users.searchBindings')}</span>
                <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
                <input class="input h-8 w-full pl-9 text-sm" type="search" placeholder={t('users.searchBindings')} bind:value={bindingSearch} />
              </label>
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
                          <div class="max-w-48 truncate text-xs text-surface-500">{binding.source_id}</div>
                        </div>
                      </td>
                      <td class="max-w-64 truncate text-xs text-surface-500">{binding.directory_user_id}</td>
                      <td>
                        <div class="space-y-1">
                          <div class="text-xs text-surface-600-400">{binding.provider_uid}</div>
                          <div class="max-w-48 truncate text-xs text-surface-500">{binding.provider_union_id || '-'}</div>
                        </div>
                      </td>
                      <td>{binding.is_primary ? t('common.yes') : t('common.no')}</td>
                      <td class="whitespace-nowrap">{formatDateTime(binding.bound_at)}</td>
                      <td>
                        <IdConfirmDialog
                          open={pendingBindingDeleteId === binding.id}
                          triggerLabel={t('common.delete')}
                          confirmLabel={t('common.confirmDelete')}
                          triggerClass="btn btn-xs preset-outlined-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                          onOpenChange={(open) => (pendingBindingDeleteId = open ? binding.id : '')}
                          onConfirm={() => void deleteBinding(binding)}
                        >
                          {#snippet trigger()}
                            <Trash2 class="size-4" aria-hidden="true" />
                          {/snippet}
                        </IdConfirmDialog>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
            {#if filteredBindings.length === 0}
              <div class="p-6 text-center text-sm text-surface-500">{t('users.noBindingSearchResults')}</div>
            {/if}
          {/if}
        </div>
      </div>
    {/if}
  </section>
{/if}
