<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type DirectoryUser } from '$lib/api';
  import { t } from '$lib/i18n';

  let users: DirectoryUser[] = [];
  let total = 0;
  let limit = 25;
  let offset = 0;
  let sourceIdFilter = '';
  let statusFilter = 'all';
  let searchTerm = '';
  let loading = true;
  let error = '';
  let detailOpen = false;
  let detailLoading = false;
  let selectedUser: DirectoryUser | null = null;

  const formatDateTime = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');
  const matchesSearch = (user: DirectoryUser, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [
      user.name,
      user.email,
      user.phone,
      user.external_user_id,
      user.external_union_id,
      user.external_open_id,
      user.source_id,
      user.status,
      user.id,
    ]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const loadUsers = async () => {
    loading = true;
    error = '';
    try {
      const data = await api.listDirectoryUsers({
        source_id: sourceIdFilter,
        limit,
        offset,
      });
      users = data.items || [];
      total = data.total || 0;
    } catch {
      error = t('directory.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const applyFilters = () => {
    offset = 0;
    void loadUsers();
  };

  const resetFilters = () => {
    sourceIdFilter = '';
    statusFilter = 'all';
    searchTerm = '';
    offset = 0;
    void loadUsers();
  };

  const previousPage = () => {
    if (offset === 0) return;
    offset = Math.max(0, offset - limit);
    void loadUsers();
  };

  const nextPage = () => {
    if (offset + limit >= total) return;
    offset += limit;
    void loadUsers();
  };

  const rawProfileText = (value: unknown): string => {
    if (!value) return '-';
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value);
    }
  };

  const openDetails = async (user: DirectoryUser) => {
    detailOpen = true;
    detailLoading = true;
    selectedUser = user;
    error = '';
    try {
      selectedUser = await api.getDirectoryUser(user.id);
    } catch {
      error = t('directory.detailFetchFailed');
    } finally {
      detailLoading = false;
    }
  };

  const closeDetails = () => {
    detailOpen = false;
    selectedUser = null;
    detailLoading = false;
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && detailOpen) {
      closeDetails();
    }
  };

  onMount(() => {
    void loadUsers();
  });

  $: filteredUsers = users.filter((user) => {
    const statusMatches = statusFilter === 'all' || user.status === statusFilter;
    return statusMatches && matchesSearch(user, searchTerm);
  });
  $: activeCount = users.filter((user) => user.status === 'active').length;
  $: withEmailCount = users.filter((user) => Boolean(user.email)).length;
  $: withPhoneCount = users.filter((user) => Boolean(user.phone)).length;
  $: syncedCount = users.filter((user) => Boolean(user.last_synced_at)).length;
  $: sourceCount = new Set(users.map((user) => user.source_id).filter(Boolean)).size;
  $: pageStart = total === 0 ? 0 : offset + 1;
  $: pageEnd = Math.min(offset + limit, total);
</script>

