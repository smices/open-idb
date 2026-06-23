<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { FileText, RotateCcw, Search, SlidersHorizontal, X } from 'lucide-svelte';
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
  const isSystemAction = (action: string): boolean => action.startsWith('sync.');
  const isAuthAction = (action: string): boolean => action.startsWith('auth.') || action.startsWith('sso.');
  const actorLabel = (log: AuditLogEntry): string => {
    if (log.actor_type === 'system' || log.actor_type === 'sync_job' || isSystemAction(log.action)) return t('audit.actor.system');
    if (log.actor_type === 'api_client') return t('audit.actor.apiClient');
    if (log.actor_user_id || isAuthAction(log.action)) return t('audit.actor.user');
    if (log.actor_type === 'admin' || !log.actor_user_id) return t('audit.actor.admin');
    return t('audit.actor.user');
  };
  const resourceTypeLabel = (value: string): string => t(`audit.resource.${value}`, value.replaceAll('_', ' '));
  const actionLabel = (value: string): string => t(`audit.action.${value}`, value);
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
  $: pageStart = total === 0 ? 0 : offset + 1;
  $: pageEnd = Math.min(offset + limit, total);
</script>

<svelte:head>
  <title>{t('audit.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <form class="flex flex-wrap items-center justify-between gap-2 border-b border-surface-200-800 p-3" on:submit|preventDefault={applyFilters}>
      <label class="relative min-w-56 flex-1 sm:max-w-sm">
        <span class="sr-only">{t('audit.search')}</span>
        <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
        <input class="input h-8 w-full pl-9 text-sm" type="search" bind:value={searchTerm} placeholder={t('audit.searchPlaceholder')} />
      </label>

      <div class="flex flex-wrap items-center gap-2">
        <label>
          <span class="sr-only">{t('audit.action')}</span>
          <input class="input h-8 w-32 text-sm" type="text" bind:value={actionFilter} placeholder={t('audit.action')} />
        </label>
        <label>
          <span class="sr-only">{t('audit.resourceType')}</span>
          <input class="input h-8 w-36 text-sm" type="text" bind:value={resourceTypeFilter} placeholder={t('audit.resourceType')} />
        </label>
        <label>
          <span class="sr-only">{t('audit.actorType')}</span>
          <select class="input h-8 w-32 text-sm" bind:value={actorTypeFilter}>
            <option value="">{t('common.all')}</option>
            <option value="user">{t('audit.actorUser')}</option>
            <option value="system">{t('audit.actorSystem')}</option>
          </select>
        </label>
        <button
          class="btn btn-xs preset-filled-primary-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0"
          type="submit"
          aria-label={t('audit.filter')}
          title={t('audit.filter')}
        >
          <SlidersHorizontal class="size-4" aria-hidden="true" />
        </button>
        <button
          class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0"
          type="button"
          on:click={resetFilters}
          aria-label={t('audit.reset')}
          title={t('audit.reset')}
        >
          <RotateCcw class="size-4" aria-hidden="true" />
        </button>
      </div>
    </form>

    {#if loading}
      <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if logs.length === 0}
      <div class="p-6 text-center text-sm text-surface-500">{t('audit.noLogs')}</div>
    {:else if filteredLogs.length === 0}
      <div class="p-6 text-center text-sm text-surface-500">{t('audit.noSearchResults')}</div>
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
                <td class="whitespace-nowrap">{actionLabel(item.action)}</td>
                <td>
                  <div class="space-y-1">
                    <div>{actorLabel(item)}</div>
                    {#if item.actor_user_id}
                      <div class="max-w-40 truncate font-mono text-[11px] text-surface-500">{item.actor_user_id}</div>
                    {/if}
                  </div>
                </td>
                <td class="whitespace-nowrap">{resourceTypeLabel(item.resource_type || '')}</td>
                <td class="max-w-40 truncate font-mono text-[11px] text-surface-500">{item.resource_id || '-'}</td>
                <td>{item.ip || '-'}</td>
                <td class="max-w-40 truncate font-mono text-[11px] text-surface-500">{item.trace_id || '-'}</td>
                <td>
                  <button
                    class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                    type="button"
                    on:click={() => openDetails(item)}
                    aria-label={t('audit.details')}
                    title={t('audit.details')}
                  >
                    <FileText class="size-4" aria-hidden="true" />
                  </button>
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
        <button
          class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
          type="button"
          on:click={closeDetails}
          aria-label={t('common.close')}
          title={t('common.close')}
        >
          <X class="size-4" aria-hidden="true" />
        </button>
      </div>

      <dl class="grid gap-3 text-sm md:grid-cols-2">
        <div>
          <dt class="text-surface-500">{t('audit.time')}</dt>
          <dd class="font-medium">{formatDateTime(selectedLog.created_at)}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.action')}</dt>
          <dd class="font-medium">{actionLabel(selectedLog.action)}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.actor')}</dt>
          <dd class="font-medium">
            {actorLabel(selectedLog)}
            {#if selectedLog.actor_user_id}
              <span class="ml-2 break-all font-mono text-[11px] text-surface-500">{selectedLog.actor_user_id}</span>
            {/if}
          </dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.resourceType')}</dt>
          <dd class="font-medium">{resourceTypeLabel(selectedLog.resource_type || '')}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.resourceId')}</dt>
          <dd class="break-all font-mono text-[11px] font-medium text-surface-600-400">{selectedLog.resource_id || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.ip')}</dt>
          <dd class="font-medium">{selectedLog.ip || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('audit.traceId')}</dt>
          <dd class="break-all font-mono text-[11px] font-medium text-surface-600-400">{selectedLog.trace_id || '-'}</dd>
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
