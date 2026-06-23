<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type SyncJob } from '$lib/api';
  import { t } from '$lib/i18n';
  import { FileText, RotateCcw, Search, X } from 'lucide-svelte';

  let jobs: SyncJob[] = [];
  let offset = 0;
  let searchTerm = '';
  let loading = true;
  let error = '';
  let detailOpen = false;
  let selectedJob: SyncJob | null = null;
  const pageSize = 20;

  const formatDateTime = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');
  const parseStats = (value: unknown): Record<string, unknown> => {
    if (!value) return {};
    if (typeof value === 'object' && !Array.isArray(value)) return value as Record<string, unknown>;
    if (typeof value !== 'string') return {};
    const parseJson = (input: string): Record<string, unknown> => {
      try {
        const parsed = JSON.parse(input);
        return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed as Record<string, unknown> : {};
      } catch {
        return {};
      }
    };
    const direct = parseJson(value);
    if (Object.keys(direct).length > 0) return direct;
    if (typeof globalThis.atob !== 'function') return {};
    try {
      return parseJson(globalThis.atob(value));
    } catch {
      return {};
    }
  };
  const statLabel = (key: string): string => t(`syncJobs.stats.${key}`, key);
  const formatStatValue = (value: unknown): string => {
    if (typeof value === 'number') return value.toLocaleString();
    if (typeof value === 'string') return value;
    if (value === null || value === undefined) return '-';
    return JSON.stringify(value);
  };
  const matchesSearch = (job: SyncJob, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [job.provider, job.type, job.status, job.source_id, job.trace_id, job.error_message, job.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const loadJobs = async () => {
    loading = true;
    error = '';
    try {
      const data = await api.listSyncJobs({ limit: 2000, offset: 0 });
      jobs = data.items || [];
    } catch {
      error = t('syncJobs.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const previousPage = () => {
    if (offset === 0) return;
    offset = Math.max(0, offset - pageSize);
  };

  const nextPage = () => {
    if (offset + pageSize >= filteredLogJobs.length) return;
    offset += pageSize;
  };

  const openDetails = (job: SyncJob) => {
    selectedJob = job;
    detailOpen = true;
  };

  const closeDetails = () => {
    detailOpen = false;
    selectedJob = null;
  };

  const resetFilters = () => {
    searchTerm = '';
    offset = 0;
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && detailOpen) {
      closeDetails();
    }
  };

  onMount(() => {
    void loadJobs();
  });

  $: runningJobs = jobs.filter((job) => job.status !== 'success' && job.status !== 'succeeded' && job.status !== 'failed');
  $: logJobs = jobs.filter((job) => !runningJobs.includes(job));
  $: filteredLogJobs = logJobs.filter((job) => matchesSearch(job, searchTerm));
  $: pagedLogJobs = filteredLogJobs.slice(offset, offset + pageSize);
  $: selectedStats = parseStats(selectedJob?.stats);
  $: selectedStatEntries = Object.entries(selectedStats).filter(([key]) => key !== 'job_id');
  $: if (offset >= filteredLogJobs.length && offset !== 0) offset = Math.max(0, Math.floor((filteredLogJobs.length - 1) / pageSize) * pageSize);
</script>

<svelte:head>
  <title>{t('syncJobs.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <form
    class="card bg-surface-50-950 border border-surface-200-800 flex flex-wrap items-center justify-between gap-2 p-3"
    on:submit|preventDefault
    aria-label={t('syncJobs.title')}
  >
    <label class="relative min-w-56 flex-1 sm:max-w-md">
      <span class="sr-only">{t('syncJobs.search')}</span>
      <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
      <input class="input h-8 w-full pl-9 text-sm" type="search" bind:value={searchTerm} placeholder={t('syncJobs.searchPlaceholder')} />
    </label>

    <div class="flex flex-wrap items-center gap-2">
      <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void loadJobs()}>{t('common.retry')}</button>
      <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={resetFilters} aria-label={t('common.reset')}>
        <RotateCcw class="size-4" aria-hidden="true" />
      </button>
    </div>
  </form>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <div class="border-b border-surface-200-800 p-3">
      <h2 class="text-base font-semibold">{t('syncJobs.runningTasks')}</h2>
    </div>

    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if runningJobs.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('syncJobs.noRunningTasks')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th>{t('syncJobs.startedAt')}</th>
              <th>{t('syncJobs.provider')}</th>
              <th>{t('syncJobs.type')}</th>
              <th>{t('syncJobs.status')}</th>
              <th>{t('syncJobs.sourceId')}</th>
              <th>{t('syncJobs.finishedAt')}</th>
              <th>{t('syncJobs.traceId')}</th>
              <th>{t('syncJobs.error')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each runningJobs as item (item.id)}
              <tr>
                <td class="whitespace-nowrap">
                  <time datetime={item.started_at}>{formatDateTime(item.started_at)}</time>
                </td>
                <td>{item.provider || '-'}</td>
                <td>{item.type || '-'}</td>
                <td>
                  <span class={`badge ${item.status === 'success' ? 'preset-tonal-success' : item.status === 'failed' ? 'preset-tonal-error' : item.status === 'running' ? 'preset-tonal-warning' : 'preset-outlined-surface-500'}`}>{item.status || '-'}</span>
                </td>
                <td class="max-w-48 truncate">{item.source_id || '-'}</td>
                <td class="whitespace-nowrap">
                  <time datetime={item.finished_at || ''}>{formatDateTime(item.finished_at)}</time>
                </td>
                <td class="max-w-48 truncate">{item.trace_id || '-'}</td>
                <td class="max-w-64 truncate">{item.error_message || '-'}</td>
                <td>
                  <button
                    class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                    type="button"
                    on:click={() => openDetails(item)}
                    aria-label={t('syncJobs.details')}
                    title={t('syncJobs.details')}
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
  </section>

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-surface-200-800 p-3">
      <h2 class="text-base font-semibold">{t('syncJobs.logs')}</h2>
    </div>

    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if logJobs.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('syncJobs.noLogs')}</div>
    {:else if pagedLogJobs.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('syncJobs.noSearchResults')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th>{t('syncJobs.startedAt')}</th>
              <th>{t('syncJobs.provider')}</th>
              <th>{t('syncJobs.type')}</th>
              <th>{t('syncJobs.status')}</th>
              <th>{t('syncJobs.sourceId')}</th>
              <th>{t('syncJobs.finishedAt')}</th>
              <th>{t('syncJobs.traceId')}</th>
              <th>{t('syncJobs.error')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each pagedLogJobs as item (item.id)}
              <tr>
                <td class="whitespace-nowrap">
                  <time datetime={item.started_at}>{formatDateTime(item.started_at)}</time>
                </td>
                <td>{item.provider || '-'}</td>
                <td>{item.type || '-'}</td>
                <td>
                  <span class={`badge ${item.status === 'success' || item.status === 'succeeded' ? 'preset-tonal-success' : item.status === 'failed' ? 'preset-tonal-error' : 'preset-outlined-surface-500'}`}>{item.status || '-'}</span>
                </td>
                <td class="max-w-48 truncate">{item.source_id || '-'}</td>
                <td class="whitespace-nowrap">
                  <time datetime={item.finished_at || ''}>{formatDateTime(item.finished_at)}</time>
                </td>
                <td class="max-w-48 truncate">{item.trace_id || '-'}</td>
                <td class="max-w-64 truncate">{item.error_message || '-'}</td>
                <td>
                  <button
                    class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                    type="button"
                    on:click={() => openDetails(item)}
                    aria-label={t('syncJobs.details')}
                    title={t('syncJobs.details')}
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
      <span class="text-xs text-surface-500">{t('dashboard.total')}: {filteredLogJobs.length}</span>
      <div class="flex gap-2">
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={previousPage} disabled={offset === 0}>{t('common.previous')}</button>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={nextPage} disabled={offset + pageSize >= filteredLogJobs.length}>{t('common.next')}</button>
      </div>
    </div>
  </section>
</section>

{#if detailOpen && selectedJob}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="sync-job-dialog-title">
    <div class="mx-auto mt-10 max-w-3xl rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="sync-job-dialog-title" class="font-semibold">{t('syncJobs.details')}</h2>
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
          <dt class="text-surface-500">{t('syncJobs.provider')}</dt>
          <dd class="font-medium">{selectedJob.provider || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('syncJobs.type')}</dt>
          <dd class="font-medium">{selectedJob.type || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('syncJobs.status')}</dt>
          <dd class="font-medium">{selectedJob.status || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('syncJobs.sourceId')}</dt>
          <dd class="break-all font-medium">{selectedJob.source_id || '-'}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('syncJobs.startedAt')}</dt>
          <dd class="font-medium">{formatDateTime(selectedJob.started_at)}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('syncJobs.finishedAt')}</dt>
          <dd class="font-medium">{formatDateTime(selectedJob.finished_at)}</dd>
        </div>
        <div class="md:col-span-2">
          <dt class="text-surface-500">{t('syncJobs.traceId')}</dt>
          <dd class="break-all font-medium">{selectedJob.trace_id || '-'}</dd>
        </div>
        <div class="md:col-span-2">
          <dt class="text-surface-500">{t('syncJobs.error')}</dt>
          <dd class="break-words font-medium">{selectedJob.error_message || '-'}</dd>
        </div>
      </dl>

      <div class="mt-4">
        <h3 class="text-sm font-semibold">{t('syncJobs.stats')}</h3>
        {#if selectedStatEntries.length === 0}
          <div class="card bg-surface-50-950 border border-surface-200-800 mt-2 p-4 text-sm text-surface-500">{t('syncJobs.noStats')}</div>
        {:else}
          <dl class="mt-2 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {#each selectedStatEntries as [key, value]}
              <div class="card bg-surface-50-950 border border-surface-200-800 p-3">
                <dt class="text-xs text-surface-500">{statLabel(key)}</dt>
                <dd class="mt-1 text-xl font-semibold tabular-nums">{formatStatValue(value)}</dd>
              </div>
            {/each}
          </dl>
        {/if}
      </div>
    </div>
  </div>
{/if}
