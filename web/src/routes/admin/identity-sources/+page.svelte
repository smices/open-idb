<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type FeishuIdentitySourceConfig, type IdentitySource } from '$lib/api';
  import { Trash2 } from 'lucide-svelte';
  import { Switch } from '@skeletonlabs/skeleton-svelte';
  import IdConfirmDialog from '$lib/components/ui/IdConfirmDialog.svelte';
  import { notifyError, notifySuccess } from '$lib/toast';

  let sources: IdentitySource[] = [];
  let config: FeishuIdentitySourceConfig | null = null;
  let providerSource: IdentitySource | null = null;
  let loading = true;
  let configSaving = false;
  let creatingSource = false;
  let providerSyncing: 'full' | 'incremental' | null = null;
  let pendingDeleteId = '';
  let error = '';
  let appId = '';
  let appSecret = '';
  let enableSync = false;
  let oauthRedirectUri = '';

  const identitySourceStatusLabel = (value: string): string => t(`identitySources.status.${value}`, value);

  const fetchSources = async () => {
    loading = true;
    error = '';
    try {
      const data = await api.listIdentitySources({ limit: 200 });
      sources = data.items || data.sources || [];
      providerSource = sources.find((item) => item.type === 'feishu') || null;
    } catch {
      notifyError(t('identitySources.fetchFailed'));
      sources = [];
      providerSource = null;
    }

    try {
      config = await api.getFeishuIdentitySourceConfig();
      appId = (config?.config?.app_id as string) || '';
      appSecret = (config?.config?.app_secret as string) || '';
      enableSync = config?.sync_enabled ?? providerSource?.sync_enabled ?? false;
      oauthRedirectUri = config?.redirect_uri || '';
    } catch {
      config = null;
      appId = '';
      appSecret = '';
      enableSync = providerSource?.sync_enabled ?? false;
      oauthRedirectUri = '';
      notifyError(t('identitySources.configFetchFailed'));
    } finally {
      loading = false;
    }
  };

  const createFeishuSource = async () => {
    if (providerSource || creatingSource) return;
    creatingSource = true;
    error = '';
    try {
      await api.createIdentitySource({
        type: 'feishu',
        name: t('identitySources.feishuSourceName'),
        sync_enabled: false,
      });
      notifySuccess(t('common.createSuccess'));
      await fetchSources();
    } catch {
      notifyError(t('identitySources.saveFailed'));
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
    const hasOAuthInput = appId.trim() !== '' || appSecret.trim() !== '';
    const oauthAlreadyConfigured = config?.oauth_configured ?? false;

    if (!providerSource) {
      notifyError(t('identitySources.feishuSourceRequired'));
      configSaving = false;
      return;
    }

    if (hasOAuthInput && (!appId.trim() || (!appSecret.trim() && !oauthAlreadyConfigured))) {
      notifyError(t('identitySources.feishuConfigIncomplete'));
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

      config = await api.upsertFeishuIdentitySourceConfig({
        provider: 'feishu',
        display_name: t('identitySources.type.feishu'),
        status: enableSync ? 'active' : 'disabled',
        sync_enabled: enableSync,
        oauth_configured: Boolean((appId.trim() && (appSecret.trim() || oauthAlreadyConfigured)) || oauthAlreadyConfigured),
        config: nextConfig,
      });
      await ensureFeishuSource();
      notifySuccess(t('identitySources.configSaveSuccess'));
      await fetchSources();
    } catch {
      notifyError(t('identitySources.configSaveFailed'));
    } finally {
      configSaving = false;
    }
  };

  const runProviderSync = async (mode: 'full' | 'incremental') => {
    providerSyncing = mode;
    error = '';
    try {
      const currentSource = await ensureFeishuSource();
      await api.triggerSourceSync(currentSource.id, mode);
      notifySuccess(mode === 'full' ? t('identitySources.fullSyncStarted') : t('identitySources.incrementalSyncStarted'));
    } catch {
      notifyError(t('identitySources.syncFailed'));
    } finally {
      providerSyncing = null;
    }
  };

  const removeSource = async (id: string) => {
    error = '';
    try {
      await api.deleteIdentitySource(id);
      pendingDeleteId = '';
      notifySuccess(t('common.deleteSuccess'));
      await fetchSources();
    } catch {
      notifyError(t('common.deleteFailed'));
    }
  };

  onMount(() => {
    void fetchSources();
  });

  $: oauthConfigured = config?.oauth_configured ?? false;
</script>

<svelte:head>
  <title>{t('identitySources.title')}</title>
</svelte:head>

<section class="space-y-4">
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
            <span class="text-sm text-surface-500">{t('identitySources.appId')}</span>
            <input class="input w-full" type="text" bind:value={appId} placeholder={t('identitySources.appIdPlaceholder')} />
          </label>
          <label class="block">
            <span class="text-sm text-surface-500">{t('identitySources.appSecret')}</span>
            <input class="input w-full" type="text" bind:value={appSecret} placeholder={t('identitySources.appSecretPlaceholder')} autocomplete="off" />
          </label>
        </div>

        <label class="block">
          <span class="text-sm text-surface-500">{t('identitySources.oauthRedirectUri')}</span>
          <input class="input w-full font-mono text-sm" type="url" value={oauthRedirectUri} readonly />
        </label>

        <Switch checked={enableSync} onCheckedChange={(details) => (enableSync = details.checked)} class="inline-flex items-center gap-2">
          <Switch.HiddenInput />
          <Switch.Control class="relative inline-flex h-5 w-9 items-center rounded-full bg-surface-300-700 transition-colors data-[state=checked]:bg-primary-500">
            <Switch.Thumb class="block size-4 rounded-full bg-white shadow transition-transform data-[state=checked]:translate-x-4" />
          </Switch.Control>
          <Switch.Label class="text-sm">{t('identitySources.enableSync')}</Switch.Label>
        </Switch>

        <div class="flex flex-wrap gap-2">
          <button class="btn btn-sm preset-filled-primary-500" disabled={configSaving} type="submit">
            {configSaving ? t('common.loading') : t('identitySources.saveFeishuConfig')}
          </button>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" disabled={providerSyncing !== null || configSaving} on:click={() => void runProviderSync('full')}>
            {providerSyncing === 'full' ? t('common.loading') : t('identitySources.fullSync')}
          </button>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" disabled={providerSyncing !== null || configSaving} on:click={() => void runProviderSync('incremental')}>
            {providerSyncing === 'incremental' ? t('common.loading') : t('identitySources.incrementalSync')}
          </button>
        </div>
      </form>

      <aside class="card bg-surface-50-950 border border-surface-200-800 self-start p-3">
        <div class="flex items-start justify-between gap-3">
          <div>
            <h2 class="text-sm font-semibold text-surface-950-50">{t('identitySources.configBoundaryTitle')}</h2>
            <p class="mt-1 text-xs leading-5 text-surface-600-400">{t('identitySources.configBoundaryDescription')}</p>
          </div>
          <div class="shrink-0">
            <IdConfirmDialog
              open={pendingDeleteId === providerSource.id}
              triggerLabel={t('common.delete')}
              title={t('common.delete')}
              message={t('identitySources.deleteConfirm')}
              confirmLabel={t('common.confirmDelete')}
              triggerClass="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
              onOpenChange={(open) => (pendingDeleteId = open ? providerSource?.id || '' : '')}
              onConfirm={() => providerSource && void removeSource(providerSource.id)}
            >
              {#snippet trigger()}
                <Trash2 class="size-4" aria-hidden="true" />
              {/snippet}
            </IdConfirmDialog>
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
