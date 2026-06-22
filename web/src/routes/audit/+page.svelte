<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type AuditLogEntry } from '$lib/api';
  import { t } from '$lib/i18n';

  let logs: AuditLogEntry[] = [];
  let total = 0;
  let limit = 25;
  let offset = 0;
  let actionFilter = '';
  let resourceTypeFilter = '';
  let actorTypeFilter = '';
  let searchTerm = '';
  let loading = true;
  let error = '';
  let detailOpen = false;
  let selectedLog: AuditLogEntry | null = null;

  const formatDateTime = (value: string): string => (value ? new Date(value).toLocaleString() : '-');
  const formatJson = (value: unknown): string => {
    if (!value) return '-';
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value);
    }
  };
  const matchesSearch = (log: AuditLogEntry, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [
      log.action,
      log.resource_type,
      log.resource_id,
      log.actor_type,
      log.actor_user_id,
      log.ip,
      log.trace_id,
      log.user_agent,
      log.id,
    ]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const loadLogs = async () => {
    loading = true;
    error = '';
    try {
      const data = await api.listAuditLogs({
        action: actionFilter,
        resource_type: resourceTypeFilter,
        actor_type: actorTypeFilter,
        limit,
        offset,
      });
      logs = data.items || [];
      total = data.total || 0;
    } catch {
      error = t('audit.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const applyFilters = () => {
    offset = 0;
    void loadLogs();
  };

  const resetFilters = () => {
    actionFilter = '';
    resourceTypeFilter = '';
    actorTypeFilter = '';
    searchTerm = '';
    offset = 0;
    void loadLogs();
  };

  const previousPage = () => {
    if (offset === 0) return;
    offset = Math.max(0, offset - limit);
    void loadLogs();
  };

  const nextPage = () => {
    if (offset + limit >= total) return;
    offset += limit;
    void loadLogs();
  };

  const openDetails = (log: AuditLogEntry) => {
    selectedLog = log;
    detailOpen = true;
  };

  const closeDetails = () => {
    detailOpen = false;
    selectedLog = null;
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && detailOpen) {
      closeDetails();
    }
  };

  onMount(() => {
    void loadLogs();
  });

  $: filteredLogs = logs.filter((log) => matchesSearch(log, searchTerm));
  $: userActorCount = logs.filter((log) => log.actor_type === 'user').length;
  $: systemActorCount = logs.filter((log) => log.actor_type === 'system').length;
  $: withTraceCount = logs.filter((log) => Boolean(log.trace_id)).length;
  $: pageStart = total === 0 ? 0 : offset + 1;
  $: pageEnd = Math.min(offset + limit, total);
</script>

<svelte:head>
  <title>{t('audit.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-5" on:submit|preventDefault={applyFilters}>
    <label class="block">
      <span class="text-sm text-surface-500">{t('audit.action')}</span>
      <input class="input w-full" type="text" bind:value={actionFilter} />
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('audit.resourceType')}</span>
      <input class="input w-full" type="text" bind:value={resourceTypeFilter} />
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('audit.actorType')}</span>
      <select class="input w-full" bind:value={actorTypeFilter}>
        <option value="">{t('common.all')}</option>
        <option value="user">{t('audit.actorUser')}</option>
        <option value="system">{t('audit.actorSystem')}</option>
      </select>
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('audit.search')}</span>
      <input class="input w-full" type="search" bind:value={searchTerm} placeholder={t('audit.searchPlaceholder')} />
    </label>
    <div class="flex flex-wrap items-end gap-2">
      <button class="btn preset-filled-primary-500 flex-1" type="submit">{t('audit.filter')}</button>
      <button class="btn preset-outlined-surface-500 flex-1" type="button" on:click={resetFilters}>{t('audit.reset')}</button>
    </div>
  </form>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <div class="grid gap-3 border-b border-surface-200-800 p-4 text-sm sm:grid-cols-5">
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('audit.pageRange')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${pageStart}-${pageEnd}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('audit.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{filteredLogs.length}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('audit.actorUser')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{userActorCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('audit.actorSystem')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{systemActorCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('audit.withTrace')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{withTraceCount}</p></article>
    </div>

    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if logs.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('audit.noLogs')}</div>
    {:else if filteredLogs.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('audit.noSearchResults')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th>{t('audit.time')}</th>
              <th>{t('audit.action')}</th>
              <th>{t('audit.actor')}</th>
              <th>{t('audit.resourceType')}</th>
              <th>{t('audit.resourceId')}</th>
              <th>{t('audit.ip')}</th>
              <th>{t('audit.traceId')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredLogs as item (item.id)}
              <tr>
                <td class="whitespace-nowrap">
                  <time datetime={item.created_at}>{formatDateTime(item.created_at)}</time>
                </td>
                <td>{item.action}</td>
                <td>
                  <div class="space-y-1">
                    <div>{item.actor_type || '-'}</div>
                    <div class="text-xs text-surface-500">{item.actor_user_id || '-'}</div>
                  </div>
                </td>
                <td>{item.resource_type || '-'}</td>
                <td class="max-w-48 truncate">{item.resource_id || '-'}</td>
                <td>{item.ip || '-'}</td>
                <td class="max-w-48 truncate">{item.trace_id || '-'}</td>
                <td>
                  <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openDetails(item)}>{t('audit.details')}</button>
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

{#if detailOpen && selectedLog}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="audit-dialog-title">
    <div class="mx-auto mt-10 max-w-4xl rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="audit-dialog-title" class="font-semibold">{t('audit.details')}</h2>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDetails}>{t('common.close')}</button>
      </div>

      <dl class="grid gap-3 text-sm md:grid-cols-2">
        <div>
          <dt class="text-surface-500">{t('audit.time')}</dt>
          <dd class="font-medium">{formatDateTime(selectedLog.created_at)}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.action')}</dt>
          <dd class="font-medium">{selectedLog.action}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.actor')}</dt>
          <dd class="break-all font-medium">{selectedLog.actor_type || '-'} / {selectedLog.actor_user_id || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.resourceType')}</dt>
          <dd class="font-medium">{selectedLog.resource_type || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.resourceId')}</dt>
          <dd class="break-all font-medium">{selectedLog.resource_id || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.ip')}</dt>
          <dd class="font-medium">{selectedLog.ip || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.traceId')}</dt>
          <dd class="break-all font-medium">{selectedLog.trace_id || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.userAgent')}</dt>
          <dd class="break-words font-medium">{selectedLog.user_agent || '-'}</dd>
        </div>
      </dl>

      <div class="mt-4 grid gap-4 md:grid-cols-2">
        <section>
          <h3 class="text-sm font-semibold">{t('audit.before')}</h3>
          <pre class="card bg-surface-100-900 border border-surface-200-800 overflow-x-auto p-3 text-xs mt-2"><code>{formatJson(selectedLog.before)}</code></pre>
        </section>
        <section>
          <h3 class="text-sm font-semibold">{t('audit.after')}</h3>
          <pre class="card bg-surface-100-900 border border-surface-200-800 overflow-x-auto p-3 text-xs mt-2"><code>{formatJson(selectedLog.after)}</code></pre>
        </section>
      </div>
    </div>
  </div>
{/if}
