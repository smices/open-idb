<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type DashboardSummary } from '$lib/api';
  import { authUser } from '$lib/stores';

  let summary: DashboardSummary = {
    users: 0,
    active_users: 0,
    new_users: 0,
    application_activity: 0,
    pending_authorization: 0,
    sync_health: 'unknown',
  };

  let loading = true;
  let error: string | null = null;
  type DashboardScope = 'user' | 'enterprise' | 'system';

  const dashboardScopeFromUrl = (): DashboardScope => {
    const scope = page.url.searchParams.get('scope');
    if (scope === 'user' || scope === 'system') return scope;
    return 'enterprise';
  };

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

  const syncBadgePreset = () => {
    if (summary.sync_health === 'ready') return 'preset-tonal-success';
    if (summary.sync_health === 'error') return 'preset-tonal-error';
    return 'preset-outlined-surface-500';
  };

  $: inactiveUsers = Math.max(summary.users - summary.active_users, 0);
  $: syncNeedsAttention = summary.sync_health === 'error' || summary.sync_health === 'unknown';
  $: actionNeeded = syncNeedsAttention || summary.pending_authorization > 0;
  $: requestedDashboardScope = dashboardScopeFromUrl();
  const canAccessScope = (scope: DashboardScope) => {
    const capabilities = $authUser?.capabilities;
    const consoleScope = $authUser?.console_scope;
    if (!capabilities?.length && !consoleScope) return true;
    if (scope === 'user') return true;
    if (scope === 'enterprise') return capabilities?.includes('enterprise') || consoleScope === 'enterprise_admin';
    return capabilities?.includes('system') || consoleScope === 'enterprise_admin';
  };
  $: dashboardScope = canAccessScope(requestedDashboardScope) ? requestedDashboardScope : canAccessScope('enterprise') ? 'enterprise' : 'user';
</script>

<svelte:head>
  <title>{t('dashboard.title')}</title>
</svelte:head>

