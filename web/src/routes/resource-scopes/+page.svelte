<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type ResourceScope } from '$lib/api';
  import { t } from '$lib/i18n';

  let scopes: ResourceScope[] = [];
  let total = 0;
  let limit = 25;
  let offset = 0;
  let typeFilter = '';
  let searchTerm = '';
  let loading = true;
  let error = '';
  let message = '';
  let saving = false;
  let dialogOpen = false;
  let editing: ResourceScope | null = null;
  let formType = '';
  let formKey = '';
  let formName = '';
  let pendingDeleteId = '';
  let detailOpen = false;
  let detailLoading = false;
  let selectedScope: ResourceScope | null = null;

  const formatDateTime = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');
  const matchesSearch = (scope: ResourceScope, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [scope.name, scope.type, scope.key, scope.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const loadScopes = async () => {
    loading = true;
    error = '';
    try {
      const data = await api.listResourceScopes({ type: typeFilter, limit, offset });
      scopes = data.items || [];
      total = data.total || 0;
    } catch {
      error = t('resourceScopes.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const openCreate = () => {
    editing = null;
    formType = '';
    formKey = '';
    formName = '';
    dialogOpen = true;
  };

  const openEdit = (scope: ResourceScope) => {
    editing = scope;
    formType = scope.type;
    formKey = scope.key;
    formName = scope.name;
    dialogOpen = true;
  };

  const closeDialog = () => {
    dialogOpen = false;
    editing = null;
    saving = false;
  };

  const openDetails = async (scope: ResourceScope) => {
    detailOpen = true;
    detailLoading = true;
    selectedScope = scope;
    error = '';
    try {
      selectedScope = await api.getResourceScope(scope.id);
    } catch {
      error = t('resourceScopes.detailFetchFailed');
    } finally {
      detailLoading = false;
    }
  };

  const closeDetails = () => {
    detailOpen = false;
    detailLoading = false;
    selectedScope = null;
  };

  const saveScope = async () => {
    saving = true;
    error = '';
    message = '';
    try {
      if (editing) {
        await api.updateResourceScope(editing.id, { name: formName });
        message = t('common.updateSuccess');
      } else {
        await api.createResourceScope({ type: formType, key: formKey, name: formName });
        message = t('common.createSuccess');
      }
      closeDialog();
      await loadScopes();
    } catch {
      error = t(editing ? 'resourceScopes.updateFailed' : 'resourceScopes.createFailed');
    } finally {
      saving = false;
    }
  };

  const deleteScope = async (scope: ResourceScope) => {
    if (pendingDeleteId !== scope.id) {
      pendingDeleteId = scope.id;
      return;
    }
    error = '';
    message = '';
    try {
      await api.deleteResourceScope(scope.id);
      pendingDeleteId = '';
      message = t('common.deleteSuccess');
      await loadScopes();
    } catch {
      error = t('common.deleteFailed');
    }
  };

  const applyFilters = () => {
    offset = 0;
    void loadScopes();
  };

  const resetFilters = () => {
    typeFilter = '';
    searchTerm = '';
    offset = 0;
    void loadScopes();
  };

  const previousPage = () => {
    if (offset === 0) return;
    offset = Math.max(0, offset - limit);
    void loadScopes();
  };

  const nextPage = () => {
    if (offset + limit >= total) return;
    offset += limit;
    void loadScopes();
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key !== 'Escape') return;
    if (dialogOpen) {
      closeDialog();
    } else if (detailOpen) {
      closeDetails();
    }
  };

  onMount(() => {
    void loadScopes();
  });

  $: filteredScopes = scopes.filter((scope) => matchesSearch(scope, searchTerm));
  $: uniqueTypeCount = new Set(scopes.map((scope) => scope.type).filter(Boolean)).size;
  $: uniqueKeyCount = new Set(scopes.map((scope) => `${scope.type}:${scope.key}`).filter(Boolean)).size;
  $: pageStart = total === 0 ? 0 : offset + 1;
  $: pageEnd = Math.min(offset + limit, total);
</script>

<svelte:head>
  <title>{t('resourceScopes.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <header class="flex flex-wrap items-end justify-between gap-3">
    <span aria-hidden="true"></span>
    <div class="flex gap-2">
      <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void loadScopes()}>{t('common.retry')}</button>
      <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openCreate}>{t('common.create')}</button>
    </div>
  </header>

  <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]" on:submit|preventDefault={applyFilters}>
    <label class="block">
      <span class="text-sm text-surface-500">{t('resourceScopes.type')}</span>
      <input class="input w-full" type="text" bind:value={typeFilter} />
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('resourceScopes.search')}</span>
      <input class="input w-full" type="search" bind:value={searchTerm} placeholder={t('resourceScopes.searchPlaceholder')} />
    </label>
    <div class="flex flex-wrap items-end gap-2">
      <button class="btn preset-filled-primary-500" type="submit">{t('resourceScopes.filter')}</button>
      <button class="btn preset-outlined-surface-500" type="button" on:click={resetFilters}>{t('resourceScopes.reset')}</button>
    </div>
  </form>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}
  {#if message}
    <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <div class="grid gap-3 border-b border-surface-200-800 p-4 text-sm sm:grid-cols-5">
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('resourceScopes.pageRange')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${pageStart}-${pageEnd}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('resourceScopes.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredScopes.length} / ${scopes.length}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('resourceScopes.uniqueTypes')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{uniqueTypeCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('resourceScopes.uniqueKeys')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{uniqueKeyCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('dashboard.total')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{total}</p></article>
    </div>

    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if scopes.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('resourceScopes.noScopes')}</div>
    {:else if filteredScopes.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('resourceScopes.noSearchResults')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th>{t('resourceScopes.name')}</th>
              <th>{t('resourceScopes.type')}</th>
              <th>{t('resourceScopes.key')}</th>
              <th>{t('resourceScopes.createdAt')}</th>
              <th>{t('resourceScopes.updatedAt')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredScopes as item (item.id)}
              <tr>
                <td>
                  <div class="space-y-1">
                    <div class="font-medium">{item.name}</div>
                    <div class="text-xs text-surface-500">{item.id}</div>
                  </div>
                </td>
                <td>{item.type || '-'}</td>
                <td class="max-w-64 truncate">{item.key || '-'}</td>
                <td class="whitespace-nowrap">{formatDateTime(item.created_at)}</td>
                <td class="whitespace-nowrap">{formatDateTime(item.updated_at)}</td>
                <td>
                  <div class="flex flex-wrap gap-2">
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => void openDetails(item)}>{t('resourceScopes.details')}</button>
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openEdit(item)}>{t('common.edit')}</button>
                    <button class="btn preset-tonal-error btn-xs" type="button" on:click={() => void deleteScope(item)}>
                      {pendingDeleteId === item.id ? t('resourceScopes.deleteConfirm') : t('common.delete')}
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}

    <div class="flex flex-wrap items-center justify-between gap-3 border-t border-surface-200-800 p-3">
      <span class="text-xs text-surface-500">{t('dashboard.total')}: {total}</span>
      <div class="flex gap-2">
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={previousPage} disabled={offset === 0}>{t('common.previous')}</button>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={nextPage} disabled={offset + limit >= total}>{t('common.next')}</button>
      </div>
    </div>
  </section>
