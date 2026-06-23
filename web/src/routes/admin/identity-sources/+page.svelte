<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type IMProviderConfig, type IdentitySource } from '$lib/api';
  import { Check, Trash2, X } from 'lucide-svelte';
  import Toast from '$lib/components/ui/Toast.svelte';

  let sources: IdentitySource[] = [];
  let config: IMProviderConfig | null = null;
  let providerSource: IdentitySource | null = null;
  let loading = true;
  let configSaving = false;
  let creatingSource = false;
  let providerSyncing: 'full' | 'incremental' | null = null;
  let pendingDeleteId = '';
  let message = '';
  let error = '';
  let appId = '';
  let appSecret = '';
  let enableSync = false;

  const identitySourceStatusLabel = (value: string): string => t(`identitySources.status.${value}`, value);

  const fetchSources = async () => {
    loading = true;
    error = '';
    try {
      const [data, providerList] = await Promise.all([
        api.listIdentitySources({ limit: 200 }),
        api.listIMProviderConfigs(),
      ]);
      sources = data.items || data.sources || [];
      config = providerList.find((item) => item.provider === 'feishu') || null;
      providerSource = sources.find((item) => item.type === 'feishu') || null;
      appId = (config?.config?.app_id as string) || '';
      appSecret = (config?.config?.app_secret as string) || '';
      enableSync = config?.sync_enabled ?? providerSource?.sync_enabled ?? false;
    } catch {
      error = t('identitySources.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const createFeishuSource = async () => {
    if (providerSource || creatingSource) return;
    creatingSource = true;
    error = '';
    message = '';
    try {
      await api.createIdentitySource({
        type: 'feishu',
        name: t('identitySources.feishuSourceName'),
        sync_enabled: false,
      });
      message = t('common.createSuccess');
      await fetchSources();
    } catch {
      error = t('identitySources.saveFailed');
    } finally {
      creatingSource = false;
    }
  };

  const ensureFeishuSource = async (): Promise<IdentitySource> => {
    if (!providerSource) {
      throw new Error(t('identitySources.feishuSourceRequired'));
    }
    if (providerSource.sync_enabled !== enableSync || providerSource.status !== (enableSync ? 'active' : 'disabled')) {
      providerSource = await api.updateIdentitySource(providerSource.id, {
        status: enableSync ? 'active' : 'disabled',
        sync_enabled: enableSync,
      });
    }
    return providerSource;
  };

  const saveFeishuConfig = async () => {
    configSaving = true;
    error = '';
    message = '';
    const hasOAuthInput = appId.trim() !== '' || appSecret.trim() !== '';
    const oauthAlreadyConfigured = config?.oauth_configured ?? false;

    if (!providerSource) {
      error = t('identitySources.feishuSourceRequired');
      configSaving = false;
      return;
    }

    if (hasOAuthInput && (!appId.trim() || (!appSecret.trim() && !oauthAlreadyConfigured))) {
      error = t('integrations.feishuConfigIncomplete');
      configSaving = false;
      return;
    }

    try {
      const { bot_app_id: _botAppId, bot_app_secret: _botAppSecret, ...safeExistingConfig } = config?.config || {};
      const nextConfig = {
        ...safeExistingConfig,
        app_id: appId.trim(),
        ...(appSecret.trim() ? { app_secret: appSecret.trim() } : {}),
      };

      config = await api.upsertIMProviderConfig('feishu', {
        provider: 'feishu',
        display_name: t('identitySources.type.feishu'),
        status: enableSync ? 'active' : 'disabled',
        sync_enabled: enableSync,
        oauth_configured: Boolean((appId.trim() && (appSecret.trim() || oauthAlreadyConfigured)) || oauthAlreadyConfigured),
        config: nextConfig,
      });
      await ensureFeishuSource();
      message = t('integrations.saveSuccess');
      await fetchSources();
    } catch {
      error = t('integrations.saveFailed');
    } finally {
      configSaving = false;
    }
  };

  const runProviderSync = async (mode: 'full' | 'incremental') => {
    providerSyncing = mode;
    error = '';
    message = '';
    try {
      const currentSource = await ensureFeishuSource();
      await api.triggerSourceSync(currentSource.id, mode);
      message = mode === 'full' ? t('integrations.fullSyncStarted') : t('integrations.incrementalSyncStarted');
    } catch {
      error = t('integrations.syncFailed');
    } finally {
      providerSyncing = null;
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

  const confirmRemoveCurrentSource = () => {
    if (!providerSource) return;
    pendingDeleteId = providerSource.id;
  };

  onMount(fetchSources);

  $: oauthConfigured = config?.oauth_configured ?? false;
</script>

<svelte:head>
  <title>{t('identitySources.title')}</title>
</svelte:head>

<section class="space-y-4">
  <Toast {message} />
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  {#if loading}
    <section class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</section>
  {:else if providerSource}
    <section class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_18rem]">
      <form class="card bg-surface-50-950 border border-surface-200-800 space-y-4 p-4" on:submit|preventDefault={saveFeishuConfig}>
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-surface-950-50">{t('identitySources.feishuConfigTitle')}</h2>
            <p class="mt-1 max-w-2xl text-sm leading-6 text-surface-600-400">{t('identitySources.feishuConfigDescription')}</p>
          </div>
          <span class={`badge ${oauthConfigured ? 'preset-tonal-success' : 'preset-outlined-surface-500'}`}>
            {oauthConfigured ? t('identitySources.oauthConfigured') : t('identitySources.oauthNotConfigured')}
          </span>
        </div>

        <div class="grid gap-3 md:grid-cols-2">
          <label class="block">
            <span class="text-sm text-surface-500">{t('integrations.appId')}</span>
            <input class="input w-full" type="text" bind:value={appId} placeholder={t('integrations.appIdPlaceholder')} />
          </label>
          <label class="block">
            <span class="text-sm text-surface-500">{t('integrations.appSecret')}</span>
            <input class="input w-full" type="text" bind:value={appSecret} placeholder={t('integrations.appSecretPlaceholder')} autocomplete="off" />
          </label>
        </div>

        <label class="block">
          <span class="text-sm text-surface-500">{t('integrations.oauthRedirectUri')}</span>
          <input class="input w-full" type="url" value={''} disabled placeholder={t('integrations.oauthRedirectUriPlaceholder')} />
        </label>

        <label class="flex items-center gap-2">
          <input type="checkbox" bind:checked={enableSync} />
          <span class="text-sm">{t('integrations.enableSync')}</span>
        </label>

        <div class="flex flex-wrap gap-2">
          <button class="btn btn-sm preset-filled-primary-500" disabled={configSaving} type="submit">
            {configSaving ? t('common.loading') : t('identitySources.saveFeishuConfig')}
          </button>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" disabled={providerSyncing !== null || configSaving} on:click={() => void runProviderSync('full')}>
            {providerSyncing === 'full' ? t('common.loading') : t('integrations.fullSync')}
          </button>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" disabled={providerSyncing !== null || configSaving} on:click={() => void runProviderSync('incremental')}>
            {providerSyncing === 'incremental' ? t('common.loading') : t('integrations.incrementalSync')}
          </button>
        </div>
      </form>

      <aside class="card bg-surface-50-950 border border-surface-200-800 self-start p-3">
        <div class="flex items-start justify-between gap-3">
          <div>
            <h2 class="text-sm font-semibold text-surface-950-50">{t('identitySources.configBoundaryTitle')}</h2>
            <p class="mt-1 text-xs leading-5 text-surface-600-400">{t('identitySources.configBoundaryDescription')}</p>
          </div>
          <div class="relative shrink-0">
            <button
              class="btn btn-xs preset-outlined-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
              type="button"
              on:click={confirmRemoveCurrentSource}
              aria-label={t('common.delete')}
              title={t('common.delete')}
            >
              <Trash2 class="size-4" aria-hidden="true" />
            </button>
            {#if pendingDeleteId === providerSource.id}
              <div class="absolute right-full top-1/2 z-10 mr-1 flex -translate-y-1/2 items-center gap-1 rounded-container border border-surface-200-800 bg-surface-50-950 p-1 shadow-lg">
                <button
                  class="btn btn-xs preset-filled-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                  type="button"
                  on:click={() => providerSource && void removeSource(providerSource.id)}
                  aria-label={t('common.confirmDelete')}
                  title={t('common.confirmDelete')}
                >
                  <Check class="size-4" aria-hidden="true" />
                </button>
                <button
                  class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                  type="button"
                  on:click={() => (pendingDeleteId = '')}
                  aria-label={t('common.cancel')}
                  title={t('common.cancel')}
                >
                  <X class="size-4" aria-hidden="true" />
                </button>
              </div>
            {/if}
          </div>
        </div>
        <dl class="mt-3 grid gap-2 text-xs">
          <div>
            <dt class="text-surface-500">{t('identitySources.type')}</dt>
            <dd class="font-medium">{t('identitySources.type.feishu')}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('identitySources.status')}</dt>
            <dd class="font-medium">{identitySourceStatusLabel(providerSource.status)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('identitySources.syncEnabled')}</dt>
            <dd class="font-medium">{enableSync ? t('common.yes') : t('common.no')}</dd>
          </div>
        </dl>
      </aside>
    </section>
  {:else}
    <section class="card bg-surface-50-950 border border-surface-200-800 p-5">
      <h2 class="text-base font-semibold text-surface-950-50">{t('identitySources.noFeishuSourceTitle')}</h2>
      <p class="mt-2 max-w-2xl text-sm leading-6 text-surface-600-400">{t('identitySources.noFeishuSourceDescription')}</p>
      <button class="btn btn-sm preset-filled-primary-500 mt-4" type="button" disabled={creatingSource} on:click={() => void createFeishuSource()}>
        {creatingSource ? t('common.loading') : t('identitySources.createFeishuSource')}
      </button>
    </section>
  {/if}
</section>
