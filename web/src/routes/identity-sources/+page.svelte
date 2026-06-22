<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type IdentitySource } from '$lib/api';

  const identitySourceTypes = ['feishu', 'dingtalk', 'wecom', 'ldap', 'local'];

  let sources: IdentitySource[] = [];
  let loading = false;
  let formOpen = false;
  let editing: IdentitySource | null = null;
  let formType = 'feishu';
  let formName = '';
  let formStatus = 'active';
  let formSyncEnabled = false;
  let detailOpen = false;
  let detailLoading = false;
  let detailSource: IdentitySource | null = null;
  let saving = false;
  let syncingSourceId = '';
  let pendingDeleteId = '';
  let message = '';
  let error = '';
  let sourceSearch = '';
  let typeFilter = 'all';
  let statusFilter = 'all';

  const identitySourceTypeLabel = (value: string): string => t(`identitySources.type.${value}`, value);
  const identitySourceStatusLabel = (value: string): string => t(`identitySources.status.${value}`, value);
  const includesQuery = (value: unknown, query: string): boolean => String(value ?? '').toLowerCase().includes(query.trim().toLowerCase());

  const matchesSourceSearch = (item: IdentitySource, query: string): boolean => {
    if (!query.trim()) return true;
    return [
      item.id,
      item.entity_id,
      item.name,
      item.type,
      identitySourceTypeLabel(item.type),
      item.status,
      identitySourceStatusLabel(item.status),
      item.sync_enabled ? 'sync' : '',
    ].some((value) => includesQuery(value, query));
  };

  const fetchSources = async () => {
    loading = true;
    error = '';
    try {
      const data = await api.listIdentitySources({ limit: 200 });
      sources = data.items || data.sources || [];
    } catch {
      error = t('identitySources.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const openCreate = () => {
    pendingDeleteId = '';
    editing = null;
    formType = 'feishu';
    formName = '';
    formStatus = 'active';
    formSyncEnabled = false;
    formOpen = true;
  };

  const openEdit = (item: IdentitySource) => {
    pendingDeleteId = '';
    editing = item;
    formType = item.type;
    formName = item.name;
    formStatus = item.status;
    formSyncEnabled = item.sync_enabled;
    formOpen = true;
  };

  const closeForm = () => {
    formOpen = false;
  };

  const resetFilters = () => {
    sourceSearch = '';
    typeFilter = 'all';
    statusFilter = 'all';
  };

  const openDetails = async (item: IdentitySource) => {
    detailOpen = true;
    detailLoading = true;
    detailSource = item;
    error = '';
    try {
      detailSource = await api.getIdentitySource(item.id);
    } catch {
      error = t('identitySources.detailFetchFailed');
    } finally {
      detailLoading = false;
    }
  };

  const closeDetails = () => {
    detailOpen = false;
    detailSource = null;
    detailLoading = false;
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key !== 'Escape') return;
    if (formOpen) {
      closeForm();
    } else if (detailOpen) {
      closeDetails();
    }
  };

  const saveForm = async () => {
    saving = true;
    error = '';
    message = '';
    try {
      if (editing) {
        await api.updateIdentitySource(editing.id, {
          name: formName,
          status: formStatus,
          sync_enabled: formSyncEnabled,
        });
      } else {
        await api.createIdentitySource({
          type: formType,
          name: formName,
          sync_enabled: formSyncEnabled,
        });
      }

      message = t(editing ? 'common.updateSuccess' : 'common.createSuccess');
      formOpen = false;
      await fetchSources();
    } catch {
      error = t('identitySources.saveFailed');
    } finally {
      saving = false;
    }
  };

  const triggerSync = async (sourceId: string, mode: 'full' | 'incremental') => {
    syncingSourceId = sourceId;
    error = '';
    message = '';
    try {
      await api.triggerSourceSync(sourceId, mode);
      message = mode === 'full' ? t('integrations.fullSyncStarted') : t('integrations.incrementalSyncStarted');
    } catch {
      error = t('integrations.syncFailed');
    } finally {
      syncingSourceId = '';
    }
  };

  const removeSource = async (id: string) => {
    error = '';
    message = '';
    try {
      await api.deleteIdentitySource(id);
      pendingDeleteId = '';
      message = t('common.deleteSuccess');
      await fetchSources();
    } catch {
      error = t('common.deleteFailed');
    }
  };

  onMount(fetchSources);

  $: filteredSources = sources.filter((item) => {
    const searchMatches = matchesSourceSearch(item, sourceSearch);
    const typeMatches = typeFilter === 'all' || item.type === typeFilter;
    const statusMatches = statusFilter === 'all' || item.status === statusFilter;
    return searchMatches && typeMatches && statusMatches;
  });
  $: activeSourceCount = sources.filter((item) => item.status === 'active').length;
  $: syncEnabledSourceCount = sources.filter((item) => item.sync_enabled).length;
  $: sourceTypeCount = new Set(sources.map((item) => item.type).filter(Boolean)).size;
</script>

<svelte:head>
  <title>{t('identitySources.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <div class="flex items-center justify-end">
    <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openCreate}>{t('identitySources.create')}</button>
  </div>

  <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_minmax(0,14rem)_minmax(0,14rem)_auto]" on:submit|preventDefault>
    <label class="block">
      <span class="text-sm text-surface-500">{t('identitySources.search')}</span>
      <input class="input w-full" type="search" bind:value={sourceSearch} placeholder={t('identitySources.searchPlaceholder')} />
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('identitySources.type')}</span>
      <select class="input w-full" bind:value={typeFilter}>
        <option value="all">{t('common.all')}</option>
        {#each identitySourceTypes as type}
          <option value={type}>{identitySourceTypeLabel(type)}</option>
        {/each}
      </select>
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('identitySources.status')}</span>
      <select class="input w-full" bind:value={statusFilter}>
        <option value="all">{t('common.all')}</option>
        <option value="active">{t('identitySources.status.active')}</option>
        <option value="disabled">{t('identitySources.status.disabled')}</option>
        <option value="deleted">{t('identitySources.status.deleted')}</option>
      </select>
    </label>
    <div class="flex flex-wrap items-end gap-2">
      <button class="btn preset-filled-primary-500" type="submit">{t('common.filter')}</button>
      <button class="btn preset-outlined-surface-500" type="button" on:click={resetFilters}>{t('common.reset')}</button>
    </div>
  </form>

  {#if message}
    <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
  {/if}
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <div class="grid gap-3 border-b border-surface-200-800 p-4 text-sm sm:grid-cols-4">
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredSources.length} / ${sources.length}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('identitySources.status.active')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeSourceCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('identitySources.syncEnabled')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{syncEnabledSourceCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('identitySources.sourceTypes')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{sourceTypeCount}</p></article>
    </div>

    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if sources.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
    {:else if filteredSources.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('identitySources.noSearchResults')}</div>
    {:else}
      <div class="space-y-3 p-4">
        {#each filteredSources as item}
          <article class="card bg-surface-50-950 border border-surface-200-800 p-3">
            <header class="flex flex-wrap items-center justify-between gap-2">
              <div>
                <div class="font-medium">{item.name}</div>
                <div class="text-xs text-surface-500">
                  {identitySourceTypeLabel(item.type)} · {identitySourceStatusLabel(item.status)}
                </div>
              </div>
              <span class={`badge ${item.sync_enabled ? 'preset-tonal-success' : 'preset-outlined-surface-500'}`}>{item.sync_enabled ? t('identitySources.syncEnabled') : t('common.no')}</span>
            </header>
            <div class="mt-2 flex flex-wrap gap-2">
              <button class="btn preset-outlined-surface-500 btn-xs" type="button" disabled={syncingSourceId !== ''} on:click={() => void triggerSync(item.id, 'full')}>
                {syncingSourceId === item.id ? t('common.loading') : t('identitySources.triggerFull')}
              </button>
              <button class="btn preset-outlined-surface-500 btn-xs" type="button" disabled={syncingSourceId !== ''} on:click={() => void triggerSync(item.id, 'incremental')}>
                {syncingSourceId === item.id ? t('common.loading') : t('identitySources.triggerIncremental')}
              </button>
              <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => void openDetails(item)}>{t('identitySources.details')}</button>
              <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openEdit(item)}>{t('users.edit')}</button>
              <button
                class="btn preset-tonal-error btn-xs"
                type="button"
                on:click={() => (pendingDeleteId === item.id ? void removeSource(item.id) : (pendingDeleteId = item.id))}
              >
                {pendingDeleteId === item.id ? t('common.confirmDelete') : t('common.delete')}
              </button>
            </div>
          </article>
        {/each}
      </div>
    {/if}
  </section>

  {#if formOpen}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="identity-source-dialog-title" tabindex="-1">
      <form class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-lg w-full overflow-y-auto p-4 space-y-3" on:submit|preventDefault={saveForm}>
        <h2 id="identity-source-dialog-title" class="font-semibold">{editing ? t('identitySources.editTitle') : t('identitySources.createTitle')}</h2>

        <label class="block">
          <span class="text-sm text-surface-500">{t('identitySources.name')}</span>
          <input class="input w-full" type="text" bind:value={formName} required />
        </label>

        <label class="block">
          <span class="text-sm text-surface-500">{t('identitySources.type')}</span>
          <select class="input w-full" bind:value={formType} disabled={editing !== null}>
            {#each identitySourceTypes as type}
              <option value={type}>{identitySourceTypeLabel(type)}</option>
            {/each}
          </select>
        </label>

        {#if editing}
          <label class="block">
            <span class="text-sm text-surface-500">{t('identitySources.status')}</span>
            <select class="input w-full" bind:value={formStatus}>
              <option value="active">{t('identitySources.status.active')}</option>
              <option value="disabled">{t('identitySources.status.disabled')}</option>
              <option value="deleted">{t('identitySources.status.deleted')}</option>
            </select>
          </label>
        {/if}

        <label class="flex items-center gap-2">
          <input type="checkbox" bind:checked={formSyncEnabled} />
          <span class="text-sm">{t('identitySources.syncEnabled')}</span>
        </label>

        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeForm}>{t('common.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={saving || formName.trim() === ''}>
            {saving ? t('common.loading') : t('common.save')}
          </button>
        </div>
      </form>
    </div>
  {/if}

  {#if detailOpen && detailSource}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="identity-source-detail-title" tabindex="-1">
      <div class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-lg w-full overflow-y-auto p-4 space-y-4">
        <header class="flex items-center justify-between gap-3">
          <h2 id="identity-source-detail-title" class="font-semibold">{t('identitySources.details')}</h2>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDetails}>{t('common.close')}</button>
        </header>

        {#if detailLoading}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else}
          <dl class="grid gap-3 text-sm sm:grid-cols-2">
            <div>
              <dt class="text-surface-500">{t('identitySources.name')}</dt>
              <dd class="font-medium">{detailSource.name}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('identitySources.type')}</dt>
              <dd class="font-medium">{identitySourceTypeLabel(detailSource.type)}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('identitySources.status')}</dt>
              <dd class="font-medium">{identitySourceStatusLabel(detailSource.status)}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('identitySources.syncEnabled')}</dt>
              <dd class="font-medium">{detailSource.sync_enabled ? t('common.yes') : t('common.no')}</dd>
            </div>
            <div class="sm:col-span-2">
              <dt class="text-surface-500">{t('identitySources.id')}</dt>
              <dd class="break-all font-medium">{detailSource.id}</dd>
            </div>
            <div class="sm:col-span-2">
              <dt class="text-surface-500">{t('identitySources.entityId')}</dt>
              <dd class="break-all font-medium">{detailSource.entity_id}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('identitySources.createdAt')}</dt>
              <dd class="font-medium">{detailSource.created_at ? new Date(detailSource.created_at).toLocaleString() : '-'}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('identitySources.updatedAt')}</dt>
              <dd class="font-medium">{detailSource.updated_at ? new Date(detailSource.updated_at).toLocaleString() : '-'}</dd>
            </div>
          </dl>
        {/if}
      </div>
    </div>
  {/if}
</section>
