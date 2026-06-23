<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type DirectoryUser } from '$lib/api';
  import { t } from '$lib/i18n';
  import { Copy, Eye, RotateCcw, Search } from 'lucide-svelte';

  let users: DirectoryUser[] = [];
  let total = 0;
  let limit = 25;
  let offset = 0;
  let statusFilter = 'all';
  let searchTerm = '';
  let loading = true;
  let error = '';
  let detailOpen = false;
  let detailLoading = false;
  let selectedUser: DirectoryUser | null = null;
  let copiedValue = '';

  const formatDateTime = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');
  const rawValue = (value: unknown, key: string): unknown =>
    value && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>)[key] : undefined;
  const formatDepartmentValue = (value: unknown): string => {
    if (Array.isArray(value)) return value.map((item) => String(item)).filter(Boolean).join(', ');
    return value ? String(value) : '';
  };
  const departmentText = (user: DirectoryUser): string => {
    const profile = user.raw_profile;
    return (
      formatDepartmentValue(rawValue(profile, 'department_names')) ||
      formatDepartmentValue(rawValue(profile, 'departments')) ||
      formatDepartmentValue(rawValue(profile, 'department')) ||
      formatDepartmentValue(rawValue(profile, 'department_ids')) ||
      formatDepartmentValue(rawValue(profile, 'department_id')) ||
      '-'
    );
  };
  const matchesSearch = (user: DirectoryUser, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [
      user.name,
      user.email,
      user.phone,
      departmentText(user),
      user.external_user_id,
      user.external_union_id,
      user.external_open_id,
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

  const resetFilters = () => {
    statusFilter = 'all';
    searchTerm = '';
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

  const copyValue = async (value?: string) => {
    if (!value || typeof navigator === 'undefined' || !navigator.clipboard) return;
    await navigator.clipboard.writeText(value);
    copiedValue = value;
    setTimeout(() => {
      if (copiedValue === value) copiedValue = '';
    }, 1200);
  };

  const copyIconLabel = (value?: string): string => (copiedValue === value ? t('common.copied') : t('common.copy'));

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
  $: withEmailCount = users.filter((user) => Boolean(user.email)).length;
</script>

<svelte:head>
  <title>{t('directory.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <form class="card bg-surface-50-950 border border-surface-200-800 flex flex-wrap items-center justify-between gap-2 p-3" on:submit|preventDefault aria-label={t('directory.title')}>
    <label class="relative min-w-56 flex-1 sm:max-w-md">
      <span class="sr-only">{t('directory.search')}</span>
      <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
      <input class="input h-8 w-full pl-9 text-sm" type="search" bind:value={searchTerm} placeholder={t('directory.searchPlaceholder')} />
    </label>

    <div class="flex flex-wrap items-center gap-2">
      <label>
        <span class="sr-only">{t('directory.statusFilter')}</span>
        <select class="input h-8 w-32 text-sm" bind:value={statusFilter}>
          <option value="all">{t('common.all')}</option>
          <option value="active">{t('identitySources.status.active')}</option>
          <option value="disabled">{t('identitySources.status.disabled')}</option>
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
              <th>{t('directory.department')}</th>
              <th>{t('directory.status')}</th>
              <th>{t('directory.email')}</th>
              <th>{t('directory.phone')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredUsers as item (item.id)}
              <tr>
                <td>
                  <div class="space-y-1">
                    <div class="font-medium">{item.name || '-'}</div>
                    <div class="flex items-center gap-1 text-[0.68rem] font-mono leading-4 text-surface-500">
                      <span class="max-w-44 truncate">{item.id}</span>
                      <button class="inline-grid size-4 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(item.id)} aria-label={copyIconLabel(item.id)} title={copyIconLabel(item.id)}>
                        <Copy class="size-3" aria-hidden="true" />
                      </button>
                    </div>
                  </div>
                </td>
                <td class="max-w-48 truncate">{departmentText(item)}</td>
                <td>
                  <span class={`badge ${item.status === 'active' ? 'preset-tonal-success' : 'preset-outlined-surface-500'}`}>{item.status || '-'}</span>
                </td>
                <td>{item.email || '-'}</td>
                <td>{item.phone || '-'}</td>
                <td>
                  <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => void openDetails(item)} aria-label={t('directory.details')} title={t('directory.details')}>
                    <Eye class="size-4" aria-hidden="true" />
                  </button>
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
            <dt class="text-surface-500">{t('directory.englishName')}</dt>
            <dd class="font-medium">{selectedUser.english_name || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.employeeNo')}</dt>
            <dd class="font-medium">{selectedUser.employee_no || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.jobTitle')}</dt>
            <dd class="font-medium">{selectedUser.job_title || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.userId')}</dt>
            <dd class="flex min-w-0 items-center gap-2 font-mono text-xs">
              <span class="break-all">{selectedUser.id}</span>
              <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.id)} aria-label={copyIconLabel(selectedUser.id)} title={copyIconLabel(selectedUser.id)}>
                <Copy class="size-3" aria-hidden="true" />
              </button>
            </dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.status')}</dt>
            <dd class="font-medium">{selectedUser.status || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.externalUserId')}</dt>
            <dd class="flex min-w-0 items-center gap-2 font-mono text-xs">
              <span class="break-all">{selectedUser.external_user_id || '-'}</span>
              {#if selectedUser.external_user_id}
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.external_user_id)} aria-label={copyIconLabel(selectedUser.external_user_id)} title={copyIconLabel(selectedUser.external_user_id)}>
                  <Copy class="size-3" aria-hidden="true" />
                </button>
              {/if}
            </dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.externalUnionId')}</dt>
            <dd class="flex min-w-0 items-center gap-2 font-mono text-xs">
              <span class="break-all">{selectedUser.external_union_id || '-'}</span>
              {#if selectedUser.external_union_id}
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.external_union_id)} aria-label={copyIconLabel(selectedUser.external_union_id)} title={copyIconLabel(selectedUser.external_union_id)}>
                  <Copy class="size-3" aria-hidden="true" />
                </button>
              {/if}
            </dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.externalOpenId')}</dt>
            <dd class="flex min-w-0 items-center gap-2 font-mono text-xs">
              <span class="break-all">{selectedUser.external_open_id || '-'}</span>
              {#if selectedUser.external_open_id}
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.external_open_id)} aria-label={copyIconLabel(selectedUser.external_open_id)} title={copyIconLabel(selectedUser.external_open_id)}>
                  <Copy class="size-3" aria-hidden="true" />
                </button>
              {/if}
            </dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.email')}</dt>
            <dd class="flex min-w-0 items-center gap-2 font-medium">
              <span class="break-all">{selectedUser.email || '-'}</span>
              {#if selectedUser.email}
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.email)} aria-label={copyIconLabel(selectedUser.email)} title={copyIconLabel(selectedUser.email)}>
                  <Copy class="size-3" aria-hidden="true" />
                </button>
              {/if}
            </dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.phone')}</dt>
            <dd class="flex min-w-0 items-center gap-2 font-medium">
              <span class="break-all">{selectedUser.phone || '-'}</span>
              {#if selectedUser.phone}
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.phone)} aria-label={copyIconLabel(selectedUser.phone)} title={copyIconLabel(selectedUser.phone)}>
                  <Copy class="size-3" aria-hidden="true" />
                </button>
              {/if}
            </dd>
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
      {/if}
    </div>
  </div>
{/if}
