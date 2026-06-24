<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type User, type UserListResponse } from '$lib/api';
  import { Power, RotateCcw, Search, Settings } from 'lucide-svelte';
  import IdConfirmDialog from '$lib/components/ui/IdConfirmDialog.svelte';
  import IdPagination from '$lib/components/ui/IdPagination.svelte';
  import { notifySuccess } from '$lib/toast';

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
  let pendingStatusUserId = '';

  const statusOptions = ['', 'active', 'disabled', 'locked'];
  const userTypeOptions = ['all', 'human', 'service_account', 'external'];
  const localeOptions = ['all', 'en-US', 'zh-CN'];
  const userStatusLabel = (value: string): string => t(`users.status.${value}`, value);
  const userTypeLabel = (value: string): string => t(`users.type.${value}`, value);
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

  const toggleUserStatus = async (user: User) => {
    error = '';
    try {
      if (user.lifecycle_status === 'active') {
        await api.disableUser(user.id);
        notifySuccess(t('users.disableSuccess'));
      } else {
        await api.enableUser(user.id);
        notifySuccess(t('users.enableSuccess'));
      }
      await loadUsers();
      pendingStatusUserId = '';
    } catch {
      error = t('common.fetchFailed');
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
  $: withEmailCount = users.filter((user) => Boolean(user.email)).length;
</script>

<svelte:head>
  <title>{t('users.title')}</title>
</svelte:head>

<section class="space-y-4">
  <form class="card bg-surface-50-950 border border-surface-200-800 flex flex-wrap items-center justify-between gap-2 p-3" on:submit|preventDefault aria-label={t('users.title')}>
    <label class="relative min-w-56 flex-1 sm:max-w-md">
      <span class="sr-only">{t('users.search')}</span>
      <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
      <input class="input h-8 w-full pl-9 text-sm" type="search" bind:value={searchTerm} placeholder={t('users.searchPlaceholder')} />
    </label>

    <div class="flex flex-wrap items-center gap-2">
      <label>
        <span class="sr-only">{t('users.status')}</span>
        <select class="input h-8 w-28 text-sm" on:change={onStatusFilterChange} value={statusFilter}>
          <option value="">{t('common.all')}</option>
          {#each statusOptions as status}
            {#if status !== ''}
              <option value={status}>{userStatusLabel(status)}</option>
            {/if}
          {/each}
        </select>
      </label>
      <label>
        <span class="sr-only">{t('users.type')}</span>
        <select class="input h-8 w-32 text-sm" bind:value={userTypeFilter}>
          {#each userTypeOptions as userTypeOption}
            <option value={userTypeOption}>{userTypeOption === 'all' ? t('common.all') : userTypeLabel(userTypeOption)}</option>
          {/each}
        </select>
      </label>
      <label>
        <span class="sr-only">{t('users.locale')}</span>
        <select class="input h-8 w-28 text-sm" bind:value={localeFilter}>
          {#each localeOptions as locale}
            <option value={locale}>{locale === 'all' ? t('common.all') : localeLabel(locale)}</option>
          {/each}
        </select>
      </label>
      <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void loadUsers()}>{t('common.retry')}</button>
      <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={resetFilters} aria-label={t('common.reset')}>
        <RotateCcw class="size-4" aria-hidden="true" />
      </button>
    </div>
  </form>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.loading')}</div>
    {:else if !users.length}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.noData')}</div>
    {:else if !filteredUsers.length}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('users.noSearchResults')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th>{t('login.account')}</th>
              <th>{t('users.displayName')}</th>
              <th>{t('users.type')}</th>
              <th class="w-16 !text-center">{t('users.status')}</th>
              <th>{t('users.email')}</th>
              <th>{t('users.phone')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
        {#each filteredUsers as user}
          <tr>
            <td>
              <div class="space-y-1">
                <div class="font-medium">{user.username || '-'}</div>
                <div class="max-w-44 truncate font-mono text-[0.68rem] leading-4 text-surface-500">{user.id}</div>
              </div>
            </td>
            <td>{user.display_name || '-'}</td>
            <td>{userTypeLabel(user.user_type)}</td>
            <td class="w-16 !text-center">
              <span class="mx-auto flex size-5 items-center justify-center">
                <span class={`size-2 rounded-full ${user.lifecycle_status === 'active' ? 'bg-success-500' : 'bg-error-500'}`} aria-hidden="true"></span>
                <span class="sr-only">{userStatusLabel(user.lifecycle_status)}</span>
              </span>
            </td>
            <td>{user.email || '-'}</td>
            <td>{user.phone || '-'}</td>
            <td>
              <div class="flex items-center gap-1.5">
                <a class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" href={`/admin/users/${user.id}`} aria-label={t('users.manage')} title={t('users.manage')}>
                  <Settings class="size-4" aria-hidden="true" />
                </a>
                <IdConfirmDialog
                  open={pendingStatusUserId === user.id}
                  triggerLabel={user.lifecycle_status === 'active' ? t('users.disable') : t('users.enable')}
                  triggerClass={`btn btn-xs inline-grid size-7 min-h-0 min-w-0 place-items-center p-0 ${user.lifecycle_status === 'active' ? 'preset-outlined-surface-500' : 'preset-filled-success-500'}`}
                  confirmClass={user.lifecycle_status === 'active' ? 'preset-filled-error-500' : 'preset-filled-success-500'}
                  onOpenChange={(open) => (pendingStatusUserId = open ? user.id : '')}
                  onConfirm={() => void toggleUserStatus(user)}
                >
                  {#snippet trigger()}
                    <Power class="size-4" aria-hidden="true" />
                  {/snippet}
                </IdConfirmDialog>
              </div>
            </td>
          </tr>
        {/each}
          </tbody>
        </table>
      </div>
    {/if}

    <div class="flex flex-wrap items-center justify-between gap-3 border-t border-surface-200-800 p-3">
      <span class="text-xs text-surface-500">{t('dashboard.total')}: {total} · {t('users.withEmail')}: {withEmailCount}</span>
      <IdPagination total={total} {offset} pageSize={limit} onPage={(nextOffset) => {
        offset = nextOffset;
        void loadUsers();
      }} />
    </div>
  </section>
</section>
