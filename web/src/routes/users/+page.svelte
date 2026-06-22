<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type User, type Application, type ApplicationAssignment, type UpdateUserRequest, type UserListResponse } from '$lib/api';

  let users: User[] = [];
  let total = 0;
  let limit = 20;
  let offset = 0;
  let statusFilter = '';
  let userTypeFilter = 'all';
  let localeFilter = 'all';
  let searchTerm = '';
  let loading = true;
  let error = '';
  let message = '';

  let editOpen = false;
  let savingUser = false;
  let editingUser: User | null = null;
  let formDisplayName = '';
  let formEmail = '';
  let formPhone = '';
  let formLocale = '';

  let authOpen = false;
  let authLoading = false;
  let authSaving = false;
  let authUser: User | null = null;
  let allApplications: Application[] = [];
  let appAuthorization = new Map<string, string>();
  let selectedAppIds: string[] = [];

  const statusOptions = ['', 'active', 'disabled', 'locked'];
  const userTypeOptions = ['all', 'human', 'service_account', 'external'];
  const localeOptions = ['all', 'en-US', 'zh-CN'];
  const userStatusLabel = (value: string): string => t(`users.status.${value}`, value);
  const userTypeLabel = (value: string): string => t(`users.type.${value}`, value);
  const applicationTypeLabel = (value: string): string => t(`applications.type.${value}`, value);
  const localeLabel = (value: string): string => (value === 'zh-CN' ? t('layout.chinese') : t('layout.english'));
  const matchesSearch = (user: User, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [user.username, user.display_name, user.email, user.phone, user.user_type, user.lifecycle_status, user.locale, user.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const loadUsers = async () => {
    loading = true;
    error = '';
    try {
      const data: UserListResponse = await api.listUsers({
        limit,
        offset,
        status: statusFilter,
      });
      users = data.items || [];
      total = data.total || 0;
    } catch {
      error = t('users.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const onStatusFilterChange = (event: Event) => {
    statusFilter = (event.currentTarget as HTMLSelectElement).value;
    offset = 0;
    void loadUsers();
  };

  const resetFilters = () => {
    searchTerm = '';
    userTypeFilter = 'all';
    localeFilter = 'all';
    statusFilter = '';
    offset = 0;
    void loadUsers();
  };

  const onPrevPage = () => {
    if (offset === 0) return;
    offset = Math.max(0, offset - limit);
    void loadUsers();
  };

  const onNextPage = () => {
    if (offset + limit >= total) return;
    offset += limit;
    void loadUsers();
  };

  const openEdit = (user: User) => {
    editingUser = user;
    formDisplayName = user.display_name || '';
    formEmail = user.email || '';
    formPhone = user.phone || '';
    formLocale = user.locale || 'en-US';
    editOpen = true;
  };

  const closeEdit = () => {
    editOpen = false;
    editingUser = null;
  };

  const saveUser = async () => {
    if (!editingUser) return;
    savingUser = true;
    error = '';
    message = '';

    try {
      const payload: UpdateUserRequest = {
        display_name: formDisplayName,
        email: formEmail,
        phone: formPhone,
        locale: formLocale,
      };
      await api.updateUser(editingUser.id, payload);
      message = t('users.saveSuccess');
      await loadUsers();
      closeEdit();
    } catch {
      error = t('users.saveFailed');
    } finally {
      savingUser = false;
    }
  };

  const toggleUserStatus = async (user: User) => {
    error = '';
    message = '';
    try {
      if (user.lifecycle_status === 'active') {
        await api.disableUser(user.id);
        message = t('users.disableSuccess');
      } else {
        await api.enableUser(user.id);
        message = t('users.enableSuccess');
      }
      await loadUsers();
    } catch {
      error = t('common.fetchFailed');
    }
  };

  const openAuthorization = async (user: User) => {
    authOpen = true;
    authLoading = true;
    authUser = user;
    appAuthorization = new Map<string, string>();
    selectedAppIds = [];

    try {
      const appData = await api.listApplications({ limit: 200 });
      allApplications = appData.applications || [];

      const entries = await Promise.all(
        allApplications.map(async (app) => {
          try {
            const assData = await api.listAssignments(app.id, { limit: 200 });
            const assignment = (assData.assignments || []).find(
              (it: ApplicationAssignment) => it.subject_type === 'user' && it.subject_id === user.id,
            );
            return assignment ? { appId: app.id, assignmentId: assignment.id } : null;
          } catch {
            return null;
          }
        }),
      );

      for (const item of entries) {
        if (!item) continue;
        appAuthorization.set(item.appId, item.assignmentId || '');
        selectedAppIds.push(item.appId);
      }
    } catch {
      error = t('users.fetchFailed');
    } finally {
      authLoading = false;
    }
  };

  const closeAuthorization = () => {
    authOpen = false;
    authUser = null;
    selectedAppIds = [];
    appAuthorization = new Map<string, string>();
    allApplications = [];
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key !== 'Escape') return;
    if (editOpen) {
      closeEdit();
    } else if (authOpen) {
      closeAuthorization();
    }
  };

  const onAppAccessToggle = (appId: string, checked: boolean) => {
    if (checked) {
      if (!selectedAppIds.includes(appId)) {
        selectedAppIds = [...selectedAppIds, appId];
      }
    } else {
      selectedAppIds = selectedAppIds.filter((id) => id !== appId);
    }
  };

  const saveAuthorization = async () => {
    if (!authUser) return;
    authSaving = true;
    error = '';
    message = '';

    try {
      const current = new Set(appAuthorization.keys());
      const next = new Set(selectedAppIds);

      const created: Promise<unknown>[] = [];
      const removed: Promise<unknown>[] = [];

      for (const appId of next) {
        if (!current.has(appId)) {
          created.push(
            api.createAssignment(appId, {
              subject_type: 'user',
              subject_id: authUser.id,
              effect: 'allow',
            }),
          );
        }
      }

      for (const appId of current) {
        if (!next.has(appId)) {
          const assignmentId = appAuthorization.get(appId);
          if (assignmentId) {
            removed.push(api.deleteAssignment(assignmentId));
          }
        }
      }

      await Promise.all([...created, ...removed]);
      message = t('users.saveSuccess');
      closeAuthorization();
      await loadUsers();
    } catch {
      error = t('assignments.createFailed');
    } finally {
      authSaving = false;
    }
  };

  onMount(() => {
    void loadUsers();
  });

  $: filteredUsers = users.filter((user) => {
    const typeMatches = userTypeFilter === 'all' || user.user_type === userTypeFilter;
    const localeMatches = localeFilter === 'all' || (user.locale || 'en-US') === localeFilter;
    return typeMatches && localeMatches && matchesSearch(user, searchTerm);
  });
  $: activeUserCount = users.filter((user) => user.lifecycle_status === 'active').length;
  $: disabledUserCount = users.filter((user) => user.lifecycle_status === 'disabled').length;
  $: humanUserCount = users.filter((user) => user.user_type === 'human').length;
  $: serviceAccountCount = users.filter((user) => user.user_type === 'service_account').length;
  $: withEmailCount = users.filter((user) => Boolean(user.email)).length;
  $: pageStart = total === 0 ? 0 : offset + 1;
  $: pageEnd = Math.min(offset + limit, total);
</script>

<svelte:head>
  <title>{t('users.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <header class="flex flex-wrap items-end gap-3 justify-between">
    <span aria-hidden="true"></span>
    <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void loadUsers()}>{t('common.retry')}</button>
  </header>

  <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_minmax(0,12rem)_minmax(0,12rem)_minmax(0,12rem)_auto]" on:submit|preventDefault>
    <label class="block">
      <span class="text-sm text-surface-500">{t('users.search')}</span>
      <input class="input w-full" type="search" bind:value={searchTerm} placeholder={t('users.searchPlaceholder')} />
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('users.status')}</span>
      <select class="input w-full" on:change={onStatusFilterChange} value={statusFilter}>
        <option value="">{t('common.all')}</option>
        {#each statusOptions as status}
          {#if status !== ''}
            <option value={status}>{userStatusLabel(status)}</option>
          {/if}
        {/each}
      </select>
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('users.type')}</span>
      <select class="input w-full" bind:value={userTypeFilter}>
        {#each userTypeOptions as userTypeOption}
          <option value={userTypeOption}>{userTypeOption === 'all' ? t('common.all') : userTypeLabel(userTypeOption)}</option>
        {/each}
      </select>
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('users.locale')}</span>
      <select class="input w-full" bind:value={localeFilter}>
        {#each localeOptions as locale}
          <option value={locale}>{locale === 'all' ? t('common.all') : localeLabel(locale)}</option>
        {/each}
      </select>
    </label>
    <div class="flex flex-wrap items-end gap-2">
      <button class="btn preset-filled-primary-500" type="submit">{t('common.filter')}</button>
      <button class="btn preset-outlined-surface-500" type="button" on:click={resetFilters}>{t('common.reset')}</button>
    </div>
  </form>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}
  {#if message}
    <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
  {/if}

  <div class="card bg-surface-50-950 border border-surface-200-800 p-4">
    <div class="mb-4 grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-6">
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('users.pageRange')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${pageStart}-${pageEnd}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('users.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredUsers.length} / ${users.length}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('users.status.active')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeUserCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('users.status.disabled')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{disabledUserCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('users.humanUsers')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{humanUserCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('users.serviceAccounts')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{serviceAccountCount}</p></article>
    </div>

    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.loading')}</div>
    {:else if !users.length}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.noData')}</div>
    {:else if !filteredUsers.length}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.noSearchResults')}</div>
    {:else}
      <div class="divide-y divide-surface-200-800">
        {#each filteredUsers as user}
          <article class="py-3">
            <div class="flex flex-wrap justify-between gap-2">
              <div>
                <div class="font-semibold">{user.username}</div>
                <div class="text-xs text-surface-500">{user.display_name || '-'}</div>
              </div>
              <span class={`badge ${user.lifecycle_status === 'active' ? 'preset-tonal-success' : 'preset-tonal-error'}`}>{userStatusLabel(user.lifecycle_status)}</span>
            </div>
            <div class="text-sm text-surface-500 mt-1">
              {user.email || '-'} · {userTypeLabel(user.user_type)}
            </div>
            <div class="mt-2 flex flex-wrap gap-2 text-xs text-surface-500">
              <span>{t('users.phone')}: {user.phone || '-'}</span>
              <span>{t('users.locale')}: {user.locale || '-'}</span>
              <span>{t('users.createdAt')}: {user.created_at ? new Date(user.created_at).toLocaleString() : '-'}</span>
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              <a class="btn btn-sm preset-outlined-surface-500 inline-flex" href={`/users/${user.id}`}>{t('users.detail')}</a>
              <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => openEdit(user)}>{t('users.edit')}</button>
              <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void openAuthorization(user)}>{t('users.authOpen')}</button>
              <button
                class={`btn btn-sm ${
                  user.lifecycle_status === 'active' ? 'preset-tonal-error' : 'preset-filled-success-500'
                }`}
                type="button"
                on:click={() => void toggleUserStatus(user)}
              >
                {user.lifecycle_status === 'active' ? t('users.disable') : t('users.enable')}
              </button>
            </div>
          </article>
        {/each}
      </div>
    {/if}

    <div class="flex justify-between items-center p-3">
      <span class="text-xs text-surface-500">{t('dashboard.total')}: {total} · {t('directory.withEmail')}: {withEmailCount}</span>
      <div class="flex gap-2">
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={onPrevPage} disabled={offset === 0}>{t('common.previous')}</button>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={onNextPage} disabled={offset + limit >= total}>{t('common.next')}</button>
      </div>
    </div>
  </div>
</section>

{#if editOpen}
  <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="user-edit-dialog-title" tabindex="-1">
    <form class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-xl w-full overflow-y-auto p-4 space-y-4" on:submit|preventDefault={saveUser}>
      <h2 id="user-edit-dialog-title" class="text-lg font-semibold">{t('users.editUser')}</h2>

      <label class="block">
        <span class="block text-sm text-surface-500 mb-1">{t('users.displayName')}</span>
        <input class="input w-full" type="text" bind:value={formDisplayName} />
      </label>
      <label class="block">
        <span class="block text-sm text-surface-500 mb-1">{t('users.email')}</span>
        <input class="input w-full" type="email" bind:value={formEmail} />
      </label>
      <label class="block">
        <span class="block text-sm text-surface-500 mb-1">{t('users.phone')}</span>
        <input class="input w-full" type="tel" bind:value={formPhone} />
      </label>
      <label class="block">
        <span class="block text-sm text-surface-500 mb-1">{t('users.locale')}</span>
        <select class="input w-full" bind:value={formLocale}>
          <option value="en-US">{t('layout.english')}</option>
          <option value="zh-CN">{t('layout.chinese')}</option>
        </select>
      </label>

      <div class="flex justify-end gap-2">
        <button class="btn preset-outlined-surface-500" type="button" on:click={closeEdit}>{t('common.cancel')}</button>
        <button class="btn preset-filled-primary-500" type="submit" disabled={savingUser}>
          {savingUser ? t('common.loading') : t('common.save')}
        </button>
      </div>
    </form>
  </div>
{/if}

{#if authOpen}
  <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="user-authorization-dialog-title" tabindex="-1">
    <form class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-2xl w-full overflow-y-auto p-4 space-y-4" on:submit|preventDefault={saveAuthorization}>
      <h2 id="user-authorization-dialog-title" class="text-lg font-semibold">{authUser ? `${t('users.assignApps')} - ${authUser.username}` : t('users.assignApps')}</h2>

      {#if authLoading}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {:else}
        <div class="divide-y divide-surface-200-800 max-h-80 overflow-auto">
          {#each allApplications as app}
            <article>
              <label class="flex items-center gap-3 p-2">
                <input
                  type="checkbox"
                  checked={selectedAppIds.includes(app.id)}
                  on:change={(event) => onAppAccessToggle(app.id, (event.currentTarget as HTMLInputElement).checked)}
                />
                <span>{app.name}</span>
                <span class="text-xs text-surface-500">({applicationTypeLabel(app.type)})</span>
              </label>
            </article>
          {/each}
          {#if !allApplications.length}
            <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
          {/if}
        </div>
      {/if}

      <div class="flex justify-end gap-2">
        <button class="btn preset-outlined-surface-500" type="button" on:click={closeAuthorization}>{t('common.cancel')}</button>
        <button class="btn preset-filled-primary-500" type="submit" disabled={authSaving}>
          {authSaving ? t('common.loading') : t('common.save')}
        </button>
      </div>
    </form>
  </div>
{/if}