{#if loading}
  <div class="card bg-surface-50-950 border border-surface-200-800 p-4 text-sm">{t('common.loading')}</div>
{:else}
  <div class="space-y-6">
    {#if error}
      <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
    {/if}

    <nav class="flex flex-wrap gap-2" aria-label={t('dashboard.scopeSwitch')}>
      {#if canAccessScope('user')}
        <a class={`btn btn-sm ${dashboardScope === 'user' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} href="/dashboard?scope=user">{t('dashboard.userTitle')}</a>
      {/if}
      {#if canAccessScope('enterprise')}
        <a class={`btn btn-sm ${dashboardScope === 'enterprise' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} href="/dashboard">{t('dashboard.enterpriseTitle')}</a>
      {/if}
      {#if canAccessScope('system')}
        <a class={`btn btn-sm ${dashboardScope === 'system' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} href="/dashboard?scope=system">{t('dashboard.systemTitle')}</a>
      {/if}
    </nav>

    {#if dashboardScope === 'user'}
      <section class="grid gap-4 md:grid-cols-3">
        <a class="card bg-surface-50-950 border border-surface-200-800 p-4 transition hover:border-primary-500" href="/profile">
          <p class="text-xs text-surface-500">{t('dashboard.userAccount')}</p>
          <p class="mt-2 text-xl font-semibold">{$authUser?.display_name || $authUser?.username || '-'}</p>
          <p class="mt-2 text-xs text-surface-500">{t('profile.title')}</p>
        </a>
        <a class="card bg-surface-50-950 border border-surface-200-800 p-4 transition hover:border-primary-500" href="/profile">
          <p class="text-xs text-surface-500">{t('dashboard.userSecurity')}</p>
          <p class="mt-2 text-xl font-semibold">{t('profile.changePassword')}</p>
          <p class="mt-2 text-xs text-surface-500">{t('dashboard.userSecurityHint')}</p>
        </a>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
          <p class="text-xs text-surface-500">{t('dashboard.userTenant')}</p>
          <p class="mt-2 break-all text-xl font-semibold">{$authUser?.entity_id || '-'}</p>
          <p class="mt-2 text-xs text-surface-500">{t('dashboard.userTenantHint')}</p>
        </article>
      </section>
    {:else if dashboardScope === 'system'}
      <section class="grid gap-4 md:grid-cols-3">
        <a class="card bg-surface-50-950 border border-surface-200-800 p-4 transition hover:border-primary-500" href="/entities">
          <p class="text-xs text-surface-500">{t('dashboard.systemTenants')}</p>
          <p class="mt-2 text-2xl font-semibold">{t('entities.title')}</p>
          <p class="mt-2 text-xs text-surface-500">{t('dashboard.systemTenantsHint')}</p>
        </a>
        <a class="card bg-surface-50-950 border border-surface-200-800 p-4 transition hover:border-primary-500" href="/resource-scopes">
          <p class="text-xs text-surface-500">{t('dashboard.systemCapabilityScopes')}</p>
          <p class="mt-2 text-2xl font-semibold">{t('resourceScopes.title')}</p>
          <p class="mt-2 text-xs text-surface-500">{t('dashboard.systemCapabilityHint')}</p>
        </a>
        <a class="card bg-surface-50-950 border border-surface-200-800 p-4 transition hover:border-primary-500" href="/mcp">
          <p class="text-xs text-surface-500">{t('dashboard.systemOpenConnectors')}</p>
          <p class="mt-2 text-2xl font-semibold">{t('mcp.title')}</p>
          <p class="mt-2 text-xs text-surface-500">{t('dashboard.systemOpenConnectorsHint')}</p>
        </a>
      </section>
    {:else}
      <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs text-surface-600">{t('dashboard.users')}</p>
            <span class="size-2.5 rounded-full bg-primary-500" aria-hidden="true"></span>
          </div>
          <p class="text-3xl font-semibold">{summary.users}</p>
        </article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs text-surface-600">{t('dashboard.activeUsers')}</p>
            <span class="size-2.5 rounded-full bg-green-500" aria-hidden="true"></span>
          </div>
          <p class="text-3xl font-semibold">{summary.active_users}</p>
        </article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs text-surface-600">{t('dashboard.newUsers')}</p>
            <span class="size-2.5 rounded-full bg-blue-500" aria-hidden="true"></span>
          </div>
          <p class="text-3xl font-semibold">{summary.new_users}</p>
        </article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs text-surface-600">{t('dashboard.applicationActivity')}</p>
            <span class="size-2.5 rounded-full bg-cyan-500" aria-hidden="true"></span>
          </div>
          <p class="text-3xl font-semibold">{summary.application_activity}</p>
        </article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs text-surface-600">{t('dashboard.pendingAuthorization')}</p>
            <span class="size-2.5 rounded-full bg-amber-500" aria-hidden="true"></span>
          </div>
          <p class="text-3xl font-semibold">{summary.pending_authorization}</p>
        </article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs text-surface-600">{t('dashboard.syncHealth')}</p>
            <span class="size-2.5 rounded-full bg-surface-500" aria-hidden="true"></span>
          </div>
          <p class="mt-2"><span class={`badge ${syncBadgePreset()}`}>{syncLabel()}</span></p>
        </article>
      </section>

      <section class="grid gap-4">
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
          <div class="mb-3 flex items-center justify-between gap-3">
            <h2 class="font-semibold">{t('dashboard.dailyTasks')}</h2>
            <span class={`badge ${actionNeeded ? 'preset-tonal-warning' : 'preset-tonal-success'}`}>
              {actionNeeded ? t('dashboard.needsAttention') : t('dashboard.noActionNeeded')}
            </span>
          </div>
          <div class="grid gap-3 md:grid-cols-3">
            <a class="card bg-surface-50-950 border border-surface-200-800 p-3 transition hover:border-primary-500" href="/application-assignments">
              <p class="text-xs text-surface-500">{t('dashboard.pendingAuth')}</p>
              <p class="mt-1 text-2xl font-semibold">{summary.pending_authorization}</p>
              <p class="mt-2 text-xs text-surface-500">{t('assignments.title')}</p>
            </a>
            <a class="card bg-surface-50-950 border border-surface-200-800 p-3 transition hover:border-primary-500" href="/sync-jobs">
              <p class="text-xs text-surface-500">{t('dashboard.syncStatus')}</p>
              <p class="mt-1 text-lg font-semibold">{syncLabel()}</p>
              <p class="mt-2 text-xs text-surface-500">{t('syncJobs.title')}</p>
            </a>
            <a class="card bg-surface-50-950 border border-surface-200-800 p-3 transition hover:border-primary-500" href="/users">
              <p class="text-xs text-surface-500">{t('dashboard.inactiveUsers')}</p>
              <p class="mt-1 text-2xl font-semibold">{inactiveUsers}</p>
              <p class="mt-2 text-xs text-surface-500">{t('users.title')}</p>
            </a>
          </div>
        </article>
      </section>
    {/if}
  </div>
{/if}
