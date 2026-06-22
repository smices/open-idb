<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type SyncJob } from '$lib/api';
  import { t } from '$lib/i18n';

  let jobs: SyncJob[] = [];
  let total = 0;
  let limit = 25;
  let offset = 0;
  let statusFilter = '';
  let searchTerm = '';
  let loading = true;
  let error = '';
  let detailOpen = false;
  let selectedJob: SyncJob | null = null;

  const formatDateTime = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');
  const formatJson = (value: unknown): string => {
    if (!value) return '-';
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value);
    }
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
      const data = await api.listSyncJobs({ limit, offset });
      jobs = data.items || [];
      total = data.total || 0;
    } catch {
      error = t('syncJobs.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const previousPage = () => {
    if (offset === 0) return;
    offset = Math.max(0, offset - limit);
    void loadJobs();
  };

  const nextPage = () => {
    if (offset + limit >= total) return;
    offset += limit;
    void loadJobs();
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
    statusFilter = '';
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && detailOpen) {
      closeDetails();
    }
  };

  onMount(() => {
    void loadJobs();
  });

  $: filteredJobs = jobs.filter((job) => {
    const statusMatches = !statusFilter || job.status === statusFilter;
    return statusMatches && matchesSearch(job, searchTerm);
  });
  $: successCount = jobs.filter((job) => job.status === 'success').length;
  $: failedCount = jobs.filter((job) => job.status === 'failed').length;
  $: runningCount = jobs.filter((job) => job.status !== 'success' && job.status !== 'failed').length;
  $: pageStart = total === 0 ? 0 : offset + 1;
  $: pageEnd = Math.min(offset + limit, total);
</script>

<svelte:head>
  <title>{t('syncJobs.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <header class="flex flex-wrap items-end justify-between gap-3">
    <span aria-hidden="true"></span>
    <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void loadJobs()}>{t('common.retry')}</button>
  </header>

  <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_minmax(0,16rem)_auto]" on:submit|preventDefault>
    <label class="block">
      <span class="text-sm text-surface-500">{t('syncJobs.search')}</span>
      <input class="input w-full" type="search" bind:value={searchTerm} placeholder={t('syncJobs.searchPlaceholder')} />
    </label>
    <label class="block">
      <span class="text-sm text-surface-500">{t('syncJobs.status')}</span>
      <select class="input w-full" bind:value={statusFilter}>
        <option value="">{t('common.all')}</option>
        <option value="success">{t('syncJobs.status.success')}</option>
        <option value="failed">{t('syncJobs.status.failed')}</option>
        <option value="running">{t('syncJobs.status.running')}</option>
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

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <div class="grid gap-3 border-b border-surface-200-800 p-4 text-sm sm:grid-cols-5">
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('syncJobs.pageRange')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${pageStart}-${pageEnd}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('syncJobs.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{filteredJobs.length}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('syncJobs.status.success')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{successCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('syncJobs.status.failed')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{failedCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('syncJobs.status.running')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{runningCount}</p></article>
    </div>

    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if jobs.length === 0}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('syncJobs.noJobs')}</div>
    {:else if filteredJobs.length === 0}
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
            {#each filteredJobs as item (item.id)}
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
                  <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openDetails(item)}>{t('syncJobs.details')}</button>
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

{#if detailOpen && selectedJob}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="sync-job-dialog-title">
    <div class="mx-auto mt-10 max-w-3xl rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="sync-job-dialog-title" class="font-semibold">{t('syncJobs.details')}</h2>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDetails}>{t('common.close')}</button>
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
        <pre class="card bg-surface-100-900 border border-surface-200-800 overflow-x-auto p-3 text-xs mt-2"><code>{formatJson(selectedJob.stats)}</code></pre>
      </div>
    </div>
  </div>
{/if}