<svelte:head>
  <title>{t('directory.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <header class="flex flex-wrap items-end justify-between gap-3">
    <span aria-hidden="true"></span>
    <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void loadUsers()}>{t('common.retry')}</button>
  </header>

  <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,14rem)_auto]" on:submit|preventDefault={applyFilters}>
    <label class="block">
      <span class="text-sm text-surface-500">{t('directory.sourceId')}</span>
      <input class="input w-full" type="text" bind:value={sourceIdFilter} />
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('directory.search')}</span>
      <input class="input w-full" type="search" bind:value={searchTerm} placeholder={t('directory.searchPlaceholder')} />
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('directory.statusFilter')}</span>
      <select class="input w-full" bind:value={statusFilter}>
        <option value="all">{t('common.all')}</option>
        <option value="active">{t('identitySources.status.active')}</option>
        <option value="disabled">{t('identitySources.status.disabled')}</option>
      </select>
    </label>
    <div class="flex flex-wrap items-end gap-2">
      <button class="btn preset-filled-primary-500" type="submit">{t('directory.filter')}</button>
      <button class="btn preset-outlined-surface-500" type="button" on:click={resetFilters}>{t('directory.reset')}</button>
    </div>
  </form>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <div class="grid gap-3 border-b border-surface-200-800 p-4 text-sm sm:grid-cols-3 lg:grid-cols-6">
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('directory.pageRange')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${pageStart}-${pageEnd}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('directory.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredUsers.length} / ${users.length}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('directory.activeUsers')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('directory.syncedUsers')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${syncedCount} / ${users.length}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('directory.sourceCount')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{sourceCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('directory.withPhone')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{withPhoneCount}</p></article>
    </div>

    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if users.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('directory.noUsers')}</div>
    {:else if filteredUsers.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('directory.noSearchResults')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th>{t('directory.name')}</th>
              <th>{t('directory.status')}</th>
              <th>{t('directory.email')}</th>
              <th>{t('directory.phone')}</th>
              <th>{t('directory.externalUserId')}</th>
              <th>{t('directory.sourceId')}</th>
              <th>{t('directory.lastSyncedAt')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredUsers as item (item.id)}
              <tr>
                <td>
                  <div class="space-y-1">
                    <div class="font-medium">{item.name || '-'}</div>
                    <div class="text-xs text-surface-500">{item.id}</div>
                  </div>
                </td>
                <td>
                  <span class={`badge ${item.status === 'active' ? 'preset-tonal-success' : 'preset-outlined-surface-500'}`}>{item.status || '-'}</span>
                </td>
                <td>{item.email || '-'}</td>
                <td>{item.phone || '-'}</td>
                <td class="max-w-48 truncate">{item.external_user_id || '-'}</td>
                <td class="max-w-48 truncate">{item.source_id || '-'}</td>
                <td class="whitespace-nowrap">
                  <time datetime={item.last_synced_at || ''}>{formatDateTime(item.last_synced_at)}</time>
                </td>
                <td>
                  <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => void openDetails(item)}>{t('directory.details')}</button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}

    <div class="flex flex-wrap items-center justify-between gap-3 border-t border-surface-200-800 p-3">
      <span class="text-xs text-surface-500">{t('dashboard.total')}: {total} · {t('directory.withEmail')}: {withEmailCount}</span>
      <div class="flex gap-2">
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={previousPage} disabled={offset === 0}>{t('common.previous')}</button>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={nextPage} disabled={offset + limit >= total}>{t('common.next')}</button>
      </div>
    </div>
  </section>
</section>

{#if detailOpen && selectedUser}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="directory-user-dialog-title">
    <div class="mx-auto mt-10 max-w-3xl rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="directory-user-dialog-title" class="font-semibold">{t('directory.details')}</h2>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDetails}>{t('common.close')}</button>
      </div>

      {#if detailLoading}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {:else}
        <dl class="grid gap-3 text-sm md:grid-cols-2">
          <div>
            <dt class="text-surface-500">{t('directory.name')}</dt>
            <dd class="font-medium">{selectedUser.name || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.status')}</dt>
            <dd class="font-medium">{selectedUser.status || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.sourceId')}</dt>
            <dd class="break-all font-medium">{selectedUser.source_id}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.externalUserId')}</dt>
            <dd class="break-all font-medium">{selectedUser.external_user_id || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.externalUnionId')}</dt>
            <dd class="break-all font-medium">{selectedUser.external_union_id || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.externalOpenId')}</dt>
            <dd class="break-all font-medium">{selectedUser.external_open_id || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.email')}</dt>
            <dd class="font-medium">{selectedUser.email || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.phone')}</dt>
            <dd class="font-medium">{selectedUser.phone || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.lastSyncedAt')}</dt>
            <dd class="font-medium">{formatDateTime(selectedUser.last_synced_at)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.updatedAt')}</dt>
            <dd class="font-medium">{formatDateTime(selectedUser.updated_at)}</dd>
          </div>
        </dl>

        <div class="mt-4">
          <h3 class="text-sm font-semibold">{t('directory.rawProfile')}</h3>
          <pre class="card bg-surface-100-900 border border-surface-200-800 overflow-x-auto p-3 text-xs mt-2"><code>{rawProfileText(selectedUser.raw_profile)}</code></pre>
        </div>
      {/if}
    </div>
  </div>
{/if}
