<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { Copy, KeyRound, Plus, Settings, Trash2 } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { api, type Application, type OIDCClient } from '$lib/api';
  import IdConfirmDialog from '$lib/components/ui/IdConfirmDialog.svelte';
  import IdModal from '$lib/components/ui/IdModal.svelte';
  import { notifySuccess } from '$lib/toast';

  let apps: Application[] = [];
  let loading = true;
  let error = '';
  let pendingDeleteKey = '';

  let appName = '';
  let appType = 'oidc_client';
  let appStatus = 'active';
  let appEditingId = '';
  let dialogOpen = false;
  let saving = false;
  let oidcClient: OIDCClient | null = null;
  let oidcLoading = false;
  let oidcClientId = '';
  let oidcRedirectUris = '';
  let oidcScopes = 'openid\nprofile\nemail';
  let oidcGrantTypes = 'authorization_code';
  let oidcResponseTypes = 'code';
  let oidcPkce = true;
  let oidcStatus = 'active';
  let oidcClientSecret = '';
  let copiedValue = '';

  const applicationTypes = ['oidc_client', 'api_client', 'internal_app'];
  const applicationStatuses = ['active', 'disabled'];

  const applicationTypeLabel = (value: string): string => t(`applications.type.${value}`, value);
  const applicationStatusLabel = (value: string): string => t(`applications.status.${value}`, value);

  const formatDate = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');
  const copyIconLabel = (value?: string): string => (copiedValue === value ? t('common.copied') : t('common.copy'));
  const parseListField = (value: string): string[] =>
    value
      .split('\n')
      .flatMap((line) => line.split(','))
      .map((item) => item.trim())
      .filter(Boolean);

  const resetOIDCForm = () => {
    oidcClient = null;
    oidcClientId = '';
    oidcRedirectUris = '';
    oidcScopes = 'openid\nprofile\nemail';
    oidcGrantTypes = 'authorization_code';
    oidcResponseTypes = 'code';
    oidcPkce = true;
    oidcStatus = 'active';
    oidcClientSecret = '';
  };

  const loadOIDCConfig = async (applicationId: string) => {
    oidcLoading = true;
    resetOIDCForm();
    try {
      const data = await api.listOIDCClients({ limit: 200 });
      const client = (data.clients || []).find((item) => item.application_id === applicationId) || null;
      if (client) {
        const detail = await api.getOIDCClient(client.id);
        oidcClient = detail;
        oidcClientId = detail.client_id;
        oidcClientSecret = detail.client_secret || '';
        oidcRedirectUris = (detail.redirect_uris || []).join('\n');
        oidcScopes = (detail.allowed_scopes || ['openid', 'profile', 'email']).join('\n');
        oidcGrantTypes = (detail.grant_types || ['authorization_code']).join('\n');
        oidcResponseTypes = (detail.response_types || ['code']).join('\n');
        oidcPkce = detail.pkce_required !== false;
        oidcStatus = detail.status || 'active';
      }
    } catch {
      error = t('applications.fetchOidcFailed');
    } finally {
      oidcLoading = false;
    }
  };

  const loadApplications = async () => {
    loading = true;
    error = '';
    try {
      const data = await api.listApplications({ limit: 200 });
      apps = data.applications || [];
    } catch {
      error = t('applications.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const openCreateApp = () => {
    pendingDeleteKey = '';
    appEditingId = '';
    appName = '';
    appType = 'oidc_client';
    appStatus = 'active';
    resetOIDCForm();
    error = '';
    dialogOpen = true;
  };

  const openEditApp = async (item: Application) => {
    pendingDeleteKey = '';
    appEditingId = item.id;
    appName = item.name;
    appType = item.type;
    appStatus = item.status;
    resetOIDCForm();
    error = '';
    dialogOpen = true;
    if (item.type === 'oidc_client') {
      await loadOIDCConfig(item.id);
    }
  };

  const closeDialog = () => {
    dialogOpen = false;
    appEditingId = '';
  };

  const copyText = async (value?: string) => {
    if (!value || typeof navigator === 'undefined' || !navigator.clipboard) return;
    await navigator.clipboard.writeText(value);
    copiedValue = value;
    setTimeout(() => {
      if (copiedValue === value) copiedValue = '';
    }, 1200);
  };

  const saveApplication = async () => {
    const name = appName.trim();
    if (!name) return;
    const shouldSaveOIDC = appType === 'oidc_client';
    const redirectUris = parseListField(oidcRedirectUris);
    if (shouldSaveOIDC && redirectUris.length === 0) {
      error = t('applications.redirectUrisRequired');
      return;
    }

    saving = true;
    error = '';

    try {
      let savedApp: Application;
      if (appEditingId) {
        savedApp = await api.updateApplication(appEditingId, { name, status: appStatus });
      } else {
        savedApp = await api.createApplication({ name, type: appType });
      }

      if (shouldSaveOIDC) {
        const payload = {
          redirect_uris: redirectUris,
          allowed_scopes: parseListField(oidcScopes),
          grant_types: parseListField(oidcGrantTypes),
          response_types: parseListField(oidcResponseTypes),
          pkce_required: oidcPkce,
        };
        if (oidcClient) {
          await api.updateOIDCClient(oidcClient.id, {
            ...payload,
            status: oidcStatus,
          });
        } else {
          const result = await api.createOIDCClient({
            application_id: savedApp.id,
            redirect_uris: payload.redirect_uris,
          });
          oidcClient = result.client;
          oidcClientId = result.client.client_id;
          oidcClientSecret = result.client.client_secret || result.client_secret || '';
          oidcScopes = (result.client.allowed_scopes || ['openid', 'profile', 'email']).join('\n');
          oidcGrantTypes = (result.client.grant_types || ['authorization_code']).join('\n');
          oidcResponseTypes = (result.client.response_types || ['code']).join('\n');
          oidcPkce = result.client.pkce_required !== false;
          oidcStatus = result.client.status || 'active';
        }
      }
      notifySuccess(t('applications.saveSuccess'));
      if (shouldSaveOIDC) {
        appEditingId = savedApp.id;
        appStatus = savedApp.status;
      } else {
        dialogOpen = false;
      }
      await loadApplications();
    } catch {
      error = t('applications.saveFailed');
    } finally {
      saving = false;
    }
  };

  const deleteApplication = async (id: string) => {
    try {
      await api.deleteApplication(id);
      pendingDeleteKey = '';
      notifySuccess(t('applications.deleteSuccess'));
      await loadApplications();
    } catch {
      error = t('applications.deleteFailed');
    }
  };

  const rotateOIDCSecret = async () => {
    if (!oidcClient) return;
    saving = true;
    error = '';
    try {
      const result = await api.rotateOIDCClientSecret(oidcClient.id);
      oidcClient = result.client;
      oidcClientSecret = result.client.client_secret || result.client_secret || '';
      notifySuccess(t('applications.secretRotated'));
    } catch {
      error = t('applications.saveFailed');
    } finally {
      saving = false;
    }
  };

  onMount(() => {
    void loadApplications();
  });
</script>

<svelte:head>
  <title>{t('applications.title')}</title>
</svelte:head>

<section class="space-y-4">
  <div class="flex items-center justify-end">
    <button class="btn btn-sm preset-filled-primary-500 gap-1.5" type="button" on:click={openCreateApp}>
      <Plus class="size-4" aria-hidden="true" />
      {t('applications.create')}
    </button>
  </div>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    {#if loading}
      <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if !apps.length}
      <div class="p-6 text-sm text-surface-500">{t('common.noData')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th scope="col">{t('applications.name')}</th>
              <th scope="col">{t('applications.type')}</th>
              <th scope="col" class="w-20 !text-center">{t('applications.status')}</th>
              <th scope="col">{t('applications.updatedAt')}</th>
              <th scope="col" class="w-28 !text-right">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each apps as app}
              <tr>
                <td>
                  <p class="font-medium">{app.name}</p>
                  <p class="max-w-64 truncate text-xs text-surface-500">{app.id}</p>
                </td>
                <td class="whitespace-nowrap text-sm">{applicationTypeLabel(app.type)}</td>
                <td class="w-20 !text-center">
                  <span class="mx-auto flex size-5 items-center justify-center">
                    <span class={`size-2 rounded-full ${app.status === 'active' ? 'bg-success-500' : 'bg-error-500'}`} aria-hidden="true"></span>
                    <span class="sr-only">{applicationStatusLabel(app.status)}</span>
                  </span>
                </td>
                <td class="whitespace-nowrap text-sm text-surface-600-400">{formatDate(app.updated_at || app.created_at)}</td>
                <td class="!text-right">
                  <div class="relative inline-flex items-center gap-1">
                    <button
                      class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                      type="button"
                      on:click={() => void openEditApp(app)}
                      aria-label={t('applications.manage')}
                      title={t('applications.manage')}
                    >
                      <Settings class="size-4" aria-hidden="true" />
                    </button>
                    <IdConfirmDialog
                      open={pendingDeleteKey === `application:${app.id}`}
                      triggerLabel={t('common.delete')}
                      confirmLabel={t('common.confirmDelete')}
                      triggerClass="btn btn-xs preset-outlined-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                      onOpenChange={(open) => (pendingDeleteKey = open ? `application:${app.id}` : '')}
                      onConfirm={() => void deleteApplication(app.id)}
                    >
                      {#snippet trigger()}
                        <Trash2 class="size-4" aria-hidden="true" />
                      {/snippet}
                    </IdConfirmDialog>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </section>
</section>

<IdModal
  open={dialogOpen}
  title={appEditingId ? t('applications.editTitle') : t('applications.createTitle')}
  subtitle={appEditingId}
  maxWidth="max-w-3xl"
  onClose={closeDialog}
>
  <form id="application-dialog-form" class="space-y-4" on:submit|preventDefault={saveApplication}>
        <section class="grid gap-3 sm:grid-cols-2">
          <label class="block sm:col-span-2">
            <span class="text-xs text-surface-500">{t('applications.name')}</span>
            <input class="input h-8 w-full bg-surface-50-950 text-sm" type="text" bind:value={appName} required />
          </label>

          <label class="block">
            <span class="text-xs text-surface-500">{t('applications.type')}</span>
            <select class="input h-8 w-full bg-surface-50-950 text-sm" bind:value={appType} disabled={appEditingId !== ''}>
              {#each applicationTypes as type}
                <option value={type}>{applicationTypeLabel(type)}</option>
              {/each}
            </select>
          </label>

          {#if appEditingId}
            <label class="block">
              <span class="text-xs text-surface-500">{t('applications.status')}</span>
              <select class="input h-8 w-full bg-surface-50-950 text-sm" bind:value={appStatus}>
                {#each applicationStatuses as status}
                  <option value={status}>{applicationStatusLabel(status)}</option>
                {/each}
              </select>
            </label>
          {/if}
        </section>

        {#if appType === 'oidc_client'}
          <section class="space-y-3 border-t border-surface-200-800 pt-4">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold">{t('applications.oidcConfig')}</h3>
              {#if oidcClient}
                <button
                  class="btn btn-xs preset-outlined-surface-500 gap-1.5"
                  type="button"
                  on:click={() => void rotateOIDCSecret()}
                  disabled={saving}
                >
                  <KeyRound class="size-3" aria-hidden="true" />
                  {t('applications.rotateSecret')}
                </button>
              {/if}
            </div>
            {#if oidcLoading}
              <div class="p-3 text-sm text-surface-500">{t('common.loading')}</div>
            {:else}
              <div class="overflow-hidden rounded-container border border-surface-200-800 bg-surface-100-900">
                <table class="w-full table-fixed text-xs">
                  <tbody class="divide-y divide-surface-200-800">
                    <tr>
                      <th class="w-28 px-3 py-2 text-left font-medium text-surface-500" scope="row">{t('applications.clientId')}</th>
                      <td class="px-3 py-2">
                        <code class="block break-all font-mono {oidcClientId ? '' : 'text-surface-500'}">{oidcClientId || t('applications.generatedAfterSave')}</code>
                      </td>
                      <td class="w-10 px-2 py-2 text-right">
                        {#if oidcClientId}
                          <button
                            class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                            type="button"
                            on:click={() => void copyText(oidcClientId)}
                            aria-label={copyIconLabel(oidcClientId)}
                            title={copyIconLabel(oidcClientId)}
                          >
                            <Copy class="size-3" aria-hidden="true" />
                          </button>
                        {/if}
                      </td>
                    </tr>
                    <tr>
                      <th class="w-28 px-3 py-2 text-left font-medium text-surface-500" scope="row">{t('applications.clientSecret')}</th>
                      <td class="px-3 py-2">
                        <code class="block break-all font-mono {oidcClientSecret ? '' : 'text-surface-500'}">{oidcClientSecret || t('applications.generatedAfterSave')}</code>
                      </td>
                      <td class="w-10 px-2 py-2 text-right">
                        {#if oidcClientSecret}
                          <button
                            class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                            type="button"
                            on:click={() => void copyText(oidcClientSecret)}
                            aria-label={copyIconLabel(oidcClientSecret)}
                            title={copyIconLabel(oidcClientSecret)}
                          >
                            <Copy class="size-3" aria-hidden="true" />
                          </button>
                        {/if}
                      </td>
                    </tr>
                    <tr>
                      <th class="w-28 px-3 py-2 text-left font-medium text-surface-500" scope="row">{t('applications.scopes')}</th>
                      <td class="px-3 py-2" colspan="2">
                        <code class="block break-all font-mono">{oidcScopes.replaceAll('\n', ' ')}</code>
                      </td>
                    </tr>
                    <tr>
                      <th class="w-28 px-3 py-2 text-left font-medium text-surface-500" scope="row">{t('applications.grantTypes')}</th>
                      <td class="px-3 py-2" colspan="2">
                        <code class="block break-all font-mono">{oidcGrantTypes.replaceAll('\n', ' ')}</code>
                      </td>
                    </tr>
                    <tr>
                      <th class="w-28 px-3 py-2 text-left font-medium text-surface-500" scope="row">{t('applications.responseTypes')}</th>
                      <td class="px-3 py-2" colspan="2">
                        <code class="block break-all font-mono">{oidcResponseTypes.replaceAll('\n', ' ')}</code>
                      </td>
                    </tr>
                    <tr>
                      <th class="w-28 px-3 py-2 text-left font-medium text-surface-500" scope="row">{t('applications.pkce')}</th>
                      <td class="px-3 py-2 font-mono" colspan="2">{oidcPkce ? t('common.yes') : t('common.no')}</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <label class="block">
                <span class="text-xs text-surface-500">{t('applications.redirectUris')}</span>
                <textarea class="textarea w-full bg-surface-50-950 font-mono text-sm" rows="2" bind:value={oidcRedirectUris}></textarea>
              </label>

              {#if oidcClient}
                <label class="block">
                  <span class="text-xs text-surface-500">{t('applications.clientStatus')}</span>
                  <select class="input h-8 w-full bg-surface-50-950 text-sm" bind:value={oidcStatus}>
                    {#each applicationStatuses as status}
                      <option value={status}>{applicationStatusLabel(status)}</option>
                    {/each}
                  </select>
                </label>
              {/if}

            {/if}
          </section>
        {/if}
  </form>

  {#snippet footer()}
    <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDialog}>{t('common.cancel')}</button>
    <button class="btn btn-sm preset-filled-primary-500" type="submit" form="application-dialog-form" disabled={saving || oidcLoading || appName.trim() === ''}>
      {saving ? t('common.loading') : t('common.save')}
    </button>
  {/snippet}
</IdModal>
