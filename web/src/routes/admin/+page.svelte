<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type DashboardSummary } from '$lib/api';

  let summary: DashboardSummary = {
    users: 0,
    active_users: 0,
    new_users: 0,
    application_activity: 0,
    pending_authorization: 0,
    sync_health: 'unknown',
  };
  let loading = true;
  let error = '';

  onMount(() => {
    api.dashboardSummary()
      .then((data) => {
        summary = { ...summary, ...data };
      })
      .catch((e) => {
        error = String(e || t('common.fetchFailed'));
      })
      .finally(() => {
        loading = false;
      });
  });

  const syncLabel = () => {
    if (summary.sync_health === 'ready') return t('dashboard.ready');
    if (summary.sync_health === 'error') return t('dashboard.error');
    return t('dashboard.unknown');
  };

  const syncBadgeClass = () => {
    if (summary.sync_health === 'ready') return 'preset-tonal-success';
    if (summary.sync_health === 'error') return 'preset-tonal-error';
    return 'preset-outlined-surface-500';
  };

  $: inactiveUsers = Math.max(summary.users - summary.active_users, 0);
  $: primaryMetrics = [
    { label: t('dashboard.users'), value: summary.users, href: '/admin/users' },
    { label: t('dashboard.activeUsers'), value: summary.active_users, href: '/admin/users' },
    { label: t('dashboard.newUsers'), value: summary.new_users, href: '/admin/users' },
    { label: t('dashboard.applicationActivity'), value: summary.application_activity, href: '/admin/applications' },
  ];
  $: taskMetrics = [
    { label: t('dashboard.syncStatus'), value: syncLabel(), hint: t('syncJobs.title'), href: '/admin/sync-jobs' },
    { label: t('dashboard.inactiveUsers'), value: inactiveUsers, hint: t('users.title'), href: '/admin/users' },
  ];
</script>

<svelte:head>
  <title>{t('dashboard.title')}</title>
</svelte:head>

{#if loading}
  <section class="card bg-surface-50-950 border border-surface-200-800 space-y-3 p-4" aria-label={t('common.loading')}>
    <div class="h-4 w-36 rounded bg-surface-200-800"></div>
    <div class="h-4 rounded bg-surface-200-800"></div>
    <div class="h-4 w-5/6 rounded bg-surface-200-800"></div>
    <div class="h-4 w-2/3 rounded bg-surface-200-800"></div>
  </section>
{:else}
  <div class="space-y-6">
    {#if error}
      <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
    {/if}

    <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
      {#each primaryMetrics as metric}
        <a class="card bg-surface-50-950 border border-surface-200-800 p-4 no-underline hover:border-primary-400" href={metric.href}>
          <p class="text-xs text-surface-600-400">{metric.label}</p>
          <p class="mt-3 text-2xl font-semibold tabular-nums text-surface-950-50">{metric.value}</p>
        </a>
      {/each}
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
        <p class="text-xs text-surface-600-400">{t('dashboard.syncHealth')}</p>
        <p class="mt-3"><span class={`badge ${syncBadgeClass()}`}>{syncLabel()}</span></p>
      </article>
    </section>

    <section class="card bg-surface-50-950 border border-surface-200-800 p-5">
      <div class="mb-3 flex items-center justify-between gap-3">
        <h2 class="font-semibold">{t('dashboard.dailyTasks')}</h2>
      </div>
      <div class="grid gap-3 md:grid-cols-2">
        {#each taskMetrics as metric}
          <a class="card bg-surface-50-950 border border-surface-200-800 p-4 no-underline hover:border-primary-400" href={metric.href}>
            <p class="text-sm font-medium text-surface-950-50">{metric.label}</p>
            <p class="mt-3 text-xl font-semibold tabular-nums text-surface-950-50">{metric.value}</p>
            <p class="mt-1 text-xs text-surface-500">{metric.hint}</p>
          </a>
        {/each}
      </div>
    </section>
  </div>
{/if}
