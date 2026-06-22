<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t, tf } from '$lib/i18n';
  import { api, type Application, type OIDCClient, type LegacyAppUser } from '$lib/api';

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
      if (appEditingId) {
        await api.updateApplication(appEditingId, { name: appName, status: appStatus });
      } else {
        await api.createApplication({ name: appName, type: appType });
      }
      message = t('applications.saveSuccess');
      appModalOpen = false;
      await loadApplications();
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
  $: activeAppCount = apps.filter((app) => app.status === 'active').length;
  $: oidcAppCount = apps.filter((app) => app.type === 'oidc_client').length;
  $: pkceClientCount = oidcClients.filter((client) => client.pkce_required).length;
  $: activeLegacyUserCount = legacyUsers.filter((record) => record.is_active).length;
</script>

<svelte:head>
  <title>{t('applications.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <header class="flex flex-wrap items-end justify-between gap-3">
    <span aria-hidden="true"></span>
    <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openCreateApp}>{t('applications.create')}</button>
  </header>

  <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_auto]" on:submit|preventDefault>
    <label class="block">
      <span class="text-sm text-surface-500">{t('applications.search')}</span>
      <input class="input w-full" type="search" bind:value={appSearch} placeholder={t('applications.searchPlaceholder')} />
    </label>
    <div class="flex flex-wrap items-end gap-2">
      <button class="btn preset-filled-primary-500" type="submit">{t('common.filter')}</button>
      <button class="btn preset-outlined-surface-500" type="button" on:click={resetFilters}>{t('common.reset')}</button>
    </div>
  </form>

  {#if message}
    <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
  {/if}
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <div class="card bg-surface-50-950 border border-surface-200-800 p-4">
    <div class="mb-4 grid gap-3 text-sm sm:grid-cols-4">
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{filteredApps.length}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('dashboard.total')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{apps.length}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.activeApps')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeAppCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.oidcApps')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{oidcAppCount}</p></article>
    </div>

    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if !apps.length}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
    {:else if !filteredApps.length}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('applications.noSearchResults')}</div>
    {:else}
      <div class="divide-y divide-surface-200-800">
        {#each filteredApps as app}
          <article class="py-3">
            <div class="flex flex-wrap justify-between gap-2">
              <div>
                <div class="font-semibold">{app.name}</div>
                <div class="text-xs text-surface-500">{applicationTypeLabel(app.type)}</div>
              </div>
              <span class={`badge ${app.status === 'active' ? 'preset-tonal-success' : 'preset-outlined-surface-500'}`}>{applicationStatusLabel(app.status)}</span>
            </div>
            <div class="mt-3 flex flex-wrap gap-2">
              <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void selectApplication(app)}>{t('applications.manage')}</button>
              <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => openEditApp(app)}>{t('users.edit')}</button>
              <button
                class="btn preset-tonal-error btn-xs"
                type="button"
                on:click={() => (pendingDeleteKey === `application:${app.id}` ? void deleteApplication(app.id) : (pendingDeleteKey = `application:${app.id}`))}
              >
                {pendingDeleteKey === `application:${app.id}` ? t('common.confirmDelete') : t('common.delete')}
              </button>
            </div>
          </article>
        {/each}
      </div>
    {/if}
  </div>

  <div class="grid gap-4 xl:grid-cols-2">
    <section class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3">
      <h2 class="font-semibold">{t('applications.oidcManagement')}</h2>
      {#if selectedApp}
        <div class="flex flex-wrap justify-between items-center gap-2">
          <span>{selectedApp.name}</span>
          <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openOIDCModal}>{t('applications.oidc')}</button>
        </div>

        {#if oidcLoading}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else}
          <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-center">
            <label class="block">
              <span class="sr-only">{t('applications.searchOidc')}</span>
              <input class="input w-full" type="search" bind:value={oidcSearch} placeholder={t('applications.searchOidc')} />
            </label>
            <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredOIDCClients.length} / ${oidcClients.length}`}</p></article>
            <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.pkceClients')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{pkceClientCount}</p></article>
          </div>
          <div class="divide-y divide-surface-200-800 max-h-80 overflow-auto">
            {#each filteredOIDCClients as item}
              <article class="py-3">
                <div class="flex justify-between items-start">
                  <div>
                    <div class="font-medium">{item.client_id}</div>
                    <div class="text-xs text-surface-500">{item.redirect_uris?.join(', ')}</div>
                    <div class="mt-1 text-xs text-surface-500">{item.allowed_scopes?.join(', ') || '-'}</div>
                  </div>
                  <div class="flex gap-2">
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => void openOIDCDetail(item)}>{t('applications.oidcDetails')}</button>
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => selectOIDC(item)}>{t('common.update')}</button>
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => void rotateSecret(item)}>{t('applications.rotateSecret')}</button>
                    <button
                      class="btn preset-tonal-error btn-xs"
                      type="button"
                      on:click={() => (pendingDeleteKey === `oidc:${item.id}` ? void removeOIDC(item) : (pendingDeleteKey = `oidc:${item.id}`))}
                    >
                      {pendingDeleteKey === `oidc:${item.id}` ? t('common.confirmDelete') : t('common.delete')}
                    </button>
                  </div>
                </div>
              </article>
            {:else}
              <div class="py-3 text-sm text-surface-500">{oidcClients.length ? t('applications.noOidcSearchResults') : t('common.noData')}</div>
            {/each}
          </div>
        {/if}
      {:else}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('assignments.selectApp')}</div>
      {/if}
    </section>

    <section class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3">
      <h2 class="font-semibold">{t('applications.legacyManagement')}</h2>
      {#if selectedApp}
        <div class="flex flex-wrap justify-between items-center gap-2">
          <span>{selectedApp.name}</span>
          <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openLegacyCreate}>{t('applications.createLegacyUser')}</button>
        </div>

        {#if legacyLoading}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
        {:else}
          <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-center">
            <label class="block">
              <span class="sr-only">{t('applications.searchLegacy')}</span>
              <input class="input w-full" type="search" bind:value={legacySearch} placeholder={t('applications.searchLegacy')} />
            </label>
            <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredLegacyUsers.length} / ${legacyUsers.length}`}</p></article>
            <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.activeLegacyUsers')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeLegacyUserCount}</p></article>
          </div>
          <div class="divide-y divide-surface-200-800 max-h-80 overflow-auto">
            {#each filteredLegacyUsers as item}
              <article class="py-3">
                <div class="flex justify-between items-start">
                  <div>
                    <div class="font-medium">{item.username}</div>
                    <div class="text-xs text-surface-500">{item.legacy_user_identifier}</div>
                    <div class="mt-1 text-xs text-surface-500">{item.user_id}</div>
                  </div>
                  <div class="flex gap-2">
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => void openLegacyDetail(item)}>{t('applications.legacyDetails')}</button>
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openLegacyEdit(item)}>{t('users.edit')}</button>
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => void toggleLegacyActive(item, !item.is_active)}>{
                      item.is_active ? t('applications.legacyDisable') : t('applications.legacyEnable')
                    }</button>
                    <button
                      class="btn preset-tonal-error btn-xs"
                      type="button"
                      on:click={() => (pendingDeleteKey === `legacy:${item.username}` ? void removeLegacy(item) : (pendingDeleteKey = `legacy:${item.username}`))}
                    >
                      {pendingDeleteKey === `legacy:${item.username}` ? t('common.confirmDelete') : t('common.delete')}
                    </button>
                  </div>
                </div>
              </article>
            {:else}
              <div class="py-3 text-sm text-surface-500">{legacyUsers.length ? t('applications.noLegacySearchResults') : t('common.noData')}</div>
            {/each}
          </div>
        {/if}
      {:else}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('assignments.selectApp')}</div>
      {/if}
    </section>
  </div>

  {#if appModalOpen}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="application-dialog-title">
      <form class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] max-w-lg w-full overflow-y-auto p-4 space-y-3" on:submit|preventDefault={saveApplication}>
        <h2 id="application-dialog-title" class="font-semibold">{appEditingId ? t('applications.editTitle') : t('applications.createTitle')}</h2>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.name')}</span>
          <input class="input w-full" type="text" bind:value={appName} required />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.type')}</span>
          <select class="input w-full" bind:value={appType} disabled={appEditingId !== ''}>
            {#each applicationTypes as type}
              <option value={type}>{applicationTypeLabel(type)}</option>
            {/each}
          </select>
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('applications.status')}</span>
          <select class="input w-full" bind:value={appStatus}>
            {#each applicationStatuses as status}
              <option value={status}>{applicationStatusLabel(status)}</option>
            {/each}
          </select>
        </label>

        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeAppModal}>{t('common.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={appSaving || appName.trim() === ''}>
            {appSaving ? t('common.loading') : t('common.save')}
          </button>
        </div>
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
          <span class="text-sm text-surface-500">{t('identitySources.name')}</span>
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
