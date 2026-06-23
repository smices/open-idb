<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { Check, Copy, KeyRound, Plus, Settings, Trash2, X } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { api, type Application, type OIDCClient } from '$lib/api';
  import Toast from '$lib/components/ui/Toast.svelte';

  let apps: Application[] = [];
  let loading = true;
  let message = '';
  let error = '';
  let pendingDeleteKey = '';

  let appName = '';
  let appType = 'oidc_client';
  let appStatus = 'active';
  let appEditingId = '';
  let drawerOpen = false;
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
    message = '';
    drawerOpen = true;
  };

  const openEditApp = async (item: Application) => {
    pendingDeleteKey = '';
    appEditingId = item.id;
    appName = item.name;
    appType = item.type;
    appStatus = item.status;
    resetOIDCForm();
    error = '';
    message = '';
    drawerOpen = true;
    if (item.type === 'oidc_client') {
      await loadOIDCConfig(item.id);
    }
  };

  const closeDrawer = () => {
    drawerOpen = false;
    appEditingId = '';
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && drawerOpen) {
      closeDrawer();
    }
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
    message = '';

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
      message = t('applications.saveSuccess');
      if (shouldSaveOIDC) {
        appEditingId = savedApp.id;
        appStatus = savedApp.status;
      } else {
        drawerOpen = false;
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
      message = t('applications.deleteSuccess');
      await loadApplications();
    } catch {
      error = t('applications.deleteFailed');
    }
  };

  const rotateOIDCSecret = async () => {
    if (!oidcClient) return;
    saving = true;
    error = '';
    message = '';
    try {
      const result = await api.rotateOIDCClientSecret(oidcClient.id);
      oidcClient = result.client;
      oidcClientSecret = result.client.client_secret || result.client_secret || '';
      message = t('applications.secretRotated');
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

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <div class="flex items-center justify-end">
    <button class="btn btn-sm preset-filled-primary-500 gap-1.5" type="button" on:click={openCreateApp}>
      <Plus class="size-4" aria-hidden="true" />
      {t('applications.create')}
    </button>
  </div>

  <Toast {message} />
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
                    <button
                      class="btn btn-xs preset-outlined-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                      type="button"
                      on:click={() => (pendingDeleteKey = `application:${app.id}`)}
                      aria-label={t('common.delete')}
                      title={t('common.delete')}
                    >
                      <Trash2 class="size-4" aria-hidden="true" />
                    </button>
                    {#if pendingDeleteKey === `application:${app.id}`}
                      <div class="absolute right-full top-1/2 z-10 mr-1 flex -translate-y-1/2 items-center gap-1 rounded-container border border-surface-200-800 bg-surface-50-950 p-1 shadow-lg">
                        <button
                          class="btn btn-xs preset-filled-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                          type="button"
                          on:click={() => void deleteApplication(app.id)}
                          aria-label={t('common.confirmDelete')}
                          title={t('common.confirmDelete')}
                        >
                          <Check class="size-4" aria-hidden="true" />
                        </button>
                        <button
                          class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                          type="button"
                          on:click={() => (pendingDeleteKey = '')}
                          aria-label={t('common.cancel')}
                          title={t('common.cancel')}
                        >
                          <X class="size-4" aria-hidden="true" />
                        </button>
                      </div>
                    {/if}
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

{#if drawerOpen}
  <div class="fixed inset-0 z-40 bg-surface-950/55 backdrop-blur-sm" aria-hidden="true" on:click={closeDrawer}></div>
  <div class="fixed inset-y-0 right-0 z-50 flex w-full justify-end" role="dialog" aria-modal="true" aria-labelledby="application-drawer-title" tabindex="-1">
    <form
      class="flex h-full w-full max-w-lg flex-col border-l border-surface-200-800 bg-surface-50-950 text-surface-950-50 shadow-2xl"
      on:submit|preventDefault={saveApplication}
    >
      <header class="flex items-center justify-between gap-3 border-b border-surface-200-800 px-5 py-4">
        <div>
          <h2 id="application-drawer-title" class="text-base font-semibold">{appEditingId ? t('applications.editTitle') : t('applications.createTitle')}</h2>
          {#if appEditingId}
            <p class="mt-1 max-w-72 truncate text-xs text-surface-500">{appEditingId}</p>
          {/if}
        </div>
        <button
          class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
          type="button"
          on:click={closeDrawer}
          aria-label={t('common.close')}
          title={t('common.close')}
        >
          <X class="size-4" aria-hidden="true" />
        </button>
      </header>

      <div class="flex-1 space-y-4 overflow-y-auto px-5 py-4">
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
      </div>

      <footer class="flex justify-end gap-2 border-t border-surface-200-800 bg-surface-100-900 px-5 py-4">
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDrawer}>{t('common.cancel')}</button>
        <button class="btn btn-sm preset-filled-primary-500" type="submit" disabled={saving || oidcLoading || appName.trim() === ''}>
          {saving ? t('common.loading') : t('common.save')}
        </button>
      </footer>
    </form>
  </div>
{/if}
