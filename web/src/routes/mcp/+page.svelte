<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, type MCPConnector } from '$lib/api';
  import { onMount } from 'svelte';

  let connectors: MCPConnector[] = [];
  let loading = false;
  let saving = false;
  let message = '';
  let error = '';
  let statusFilter: 'all' | MCPConnector['status'] = 'all';
  let authFilter: 'all' | MCPConnector['auth_type'] = 'all';
  let connectorSearch = '';
  let detailOpen = false;
  let selectedConnector: MCPConnector | null = null;

  let formOpen = false;
  let formName = '';
  let formEndpoint = '';
  let formAuth: MCPConnector['auth_type'] = 'none';
  let formCapability = '';
  let formEnabled = true;

  const authTypeLabel = (value: MCPConnector['auth_type']): string => {
    if (value === 'api_key') return t('mcp.authApiKey');
    if (value === 'bearer') return t('mcp.authBearer');
    if (value === 'basic') return t('mcp.authBasic');
    return t('mcp.authNone');
  };

  const statusLabel = (value: MCPConnector['status']): string => (value === 'active' ? t('mcp.enabled') : t('mcp.disabled'));

  const includesQuery = (value: unknown, query: string): boolean => String(value ?? '').toLowerCase().includes(query.trim().toLowerCase());

  const matchesConnectorSearch = (connector: MCPConnector, query: string): boolean => {
    if (!query.trim()) return true;
    return [
      connector.id,
      connector.name,
      connector.endpoint_url,
      connector.auth_type,
      authTypeLabel(connector.auth_type),
      connector.status,
      statusLabel(connector.status),
      connector.description,
    ].some((value) => includesQuery(value, query));
  };

  $: filteredConnectors = connectors.filter((connector) => {
    const statusMatches = statusFilter === 'all' || connector.status === statusFilter;
    const authMatches = authFilter === 'all' || connector.auth_type === authFilter;
    const searchMatches = matchesConnectorSearch(connector, connectorSearch);
    return statusMatches && authMatches && searchMatches;
  });
  $: activeConnectorCount = connectors.filter((connector) => connector.status === 'active').length;
  $: authenticatedConnectorCount = connectors.filter((connector) => connector.auth_type !== 'none').length;
  $: authTypeCount = new Set(connectors.map((connector) => connector.auth_type).filter(Boolean)).size;

  const loadData = async () => {
    loading = true;
    error = '';
    try {
      connectors = await api.listMCPConnectors();
    } catch {
      error = t('common.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const closeForm = () => {
    formOpen = false;
  };

  const resetFilters = () => {
    connectorSearch = '';
    statusFilter = 'all';
    authFilter = 'all';
  };

  const openDetails = (connector: MCPConnector) => {
    selectedConnector = connector;
    detailOpen = true;
  };

  const closeDetails = () => {
    selectedConnector = null;
    detailOpen = false;
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key !== 'Escape') return;
    if (formOpen) {
      closeForm();
    } else if (detailOpen) {
      closeDetails();
    }
  };

  const openForm = () => {
    message = '';
    error = '';
    formName = '';
    formEndpoint = '';
    formAuth = 'none';
    formCapability = '';
    formEnabled = true;
    formOpen = true;
  };

  const addConnector = async () => {
    message = '';
    error = '';
    if (formName.trim() === '') {
      error = t('mcp.nameRequired');
      return;
    }
    if (formEndpoint.trim() === '') {
      error = t('mcp.urlRequired');
      return;
    }
    try {
      new URL(formEndpoint);
    } catch {
      error = t('mcp.urlInvalid');
      return;
    }

    saving = true;
    try {
      const next = await api.createMCPConnector({
        name: formName.trim(),
        endpoint_url: formEndpoint.trim(),
        auth_type: formAuth,
        status: formEnabled ? 'active' : 'disabled',
        description: formCapability.trim(),
      });
      connectors = [...connectors, next];
      formOpen = false;
      message = t('mcp.addSuccess');
    } catch {
      error = t('mcp.addFailed');
    } finally {
      saving = false;
    }
  };

  onMount(() => {
    void loadData();
  });
</script>

<svelte:head>
  <title>{t('mcp.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <header class="flex items-center justify-between">
    <span aria-hidden="true"></span>
    <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openForm}>{t('mcp.addConnector')}</button>
  </header>

  <div class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-2">
    <p class="text-sm text-surface-500">{t('mcp.subtitle')}</p>
    <p class="text-sm">{t('mcp.outsideCore')}</p>
  </div>

  {#if message}
    <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
  {/if}
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_minmax(0,14rem)_minmax(0,14rem)_auto]" on:submit|preventDefault>
    <label class="block">
      <span class="text-sm text-surface-500">{t('mcp.search')}</span>
      <input class="input w-full" type="search" bind:value={connectorSearch} placeholder={t('mcp.searchPlaceholder')} />
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('mcp.statusFilter')}</span>
      <select class="input w-full" bind:value={statusFilter}>
        <option value="all">{t('common.all')}</option>
        <option value="active">{t('mcp.enabled')}</option>
        <option value="disabled">{t('mcp.disabled')}</option>
      </select>
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('mcp.authFilter')}</span>
      <select class="input w-full" bind:value={authFilter}>
        <option value="all">{t('common.all')}</option>
        <option value="none">{t('mcp.authNone')}</option>
        <option value="api_key">{t('mcp.authApiKey')}</option>
        <option value="bearer">{t('mcp.authBearer')}</option>
        <option value="basic">{t('mcp.authBasic')}</option>
      </select>
    </label>
    <div class="flex flex-wrap items-end gap-2">
      <button class="btn preset-filled-primary-500" type="submit">{t('common.filter')}</button>
      <button class="btn preset-outlined-surface-500" type="button" on:click={resetFilters}>{t('common.reset')}</button>
    </div>
  </form>

  <div class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
      <div class="grid gap-3 border-b border-surface-200-800 p-4 text-sm sm:grid-cols-4">
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredConnectors.length} / ${connectors.length}`}</p></article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('mcp.enabled')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeConnectorCount}</p></article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('mcp.authenticatedConnectors')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{authenticatedConnectorCount}</p></article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('mcp.authTypes')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{authTypeCount}</p></article>
      </div>

      {#if loading}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {:else if connectors.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('mcp.noConnectors')}</div>
      {:else if filteredConnectors.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('mcp.noFilteredConnectors')}</div>
      {:else}
        <div class="divide-y divide-surface-200-800 px-4">
          {#each filteredConnectors as connector (connector.id || connector.name)}
            <article class="py-3">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="space-y-1">
                  <h3 class="font-medium">{connector.name}</h3>
                  <p class="text-sm text-surface-500">{connector.endpoint_url}</p>
                  <p class="text-xs">{t('mcp.authType')}: {authTypeLabel(connector.auth_type)}</p>
                  <p class="text-xs">{t('mcp.workflowCapability')}: {connector.description || '-'}</p>
                  <p class="text-xs">{t('mcp.status')}: {statusLabel(connector.status)}</p>
                </div>
                <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openDetails(connector)}>{t('mcp.details')}</button>
              </div>
            </article>
          {/each}
        </div>
      {/if}
  </div>

  {#if formOpen}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="mcp-connector-dialog-title" tabindex="-1">
      <form class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-lg w-full overflow-y-auto p-4 space-y-3" on:submit|preventDefault={addConnector}>
        <h2 id="mcp-connector-dialog-title" class="font-semibold">{t('mcp.addConnector')}</h2>

        <label class="block">
          <span class="text-sm text-surface-500">{t('mcp.connectorName')}</span>
          <input class="input w-full" type="text" bind:value={formName} placeholder={t('mcp.connectorNamePlaceholder')} required />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('mcp.endpointUrl')}</span>
          <input class="input w-full" type="url" bind:value={formEndpoint} placeholder={t('mcp.endpointUrlPlaceholder')} required />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('mcp.authType')}</span>
          <select class="input w-full" bind:value={formAuth}>
            <option value="none">{t('mcp.authNone')}</option>
            <option value="api_key">{t('mcp.authApiKey')}</option>
            <option value="bearer">{t('mcp.authBearer')}</option>
            <option value="basic">{t('mcp.authBasic')}</option>
          </select>
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('mcp.workflowCapability')}</span>
          <textarea class="input w-full min-h-20" bind:value={formCapability} placeholder={t('mcp.workflowCapabilityPlaceholder')}></textarea>
        </label>

        <label class="flex items-center gap-2">
          <input type="checkbox" bind:checked={formEnabled} />
          <span>{t('mcp.enabled')}</span>
        </label>

        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeForm}>{t('mcp.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={saving}>{saving ? t('common.loading') : t('mcp.confirm')}</button>
        </div>
      </form>
    </div>
  {/if}

  {#if detailOpen && selectedConnector}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="mcp-detail-dialog-title" tabindex="-1">
      <div class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-lg w-full overflow-y-auto p-4 space-y-4">
        <header class="flex items-center justify-between gap-3">
          <h2 id="mcp-detail-dialog-title" class="font-semibold">{t('mcp.details')}</h2>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDetails}>{t('common.close')}</button>
        </header>

        <dl class="grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt class="text-surface-500">{t('mcp.connectorName')}</dt>
            <dd class="font-medium">{selectedConnector.name}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('mcp.status')}</dt>
            <dd class="font-medium">{statusLabel(selectedConnector.status)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('mcp.authType')}</dt>
            <dd class="font-medium">{authTypeLabel(selectedConnector.auth_type)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('mcp.connectorId')}</dt>
            <dd class="break-all font-medium">{selectedConnector.id || '-'}</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="text-surface-500">{t('mcp.endpointUrl')}</dt>
            <dd class="break-all font-medium">{selectedConnector.endpoint_url}</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="text-surface-500">{t('mcp.workflowCapability')}</dt>
            <dd class="whitespace-pre-wrap font-medium">{selectedConnector.description || '-'}</dd>
          </div>
        </dl>
      </div>
    </div>
  {/if}
</section>
