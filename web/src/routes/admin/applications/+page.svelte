<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { Check, Edit3, FileText, KeyRound, Plus, Power, RotateCcw, Search, Settings, Trash2, X } from 'lucide-svelte';
  import { t, tf } from '$lib/i18n';
  import { api, type Application, type OIDCClient, type LegacyAppUser } from '$lib/api';
  import Toast from '$lib/components/ui/Toast.svelte';

  let apps: Application[] = [];
  let loading = true;
  let selectedApp: Application | null = null;
  let message = '';
  let error = '';
  let pendingDeleteKey = '';
  let appSearch = '';
  let oidcSearch = '';
  let legacySearch = '';

  let appName = '';
  let appType = 'oidc_client';
  let appStatus = 'active';
  let appEditingId = '';
  let appModalOpen = false;
  let appSaving = false;
  const applicationTypes = ['oidc_client', 'api_client', 'internal_app'];
  const applicationStatuses = ['active', 'disabled'];

  const applicationTypeLabel = (value: string): string => t(`applications.type.${value}`, value);
  const applicationStatusLabel = (value: string): string => t(`applications.status.${value}`, value);
  const matchesApplicationSearch = (app: Application, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [app.name, app.type, app.status, app.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const includesQuery = (value: unknown, query: string): boolean => String(value ?? '').toLowerCase().includes(query.trim().toLowerCase());

  const matchesOIDCSearch = (client: OIDCClient, query: string): boolean => {
    if (!query.trim()) return true;
    return [
      client.id,
      client.client_id,
      client.status,
      ...(client.redirect_uris || []),
      ...(client.allowed_scopes || []),
      ...(client.grant_types || []),
      ...(client.response_types || []),
    ].some((value) => includesQuery(value, query));
  };

  const matchesLegacySearch = (record: LegacyAppUser, query: string): boolean => {
    if (!query.trim()) return true;
    return [
      record.username,
      record.user_id,
      record.legacy_user_identifier,
      record.auth_scheme,
      record.id,
      record.is_active ? 'active' : 'disabled',
    ].some((value) => includesQuery(value, query));
  };

  const resetFilters = () => {
    appSearch = '';
  };

  let oidcClients: OIDCClient[] = [];
  let oidcLoading = false;
  let oidcModalOpen = false;
  let oidcSaving = false;
  let oidcFormClientId = '';
  let oidcRedirectUris = '';
  let oidcScopes = '';
  let oidcGrantTypes = 'authorization_code';
  let oidcResponseTypes = 'code';
  let oidcPkce = true;
  let oidcClientEditing: OIDCClient | null = null;
  let oidcDetailOpen = false;
  let oidcDetailLoading = false;
  let oidcDetail: OIDCClient | null = null;

  let legacyUsers: LegacyAppUser[] = [];
  let legacyLoading = false;
  let legacyModalOpen = false;
  let legacySaving = false;
  let legacyUsername = '';
  let legacyPassword = '';
  let legacyUserId = '';
  let legacyLegacyId = '';
  let legacyActive = true;
  let legacyEditing: LegacyAppUser | null = null;
  let legacyDetailOpen = false;
  let legacyDetailLoading = false;
  let legacyDetail: LegacyAppUser | null = null;

  const parseCommaField = (value: string): string[] =>
    value
      .split('\n')
      .flatMap((line) => line.split(','))
      .map((item) => item.trim())
      .filter(Boolean);

  const normalizeGrantTypes = (value: string): string[] =>
    value
      .split('\n')
      .map((item) => item.trim())
      .filter(Boolean);

  const loadApplications = async () => {
    loading = true;
    try {
      const data = await api.listApplications({ limit: 200 });
      apps = data.applications || [];
      if (selectedApp) {
        selectedApp = apps.find((item) => item.id === selectedApp?.id) || null;
        if (!selectedApp) {
          oidcClients = [];
          legacyUsers = [];
        }
      }
    } catch {
      error = t('applications.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const loadSelectedDetails = async () => {
    if (!selectedApp) {
      oidcClients = [];
      legacyUsers = [];
      return;
    }

    oidcLoading = true;
    legacyLoading = true;

    try {
      const [allOidc, allLegacy] = await Promise.all([
        api.listOIDCClients({ limit: 200 }),
        api.listLegacyUsers(selectedApp.id, { limit: 200 }),
      ]);

      oidcClients = (allOidc.clients || []).filter((item) => item.application_id === selectedApp!.id);
      legacyUsers = allLegacy.items || [];
    } catch {
      error = t('applications.fetchOidcFailed');
    } finally {
      oidcLoading = false;
      legacyLoading = false;
    }
  };

  const openCreateApp = () => {
    pendingDeleteKey = '';
    appEditingId = '';
    appName = '';
    appType = 'oidc_client';
    appStatus = 'active';
    appModalOpen = true;
  };

  const openEditApp = (item: Application) => {
    pendingDeleteKey = '';
    appEditingId = item.id;
    appName = item.name;
    appType = item.type;
    appStatus = item.status;
    appModalOpen = true;
  };

  const closeAppModal = () => {
    appModalOpen = false;
    appEditingId = '';
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key !== 'Escape') return;
    if (appModalOpen) {
      closeAppModal();
    } else if (oidcModalOpen) {
      closeOIDC();
    } else if (oidcDetailOpen) {
      closeOIDCDetail();
    } else if (legacyModalOpen) {
      closeLegacy();
    } else if (legacyDetailOpen) {
      closeLegacyDetail();
    }
  };

  const selectApplication = async (app: Application) => {
    selectedApp = app;
    oidcSearch = '';
    legacySearch = '';
    await loadSelectedDetails();
  };

  const saveApplication = async () => {
    appSaving = true;
    error = '';
    message = '';

    try {
      const name = appName.trim();
      if (!name) return;
      const saved = appEditingId
        ? await api.updateApplication(appEditingId, { name, status: appStatus })
        : await api.createApplication({ name, type: appType });
      message = t('applications.saveSuccess');
      appModalOpen = false;
      await loadApplications();
      selectedApp = apps.find((item) => item.id === saved.id) || saved;
      if (selectedApp) {
        await loadSelectedDetails();
      }
    } catch {
      error = t('applications.saveFailed');
    } finally {
      appSaving = false;
    }
  };

  const deleteApplication = async (id: string) => {
    try {
      await api.deleteApplication(id);
      pendingDeleteKey = '';
      if (selectedApp?.id === id) {
        selectedApp = null;
      }
      message = t('applications.deleteSuccess');
      await loadApplications();
    } catch {
      error = t('applications.deleteFailed');
    }
  };

  const openOIDCModal = () => {
    pendingDeleteKey = '';
    oidcModalOpen = true;
    oidcClientEditing = null;
    oidcFormClientId = '';
    oidcRedirectUris = '';
    oidcScopes = 'openid,profile,email';
    oidcGrantTypes = 'authorization_code';
    oidcResponseTypes = 'code';
    oidcPkce = true;
  };

  const selectOIDC = (client: OIDCClient) => {
    pendingDeleteKey = '';
    oidcModalOpen = true;
    oidcClientEditing = client;
    oidcFormClientId = client.client_id;
    oidcRedirectUris = (client.redirect_uris || []).join('\n');
    oidcScopes = (client.allowed_scopes || ['openid']).join('\n');
    oidcGrantTypes = (client.grant_types || ['authorization_code']).join('\n');
    oidcResponseTypes = (client.response_types || ['code']).join('\n');
    oidcPkce = Boolean(client.pkce_required);
  };

  const closeOIDC = () => {
    oidcModalOpen = false;
    oidcClientEditing = null;
    oidcFormClientId = '';
    oidcRedirectUris = '';
  };

  const openOIDCDetail = async (client: OIDCClient) => {
    oidcDetailOpen = true;
    oidcDetailLoading = true;
    oidcDetail = client;
    error = '';
    try {
      oidcDetail = await api.getOIDCClient(client.id);
    } catch {
      error = t('applications.fetchOidcFailed');
    } finally {
      oidcDetailLoading = false;
    }
  };

  const closeOIDCDetail = () => {
    oidcDetailOpen = false;
    oidcDetail = null;
    oidcDetailLoading = false;
  };

  const saveOIDC = async () => {
    if (!selectedApp) return;
    oidcSaving = true;
    error = '';
    message = '';

    const redirect_uris = parseCommaField(oidcRedirectUris);
    const payload = {
      application_id: selectedApp.id,
      client_id: oidcFormClientId,
      redirect_uris,
      allowed_scopes: parseCommaField(oidcScopes),
      grant_types: normalizeGrantTypes(oidcGrantTypes),
      response_types: normalizeGrantTypes(oidcResponseTypes),
      pkce_required: oidcPkce,
    };

    try {
      if (oidcClientEditing) {
        await api.updateOIDCClient(oidcClientEditing.id, payload);
      } else {
        const result = await api.createOIDCClient({
          ...payload,
          client_id: oidcFormClientId,
        });
        if (result.client_secret) {
          message = tf('applications.secretCreated', { secret: result.client_secret });
        }
      }
      if (!message) {
        message = t(oidcClientEditing ? 'applications.saveSuccess' : 'applications.oidcClientCreateSuccess');
      }
      await loadSelectedDetails();
      closeOIDC();
    } catch {
      error = t('applications.saveFailed');
    } finally {
      oidcSaving = false;
    }
  };

  const rotateSecret = async (client: OIDCClient) => {
    try {
      const result = await api.rotateOIDCClientSecret(client.id);
      message = tf('applications.secretCreated', { secret: result.client_secret || '' });
      await loadSelectedDetails();
    } catch {
      error = t('applications.saveFailed');
    }
  };

  const removeOIDC = async (client: OIDCClient) => {
    try {
      await api.deleteOIDCClient(client.id);
      pendingDeleteKey = '';
      message = t('applications.deleteSuccess');
      await loadSelectedDetails();
    } catch {
      error = t('applications.deleteFailed');
    }
  };

  const openLegacyCreate = () => {
    pendingDeleteKey = '';
    legacyModalOpen = true;
    legacyEditing = null;
    legacyUsername = '';
    legacyPassword = '';
    legacyUserId = '';
    legacyLegacyId = '';
    legacyActive = true;
  };

  const openLegacyEdit = (record: LegacyAppUser) => {
    pendingDeleteKey = '';
    legacyModalOpen = true;
    legacyEditing = record;
    legacyUsername = record.username;
    legacyPassword = '';
    legacyUserId = record.user_id;
    legacyLegacyId = record.legacy_user_identifier;
    legacyActive = record.is_active;
  };

  const closeLegacy = () => {
    legacyModalOpen = false;
    legacyEditing = null;
    legacyUsername = '';
    legacyPassword = '';
  };

  const openLegacyDetail = async (record: LegacyAppUser) => {
    if (!selectedApp) return;
    legacyDetailOpen = true;
    legacyDetailLoading = true;
    legacyDetail = record;
    error = '';
    try {
      legacyDetail = await api.getLegacyUser(selectedApp.id, record.username);
    } catch {
      error = t('applications.fetchLegacyFailed');
    } finally {
      legacyDetailLoading = false;
    }
  };

  const closeLegacyDetail = () => {
    legacyDetailOpen = false;
    legacyDetail = null;
    legacyDetailLoading = false;
  };

  const saveLegacy = async () => {
    if (!selectedApp) return;
    legacySaving = true;
    error = '';
    message = '';
    const wasEditing = legacyEditing !== null;

    try {
      if (legacyEditing) {
        await api.updateLegacyUser(selectedApp.id, legacyEditing.username, {
          user_id: legacyUserId,
          password: legacyPassword || undefined,
          legacy_user_identifier: legacyLegacyId,
          is_active: legacyActive,
        });
      } else {
        await api.createLegacyUser(selectedApp.id, {
          username: legacyUsername,
          user_id: legacyUserId,
          password: legacyPassword,
          legacy_user_identifier: legacyLegacyId,
          is_active: legacyActive,
        });
      }

      legacyEditing = null;
      legacyModalOpen = false;
      message = t(wasEditing ? 'applications.legacySaveSuccess' : 'applications.legacyCreateSuccess');
      await loadSelectedDetails();
      legacyUsername = '';
      legacyPassword = '';
    } catch {
      error = t('applications.saveFailed');
    } finally {
      legacySaving = false;
    }
  };

  const removeLegacy = async (record: LegacyAppUser) => {
    if (!selectedApp) return;

    try {
      await api.deleteLegacyUser(selectedApp.id, record.username);
      pendingDeleteKey = '';
      message = t('applications.legacyDeleted');
      await loadSelectedDetails();
    } catch {
      error = t('applications.saveFailed');
    }
  };

  const toggleLegacyActive = async (record: LegacyAppUser, active: boolean) => {
    if (!selectedApp) return;
    try {
      await api.setLegacyUserStatus(selectedApp.id, record.username, active);
      message = t(active ? 'applications.legacyEnable' : 'applications.legacyDisable');
      await loadSelectedDetails();
    } catch {
      error = t('applications.saveFailed');
    }
  };

  onMount(() => {
    void loadApplications();
  });

  $: filteredApps = apps.filter((app) => matchesApplicationSearch(app, appSearch));
  $: filteredOIDCClients = oidcClients.filter((client) => matchesOIDCSearch(client, oidcSearch));
  $: filteredLegacyUsers = legacyUsers.filter((record) => matchesLegacySearch(record, legacySearch));
</script>

<svelte:head>
  <title>{t('applications.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <Toast {message} />
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <div class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <form class="flex flex-wrap items-center justify-between gap-2 border-b border-surface-200-800 p-3" on:submit|preventDefault>
      <label class="relative min-w-56 flex-1 sm:max-w-md">
        <span class="sr-only">{t('applications.search')}</span>
        <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
        <input class="input h-8 w-full pl-9 text-sm" type="search" bind:value={appSearch} placeholder={t('applications.searchPlaceholder')} />
      </label>
      <div class="flex flex-wrap items-center gap-2">
        <button
          class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0"
          type="button"
          on:click={resetFilters}
          aria-label={t('common.reset')}
          title={t('common.reset')}
        >
          <RotateCcw class="size-4" aria-hidden="true" />
        </button>
        <button
          class="btn btn-xs preset-filled-primary-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0"
          type="button"
          on:click={openCreateApp}
          aria-label={t('applications.create')}
          title={t('applications.create')}
        >
          <Plus class="size-4" aria-hidden="true" />
        </button>
      </div>
    </form>

    {#if loading}
      <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if !apps.length}
      <div class="p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
    {:else if !filteredApps.length}
      <div class="p-6 text-center text-sm text-surface-500">{t('applications.noSearchResults')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th>{t('applications.name')}</th>
              <th>{t('applications.type')}</th>
              <th class="w-20 !text-center">{t('applications.status')}</th>
              <th>{t('applications.createdAt')}</th>
              <th>{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredApps as app}
              <tr class={selectedApp?.id === app.id ? 'bg-primary-50-950/40' : ''}>
                <td>
                  <div class="font-medium">{app.name}</div>
                  <div class="max-w-56 truncate text-xs text-surface-500">{app.id}</div>
                </td>
                <td>{applicationTypeLabel(app.type)}</td>
                <td class="w-20 !text-center">
                  <span class="mx-auto flex size-5 items-center justify-center">
                    <span class={`size-2 rounded-full ${app.status === 'active' ? 'bg-success-500' : 'bg-error-500'}`} aria-hidden="true"></span>
                    <span class="sr-only">{applicationStatusLabel(app.status)}</span>
                  </span>
                </td>
                <td class="whitespace-nowrap">{app.created_at ? new Date(app.created_at).toLocaleString() : '-'}</td>
                <td>
                  <div class="relative flex items-center gap-1">
                    <button
                      class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                      type="button"
                      on:click={() => void selectApplication(app)}
                      aria-label={t('applications.manage')}
                      title={t('applications.manage')}
                    >
                      <Settings class="size-4" aria-hidden="true" />
                    </button>
                    <button
                      class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                      type="button"
                      on:click={() => openEditApp(app)}
                      aria-label={t('users.edit')}
                      title={t('users.edit')}
                    >
                      <Edit3 class="size-4" aria-hidden="true" />
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
  </div>

  <div class="grid gap-4 xl:grid-cols-2">
    <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
      <div class="flex items-center justify-between gap-3 border-b border-surface-200-800 p-3">
        <h2 class="text-base font-semibold">{t('applications.oidcManagement')}</h2>
        {#if selectedApp}
          <button
            class="btn btn-xs preset-filled-primary-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0"
            type="button"
            on:click={openOIDCModal}
            aria-label={t('applications.oidc')}
            title={t('applications.oidc')}
          >
            <Plus class="size-4" aria-hidden="true" />
          </button>
        {/if}
      </div>
      {#if selectedApp}
        {#if oidcLoading}
          <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else}
          <div class="border-b border-surface-200-800 p-3">
            <label class="relative block max-w-sm">
              <span class="sr-only">{t('applications.searchOidc')}</span>
              <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
              <input class="input h-8 w-full pl-9 text-sm" type="search" bind:value={oidcSearch} placeholder={t('applications.searchOidc')} />
            </label>
          </div>
          {#if filteredOIDCClients.length === 0}
            <div class="p-6 text-center text-sm text-surface-500">{oidcClients.length ? t('applications.noOidcSearchResults') : t('common.noData')}</div>
          {:else}
            <div class="overflow-x-auto">
              <table class="table min-w-full">
                <thead>
                  <tr>
                    <th>{t('applications.clientId')}</th>
                    <th>{t('applications.redirectUris')}</th>
                    <th class="w-20 !text-center">{t('applications.pkce')}</th>
                    <th>{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {#each filteredOIDCClients as item}
                    <tr>
                      <td>
                        <div class="font-medium">{item.client_id}</div>
                        <div class="max-w-48 truncate text-xs text-surface-500">{item.allowed_scopes?.join(', ') || '-'}</div>
                      </td>
                      <td class="max-w-72 truncate text-xs text-surface-500">{item.redirect_uris?.join(', ') || '-'}</td>
                      <td class="w-20 !text-center">{item.pkce_required ? t('common.yes') : t('common.no')}</td>
                      <td>
                        <div class="relative flex items-center gap-1">
                          <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => void openOIDCDetail(item)} aria-label={t('applications.oidcDetails')} title={t('applications.oidcDetails')}>
                            <FileText class="size-4" aria-hidden="true" />
                          </button>
                          <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => selectOIDC(item)} aria-label={t('common.update')} title={t('common.update')}>
                            <Edit3 class="size-4" aria-hidden="true" />
                          </button>
                          <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => void rotateSecret(item)} aria-label={t('applications.rotateSecret')} title={t('applications.rotateSecret')}>
                            <KeyRound class="size-4" aria-hidden="true" />
                          </button>
                          <button class="btn btn-xs preset-outlined-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => (pendingDeleteKey = `oidc:${item.id}`)} aria-label={t('common.delete')} title={t('common.delete')}>
                            <Trash2 class="size-4" aria-hidden="true" />
                          </button>
                          {#if pendingDeleteKey === `oidc:${item.id}`}
                            <div class="absolute right-full top-1/2 z-10 mr-1 flex -translate-y-1/2 items-center gap-1 rounded-container border border-surface-200-800 bg-surface-50-950 p-1 shadow-lg">
                              <button class="btn btn-xs preset-filled-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => void removeOIDC(item)} aria-label={t('common.confirmDelete')} title={t('common.confirmDelete')}>
                                <Check class="size-4" aria-hidden="true" />
                              </button>
                              <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => (pendingDeleteKey = '')} aria-label={t('common.cancel')} title={t('common.cancel')}>
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
        {/if}
      {:else}
        <div class="p-6 text-center text-sm text-surface-500">{t('applications.selectApp')}</div>
      {/if}
    </section>

    <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
      <div class="flex items-center justify-between gap-3 border-b border-surface-200-800 p-3">
        <h2 class="text-base font-semibold">{t('applications.legacyManagement')}</h2>
        {#if selectedApp}
          <button
            class="btn btn-xs preset-filled-primary-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0"
            type="button"
            on:click={openLegacyCreate}
            aria-label={t('applications.createLegacyUser')}
            title={t('applications.createLegacyUser')}
          >
            <Plus class="size-4" aria-hidden="true" />
          </button>
        {/if}
      </div>
      {#if selectedApp}
        {#if legacyLoading}
          <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else}
          <div class="border-b border-surface-200-800 p-3">
            <label class="relative block max-w-sm">
              <span class="sr-only">{t('applications.searchLegacy')}</span>
              <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
              <input class="input h-8 w-full pl-9 text-sm" type="search" bind:value={legacySearch} placeholder={t('applications.searchLegacy')} />
            </label>
          </div>
          {#if filteredLegacyUsers.length === 0}
            <div class="p-6 text-center text-sm text-surface-500">{legacyUsers.length ? t('applications.noLegacySearchResults') : t('common.noData')}</div>
          {:else}
            <div class="overflow-x-auto">
              <table class="table min-w-full">
                <thead>
                  <tr>
                    <th>{t('applications.username')}</th>
                    <th>{t('applications.legacyIdentifier')}</th>
                    <th>{t('applications.userId')}</th>
                    <th class="w-20 !text-center">{t('applications.status')}</th>
                    <th>{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {#each filteredLegacyUsers as item}
                    <tr>
                      <td class="font-medium">{item.username}</td>
                      <td class="max-w-48 truncate text-xs text-surface-500">{item.legacy_user_identifier || '-'}</td>
                      <td class="max-w-48 truncate text-xs text-surface-500">{item.user_id}</td>
                      <td class="w-20 !text-center">
                        <span class="mx-auto flex size-5 items-center justify-center">
                          <span class={`size-2 rounded-full ${item.is_active ? 'bg-success-500' : 'bg-error-500'}`} aria-hidden="true"></span>
                          <span class="sr-only">{item.is_active ? t('applications.status.active') : t('applications.status.disabled')}</span>
                        </span>
                      </td>
                      <td>
                        <div class="relative flex items-center gap-1">
                          <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => void openLegacyDetail(item)} aria-label={t('applications.legacyDetails')} title={t('applications.legacyDetails')}>
                            <FileText class="size-4" aria-hidden="true" />
                          </button>
                          <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => openLegacyEdit(item)} aria-label={t('users.edit')} title={t('users.edit')}>
                            <Edit3 class="size-4" aria-hidden="true" />
                          </button>
                          <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => void toggleLegacyActive(item, !item.is_active)} aria-label={item.is_active ? t('applications.legacyDisable') : t('applications.legacyEnable')} title={item.is_active ? t('applications.legacyDisable') : t('applications.legacyEnable')}>
                            <Power class="size-4" aria-hidden="true" />
                          </button>
                          <button class="btn btn-xs preset-outlined-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => (pendingDeleteKey = `legacy:${item.username}`)} aria-label={t('common.delete')} title={t('common.delete')}>
                            <Trash2 class="size-4" aria-hidden="true" />
                          </button>
                          {#if pendingDeleteKey === `legacy:${item.username}`}
                            <div class="absolute right-full top-1/2 z-10 mr-1 flex -translate-y-1/2 items-center gap-1 rounded-container border border-surface-200-800 bg-surface-50-950 p-1 shadow-lg">
                              <button class="btn btn-xs preset-filled-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => void removeLegacy(item)} aria-label={t('common.confirmDelete')} title={t('common.confirmDelete')}>
                                <Check class="size-4" aria-hidden="true" />
                              </button>
                              <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => (pendingDeleteKey = '')} aria-label={t('common.cancel')} title={t('common.cancel')}>
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
        {/if}
      {:else}
        <div class="p-6 text-center text-sm text-surface-500">{t('applications.selectApp')}</div>
      {/if}
    </section>
  </div>

  {#if appModalOpen}
    <div class="fixed inset-0 z-40 bg-surface-950/55 backdrop-blur-sm" aria-hidden="true" on:click={closeAppModal}></div>
    <div class="fixed inset-y-0 right-0 z-50 flex w-full justify-end" role="dialog" aria-modal="true" aria-labelledby="application-dialog-title" tabindex="-1">
      <form
        class="flex h-full w-full max-w-xl flex-col border-l border-surface-200-800 bg-surface-50-950 text-surface-950-50 shadow-2xl"
        on:submit|preventDefault={saveApplication}
      >
        <header class="flex items-center justify-between gap-3 border-b border-surface-200-800 px-5 py-4">
          <div>
            <h2 id="application-dialog-title" class="text-lg font-semibold">{appEditingId ? t('applications.editTitle') : t('applications.createTitle')}</h2>
            <p class="mt-1 text-sm text-surface-500">{appEditingId ? appType : t('applications.type')}</p>
          </div>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeAppModal} aria-label={t('common.close')}>{t('common.close')}</button>
        </header>

        <div class="flex-1 space-y-4 overflow-y-auto px-5 py-5">
          <label class="block">
            <span class="text-sm text-surface-500">{t('applications.name')}</span>
            <input class="input w-full bg-surface-50-950" type="text" bind:value={appName} required />
          </label>

          <label class="block">
            <span class="text-sm text-surface-500">{t('applications.type')}</span>
            <select class="input w-full bg-surface-50-950" bind:value={appType} disabled={appEditingId !== ''}>
              {#each applicationTypes as type}
                <option value={type}>{applicationTypeLabel(type)}</option>
              {/each}
            </select>
          </label>

          {#if appEditingId}
            <label class="block">
              <span class="text-sm text-surface-500">{t('applications.status')}</span>
              <select class="input w-full bg-surface-50-950" bind:value={appStatus}>
                {#each applicationStatuses as status}
                  <option value={status}>{applicationStatusLabel(status)}</option>
                {/each}
              </select>
            </label>
          {/if}
        </div>

        <footer class="flex justify-end gap-2 border-t border-surface-200-800 bg-surface-100-900 px-5 py-4">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeAppModal}>{t('common.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={appSaving || appName.trim() === ''}>
            {appSaving ? t('common.loading') : t('common.save')}
          </button>
        </footer>
      </form>
    </div>
  {/if}

  {#if selectedApp && oidcModalOpen}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="oidc-client-dialog-title">
      <form class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-xl w-full overflow-y-auto p-4 space-y-3" on:submit|preventDefault={saveOIDC}>
        <h2 id="oidc-client-dialog-title" class="font-semibold">{oidcClientEditing ? t('common.update') : t('applications.oidcManagement')}</h2>

        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.clientId')}</span>
          <input class="input w-full" type="text" bind:value={oidcFormClientId} required />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.redirectUris')}</span>
          <textarea class="input w-full min-h-24" bind:value={oidcRedirectUris} required></textarea>
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.scopes')}</span>
          <textarea class="input w-full min-h-20" bind:value={oidcScopes}></textarea>
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.grantTypes')}</span>
          <textarea class="input w-full min-h-20" bind:value={oidcGrantTypes}></textarea>
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.responseTypes')}</span>
          <textarea class="input w-full min-h-20" bind:value={oidcResponseTypes}></textarea>
        </label>
        <label class="flex items-center gap-2">
          <input type="checkbox" bind:checked={oidcPkce} />
          <span class="text-sm">{t('applications.pkce')}</span>
        </label>

        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeOIDC}>{t('common.cancel')}</button>
          <button
            class="btn preset-filled-primary-500"
            type="submit"
            disabled={oidcSaving || oidcFormClientId.trim() === '' || oidcRedirectUris.trim() === ''}
          >
            {oidcSaving ? t('common.loading') : t('common.save')}
          </button>
        </div>
      </form>
    </div>
  {/if}

  {#if oidcDetailOpen && oidcDetail}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="oidc-detail-dialog-title">
      <div class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-3xl w-full overflow-y-auto p-4 space-y-4">
        <div class="flex items-center justify-between gap-3">
          <h2 id="oidc-detail-dialog-title" class="font-semibold">{t('applications.oidcDetails')}</h2>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeOIDCDetail}>{t('common.close')}</button>
        </div>

        {#if oidcDetailLoading}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else}
          <dl class="grid gap-3 text-sm md:grid-cols-2">
            <div>
              <dt class="text-surface-500">{t('applications.clientId')}</dt>
              <dd class="break-all font-medium">{oidcDetail.client_id}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.status')}</dt>
              <dd class="font-medium">{oidcDetail.status ? applicationStatusLabel(oidcDetail.status) : '-'}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.pkce')}</dt>
              <dd class="font-medium">{oidcDetail.pkce_required ? t('common.yes') : t('common.no')}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.scopes')}</dt>
              <dd class="break-words font-medium">{oidcDetail.allowed_scopes?.join(', ') || '-'}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.grantTypes')}</dt>
              <dd class="break-words font-medium">{oidcDetail.grant_types?.join(', ') || '-'}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.responseTypes')}</dt>
              <dd class="break-words font-medium">{oidcDetail.response_types?.join(', ') || '-'}</dd>
            </div>
            <div class="md:col-span-2">
              <dt class="text-surface-500">{t('applications.redirectUris')}</dt>
              <dd class="break-words font-medium">{oidcDetail.redirect_uris?.join(', ') || '-'}</dd>
            </div>
          </dl>
        {/if}
      </div>
    </div>
  {/if}

  {#if legacyDetailOpen && legacyDetail}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="legacy-detail-dialog-title">
      <div class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-3xl w-full overflow-y-auto p-4 space-y-4">
        <div class="flex items-center justify-between gap-3">
          <h2 id="legacy-detail-dialog-title" class="font-semibold">{t('applications.legacyDetails')}</h2>
          <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeLegacyDetail}>{t('common.close')}</button>
        </div>

        {#if legacyDetailLoading}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else}
          <dl class="grid gap-3 text-sm md:grid-cols-2">
            <div>
              <dt class="text-surface-500">{t('applications.username')}</dt>
              <dd class="break-all font-medium">{legacyDetail.username}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.status')}</dt>
              <dd class="font-medium">{legacyDetail.is_active ? t('applications.status.active') : t('applications.status.disabled')}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.userId')}</dt>
              <dd class="break-all font-medium">{legacyDetail.user_id}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.legacyIdentifier')}</dt>
              <dd class="break-all font-medium">{legacyDetail.legacy_user_identifier || '-'}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.authScheme')}</dt>
              <dd class="font-medium">{legacyDetail.auth_scheme || '-'}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.createdAt')}</dt>
              <dd class="font-medium">{legacyDetail.created_at ? new Date(legacyDetail.created_at).toLocaleString() : '-'}</dd>
            </div>
            <div>
              <dt class="text-surface-500">{t('applications.updatedAt')}</dt>
              <dd class="font-medium">{legacyDetail.updated_at ? new Date(legacyDetail.updated_at).toLocaleString() : '-'}</dd>
            </div>
          </dl>
        {/if}
      </div>
    </div>
  {/if}

  {#if selectedApp && legacyModalOpen}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="legacy-user-dialog-title">
      <form class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-xl w-full overflow-y-auto p-4 space-y-3" on:submit|preventDefault={saveLegacy}>
        <h2 id="legacy-user-dialog-title" class="font-semibold">{legacyEditing ? t('applications.legacyManagement') : t('applications.createLegacyUser')}</h2>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.username')}</span>
          <input class="input w-full" type="text" bind:value={legacyUsername} disabled={legacyEditing !== null} required />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.userId')}</span>
          <input class="input w-full" type="text" bind:value={legacyUserId} required />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.legacyIdentifier')}</span>
          <input class="input w-full" type="text" bind:value={legacyLegacyId} />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.legacyPassword')}</span>
          <input class="input w-full" type="password" bind:value={legacyPassword} autocomplete="new-password" required={!legacyEditing} />
        </label>
        <label class="flex items-center gap-2">
          <input type="checkbox" bind:checked={legacyActive} />
          <span class="text-sm">{t('applications.legacyEnable')}</span>
        </label>

        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeLegacy}>{t('common.cancel')}</button>
          <button
            class="btn preset-filled-primary-500"
            type="submit"
            disabled={legacySaving || legacyUsername.trim() === '' || legacyUserId.trim() === '' || (!legacyEditing && legacyPassword.trim() === '')}
          >
            {legacySaving ? t('common.loading') : t('common.save')}
          </button>
        </div>
      </form>
    </div>
  {/if}
</section>
