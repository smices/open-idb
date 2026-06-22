<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type IMProviderConfig, type IdentitySource } from '$lib/api';

  let loading = false;
  let configs: IMProviderConfig[] = [];
  let sources: IdentitySource[] = [];

  let providerSource: IdentitySource | null = null;
  let config: IMProviderConfig | null = null;

  let appId = '';
  let appSecret = '';
  let enableSync = false;
  let detailOpen = false;
  let selectedConfig: IMProviderConfig | null = null;
  let saving = false;
  let syncing: 'full' | 'incremental' | null = null;
  let message = '';
  let error = '';
  let providerSearch = '';

  const providerStatusLabel = (value: string): string => {
    if (value === 'active') return t('identitySources.status.active');
    if (value === 'disabled') return t('identitySources.status.disabled');
    return value || t('dashboard.unknown');
  };

  const includesQuery = (value: unknown, query: string): boolean => String(value ?? '').toLowerCase().includes(query.trim().toLowerCase());

  const matchesProviderSearch = (item: IMProviderConfig, query: string): boolean => {
    if (!query.trim()) return true;
    return [
      item.id,
      item.provider,
      item.display_name,
      item.status,
      item.oauth_configured ? 'oauth' : '',
      item.sync_enabled ? 'sync' : '',
      JSON.stringify(item.config || {}),
    ].some((value) => includesQuery(value, query));
  };

  const openDetails = (item: IMProviderConfig) => {
    selectedConfig = item;
    detailOpen = true;
  };

  const closeDetails = () => {
    selectedConfig = null;
    detailOpen = false;
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && detailOpen) {
      closeDetails();
    }
  };

  const loadData = async () => {
    loading = true;
    error = '';
    try {
      const [providerList, sourceList] = await Promise.all([
        api.listIMProviderConfigs(),
        api.listIdentitySources({ limit: 200 }),
      ]);
      configs = providerList;
      sources = sourceList.items || sourceList.sources || [];
      const feishuConfig = providerList.find((item) => item.provider === 'feishu') || null;
      config = feishuConfig;
      providerSource = sources.find((item) => item.type === 'feishu') || null;

      appId = (feishuConfig?.config?.app_id as string) || '';
      appSecret = '';
      enableSync = config?.sync_enabled ?? providerSource?.sync_enabled ?? false;
    } catch {
      error = t('integrations.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const ensureFeishuSource = async (): Promise<IdentitySource> => {
    if (providerSource) {
      if (providerSource.sync_enabled !== enableSync) {
        providerSource = await api.updateIdentitySource(providerSource.id, {
          status: enableSync ? 'active' : 'disabled',
          sync_enabled: enableSync,
        });
      }
      return providerSource;
    }

    providerSource = await api.createIdentitySource({
      type: 'feishu',
      name: 'Feishu',
      sync_enabled: enableSync,
    });
    return providerSource;
  };

  const saveConfig = async () => {
    saving = true;
    error = '';
    message = '';
    const hasOAuthInput = appId !== '' || appSecret !== '';
    const oauthAlreadyConfigured = config?.oauth_configured ?? false;

    if (hasOAuthInput && (!appId || (!appSecret && !oauthAlreadyConfigured))) {
      error = t('integrations.feishuConfigIncomplete');
      saving = false;
      return;
    }

    try {
      const { bot_app_id: _botAppId, bot_app_secret: _botAppSecret, ...safeExistingConfig } = config?.config || {};
      const nextConfig = {
        ...safeExistingConfig,
        app_id: appId,
        ...(appSecret ? { app_secret: appSecret } : {}),
      };

      const saved = await api.upsertIMProviderConfig('feishu', {
        provider: 'feishu',
        display_name: 'Feishu',
        status: enableSync ? 'active' : 'disabled',
        sync_enabled: enableSync,
        oauth_configured: Boolean((appId && (appSecret || oauthAlreadyConfigured)) || oauthAlreadyConfigured),
        config: nextConfig,
      });

      config = saved;
      await ensureFeishuSource();
      message = t('integrations.saveSuccess');
    } catch {
      error = t('integrations.saveFailed');
    } finally {
      saving = false;
      await loadData();
    }
  };

  const runSync = async (mode: 'full' | 'incremental') => {
    syncing = mode;
    error = '';
    message = '';
    try {
      const currentSource = await ensureFeishuSource();
      await api.triggerSourceSync(currentSource.id, mode);
      message = mode === 'full' ? t('integrations.fullSyncStarted') : t('integrations.incrementalSyncStarted');
    } catch {
      error = t('integrations.syncFailed');
    } finally {
      syncing = null;
    }
  };

  onMount(loadData);

  $: filteredConfigs = configs.filter((item) => matchesProviderSearch(item, providerSearch));
  $: activeProviderCount = configs.filter((item) => item.status === 'active').length;
  $: readyProviderCount = configs.filter((item) => item.oauth_configured || item.sync_enabled).length;
</script>

<svelte:head>
  <title>{t('integrations.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <div class="grid gap-4 xl:grid-cols-2">
    <section class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-4">
      <h2 class="font-semibold">{t('integrations.provider.title')}</h2>

      {#if loading}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('integrations.loading')}</div>
      {:else}
        <form class="space-y-3" on:submit|preventDefault={saveConfig}>
          {#if message}
            <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
          {/if}
          {#if error}
            <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
          {/if}

          <div class="alert preset-tonal-primary text-sm">
            {t('integrations.feishuConfigNotice')}
          </div>

          <label class="block">
            <span class="text-sm text-surface-500">{t('integrations.appId')}</span>
            <input class="input w-full" type="text" bind:value={appId} placeholder={t('integrations.appIdPlaceholder')} />
          </label>
          <label class="block">
            <span class="text-sm text-surface-500">{t('integrations.appSecret')}</span>
            <input class="input w-full" type="password" bind:value={appSecret} placeholder={t('integrations.appSecretPlaceholder')} autocomplete="off" />
          </label>

          <label class="block">
            <span class="text-sm text-surface-500">{t('integrations.oauthRedirectUri')}</span>
            <input class="input w-full" type="url" value={''} disabled placeholder={t('integrations.oauthRedirectUriPlaceholder')} />
          </label>

          <label class="flex items-center gap-2">
            <input type="checkbox" bind:checked={enableSync} />
            <span class="text-sm">{t('integrations.enableSync')}</span>
          </label>

          <div class="flex gap-2">
            <button class="btn preset-filled-primary-500" disabled={saving} type="submit">
              {saving ? t('common.loading') : t('integrations.save')}
            </button>
          </div>
        </form>
      {/if}
    </section>

    <section class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3">
      <h2 class="font-semibold">{t('integrations.directorySync')}</h2>

      <div class="text-sm text-surface-500">{t('integrations.fullSyncDescription')}</div>
      <div class="text-sm text-surface-500">{t('integrations.incrementalSyncDescription')}</div>

      <dl class="grid gap-3 text-sm sm:grid-cols-2">
        <div>
          <dt class="text-surface-500">{t('identitySources.status')}</dt>
          <dd class="font-medium">{providerStatusLabel(providerSource?.status || config?.status || '')}</dd>
        </div>
        <div>
          <dt class="text-surface-500">{t('identitySources.syncEnabled')}</dt>
          <dd class="font-medium">{enableSync ? t('common.yes') : t('common.no')}</dd>
        </div>
        <div class="sm:col-span-2">
          <dt class="text-surface-500">{t('identitySources.id')}</dt>
          <dd class="break-all font-medium">{providerSource?.id || '-'}</dd>
        </div>
      </dl>

      <div class="flex flex-wrap gap-2">
        <button class="btn btn-sm preset-outlined-surface-500" type="button" disabled={syncing !== null} on:click={() => void runSync('full')}>
          {syncing === 'full' ? t('common.loading') : t('integrations.fullSync')}
        </button>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" disabled={syncing !== null} on:click={() => void runSync('incremental')}>
          {syncing === 'incremental' ? t('common.loading') : t('integrations.incrementalSync')}
        </button>
      </div>
    </section>

    <section class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-2 xl:col-span-2">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="font-semibold">{t('integrations.otherProviders')}</h2>
        <label class="block min-w-64">
          <span class="sr-only">{t('integrations.searchProviders')}</span>
          <input class="input w-full" type="search" bind:value={providerSearch} placeholder={t('integrations.searchProviders')} />
        </label>
      </div>
      <div class="text-sm text-surface-500">{t('integrations.dingtalk')} / {t('integrations.wecom')}: {t('integrations.comingSoon')}</div>

      <div class="grid gap-3 text-sm sm:grid-cols-3">
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredConfigs.length} / ${configs.length}`}</p></article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('integrations.activeProviders')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeProviderCount}</p></article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('integrations.readyProviders')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{readyProviderCount}</p></article>
      </div>

      {#if configs.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('integrations.noProviders')}</div>
      {:else if filteredConfigs.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('integrations.noProviderSearchResults')}</div>
      {:else}
        <div class="grid gap-3 md:grid-cols-2">
          {#each filteredConfigs as item (item.id || item.provider)}
            <article class="card bg-surface-50-950 border border-surface-200-800 p-3">
              <header class="flex items-start justify-between gap-3">
                <div>
                  <h3 class="font-medium">{item.display_name || item.provider}</h3>
                  <p class="text-xs text-surface-500">{item.provider}</p>
                </div>
                <span class={`badge ${item.status === 'active' ? 'preset-tonal-success' : 'preset-outlined-surface-500'}`}>{providerStatusLabel(item.status)}</span>
              </header>
              <dl class="mt-3 grid gap-2 text-xs sm:grid-cols-2">
                <div>
                  <dt class="text-surface-500">{t('integrations.oauthConfigured')}</dt>
                  <dd class="font-medium">{item.oauth_configured ? t('common.yes') : t('common.no')}</dd>
                </div>
                <div>
                  <dt class="text-surface-500">{t('identitySources.syncEnabled')}</dt>
                  <dd class="font-medium">{item.sync_enabled ? t('common.yes') : t('common.no')}</dd>
                </div>
              </dl>
              <div class="mt-3 flex justify-end">
                <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openDetails(item)}>{t('integrations.details')}</button>
              </div>
            </article>
          {/each}
        </div>
      {/if}
    </section>
  </div>

  {#if detailOpen && selectedConfig}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="integration-detail-dialog-title" tabindex="-1">
      <div class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-xl w-full overflow-y-auto p-4 space-y-4">
        <header class="flex items-center justify-between gap-3">
          <h2 id="integration-detail-dialog-title" class="font-semibold">{t('integrations.details')}</h2>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDetails}>{t('common.close')}</button>
        </header>

        <dl class="grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt class="text-surface-500">{t('integrations.providerName')}</dt>
            <dd class="font-medium">{selectedConfig.display_name || selectedConfig.provider}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('integrations.providerKey')}</dt>
            <dd class="font-medium">{selectedConfig.provider}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('identitySources.status')}</dt>
            <dd class="font-medium">{providerStatusLabel(selectedConfig.status)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('identitySources.syncEnabled')}</dt>
            <dd class="font-medium">{selectedConfig.sync_enabled ? t('common.yes') : t('common.no')}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('integrations.oauthConfigured')}</dt>
            <dd class="font-medium">{selectedConfig.oauth_configured ? t('common.yes') : t('common.no')}</dd>
          </div>
          <div class="sm:col-span-2">
            <dt class="text-surface-500">{t('integrations.safeConfigPreview')}</dt>
            <dd class="mt-1"><pre class="card bg-surface-100-900 border border-surface-200-800 overflow-x-auto p-3 text-xs font-mono"><code>{JSON.stringify(selectedConfig.config, null, 2)}</code></pre></dd>
          </div>
        </dl>
      </div>
    </div>
  {/if}
</section>