</section>

{#if dialogOpen}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="resource-scope-dialog-title">
    <div class="mx-auto mt-10 max-w-lg rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="resource-scope-dialog-title" class="font-semibold">{editing ? t('resourceScopes.editTitle') : t('resourceScopes.createTitle')}</h2>
        <button class="btn preset-outlined-surface-500" type="button" on:click={closeDialog}>{t('common.cancel')}</button>
      </div>

      <form class="space-y-4" on:submit|preventDefault={saveScope}>
        <label class="block">
          <span class="text-sm text-surface-500">{t('resourceScopes.type')}</span>
          <input class="input w-full" type="text" bind:value={formType} required disabled={!!editing} />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('resourceScopes.key')}</span>
          <input class="input w-full" type="text" bind:value={formKey} required disabled={!!editing} />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('resourceScopes.name')}</span>
          <input class="input w-full" type="text" bind:value={formName} required />
        </label>
        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeDialog}>{t('common.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={saving}>{saving ? t('common.loading') : t('common.save')}</button>
        </div>
      </form>
    </div>
  </div>
{/if}

{#if detailOpen && selectedScope}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="resource-scope-detail-title">
    <div class="mx-auto mt-10 max-w-2xl rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="resource-scope-detail-title" class="font-semibold">{t('resourceScopes.details')}</h2>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDetails}>{t('common.close')}</button>
      </div>

      {#if detailLoading}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {:else}
        <dl class="grid gap-3 text-sm md:grid-cols-2">
          <div>
            <dt class="text-surface-500">{t('resourceScopes.name')}</dt>
            <dd class="font-medium">{selectedScope.name}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('resourceScopes.type')}</dt>
            <dd class="font-medium">{selectedScope.type || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('resourceScopes.key')}</dt>
            <dd class="break-all font-medium">{selectedScope.key || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('resourceScopes.id')}</dt>
            <dd class="break-all font-medium">{selectedScope.id}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('resourceScopes.createdAt')}</dt>
            <dd class="font-medium">{formatDateTime(selectedScope.created_at)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('resourceScopes.updatedAt')}</dt>
            <dd class="font-medium">{formatDateTime(selectedScope.updated_at)}</dd>
          </div>
        </dl>
      {/if}
    </div>
  </div>
{/if}
