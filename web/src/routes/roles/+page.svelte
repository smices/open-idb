<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type Role, type Permission, type ResourceScope } from '$lib/api';

  let roles: Role[] = [];
  let permissions: Permission[] = [];
  let resourceScopes: ResourceScope[] = [];
  let roleSearch = '';
  let permissionSearch = '';

  let roleLoading = false;
  let permLoading = false;

  let roleOpen = false;
  let permOpen = false;
  let editingRole: Role | null = null;
  let editingPermission: Permission | null = null;

  let roleName = '';
  let roleCode = '';
  let roleDescription = '';
  let permissionCode = '';
  let permissionName = '';
  let permissionType = 'api';

  let selectedRole: Role | null = null;
  let rolePerms: string[] = [];
  let selectedScopeId = '';
  let selectedScopeEffect: 'allow' | 'deny' = 'allow';
  let rolePermLoading = false;
  let savingRole = false;
  let savingPermission = false;
  let checkingPermission = false;
  let checkUserId = '';
  let checkPermissionCode = '';
  let checkResult: boolean | null = null;
  let pendingDeleteKey = '';
  let error = '';
  let message = '';

  const matchesRoleSearch = (role: Role, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [role.name, role.code, role.description, role.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const matchesPermissionSearch = (permission: Permission, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [permission.name, permission.code, permission.type, permission.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const fetchAll = async () => {
    roleLoading = true;
    permLoading = true;
    error = '';

    try {
      const [roleData, permData, scopeData] = await Promise.all([
        api.listRoles({ limit: 200 }),
        api.listPermissions({ limit: 200 }),
        api.listResourceScopes({ limit: 200 }),
      ]);
      roles = roleData.items || [];
      permissions = permData.items || [];
      resourceScopes = scopeData.items || [];
    } catch {
      error = t('roles.fetchFailed');
    } finally {
      roleLoading = false;
      permLoading = false;
    }
  };

  const openRoleForm = () => {
    pendingDeleteKey = '';
    editingRole = null;
    roleName = '';
    roleCode = '';
    roleDescription = '';
    roleOpen = true;
  };

  const openRoleEdit = (role: Role) => {
    pendingDeleteKey = '';
    editingRole = role;
    roleName = role.name;
    roleCode = role.code;
    roleDescription = role.description || '';
    roleOpen = true;
  };

  const saveRole = async () => {
    savingRole = true;
    error = '';
    message = '';
    try {
      if (editingRole) {
        await api.updateRole(editingRole.id, {
          name: roleName,
          description: roleDescription,
        });
        message = t('roles.updateSuccess');
      } else {
        await api.createRole({
          name: roleName,
          code: roleCode,
          description: roleDescription,
        });
        message = t('roles.createSuccess');
      }
      editingRole = null;
      roleOpen = false;
      await fetchAll();
    } catch {
      error = t(editingRole ? 'roles.updateFailed' : 'roles.createFailed');
    } finally {
      savingRole = false;
    }
  };

  const deleteRole = async (role: Role) => {
    error = '';
    message = '';
    try {
      await api.deleteRole(role.id);
      pendingDeleteKey = '';
      message = t('roles.deleteSuccess');
      await fetchAll();
    } catch {
      error = t('roles.deleteFailed');
    }
  };

  const openPermForm = () => {
    pendingDeleteKey = '';
    editingPermission = null;
    permissionCode = '';
    permissionName = '';
    permissionType = 'ui';
    permOpen = true;
  };

  const openPermissionEdit = (permission: Permission) => {
    pendingDeleteKey = '';
    editingPermission = permission;
    permissionCode = permission.code;
    permissionName = permission.name;
    permissionType = permission.type;
    permOpen = true;
  };

  const savePermission = async () => {
    savingPermission = true;
    error = '';
    message = '';
    try {
      if (editingPermission) {
        await api.updatePermission(editingPermission.id, { name: permissionName });
        message = t('roles.updateSuccess');
      } else {
        await api.createPermission({
          code: permissionCode,
          name: permissionName,
          type: permissionType,
        });
        message = t('roles.createSuccess');
      }
      editingPermission = null;
      permOpen = false;
      await fetchAll();
    } catch {
      error = t(editingPermission ? 'roles.updateFailed' : 'roles.createFailed');
    } finally {
      savingPermission = false;
    }
  };

  const deletePermission = async (permissionId: string) => {
    error = '';
    message = '';
    try {
      await api.deletePermission(permissionId);
      pendingDeleteKey = '';
      message = t('roles.deleteSuccess');
      await fetchAll();
    } catch {
      error = t('roles.deleteFailed');
    }
  };

  const openRolePermission = async (role: Role) => {
    selectedRole = role;
    selectedScopeId = '';
    selectedScopeEffect = 'allow';
    rolePermLoading = true;

    try {
      const perms = await api.listRolePermissions(role.id);
      rolePerms = (perms || []).map((item) => item.id);
    } catch {
      error = t('roles.permissionFetchFailed');
    } finally {
      rolePermLoading = false;
    }
  };

  const closeRolePermission = () => {
    selectedRole = null;
    rolePerms = [];
    selectedScopeId = '';
    selectedScopeEffect = 'allow';
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key !== 'Escape') return;
    if (roleOpen) {
      roleOpen = false;
      editingRole = null;
    } else if (permOpen) {
      permOpen = false;
      editingPermission = null;
    } else if (selectedRole) {
      closeRolePermission();
    }
  };

  const onRolePermissionToggle = async (permId: string, checked: boolean) => {
    if (!selectedRole) return;
    error = '';
    message = '';
    try {
      if (checked) {
        await api.assignPermissionToRole(selectedRole.id, permId);
        rolePerms = [...rolePerms, permId];
      } else {
        await api.removePermissionFromRole(selectedRole.id, permId);
        rolePerms = rolePerms.filter((id) => id !== permId);
      }
      message = t(checked ? 'roles.assignSuccess' : 'roles.removeSuccess');
    } catch {
      error = t('roles.assignFailed');
    }
  };

  const assignScopeToRole = async () => {
    if (!selectedRole || !selectedScopeId) return;
    error = '';
    message = '';
    try {
      await api.assignResourceScopeToRole(selectedRole.id, selectedScopeId, selectedScopeEffect);
      message = t('roles.scopeAssignSuccess');
    } catch {
      error = t('roles.assignFailed');
    }
  };

  const removeScopeFromRole = async () => {
    if (!selectedRole || !selectedScopeId) return;
    error = '';
    message = '';
    try {
      await api.removeResourceScopeFromRole(selectedRole.id, selectedScopeId);
      message = t('roles.scopeRemoveSuccess');
    } catch {
      error = t('roles.assignFailed');
    }
  };

  const runPermissionCheck = async () => {
    checkingPermission = true;
    checkResult = null;
    error = '';
    message = '';
    try {
      const result = await api.checkPermission({
        user_id: checkUserId,
        permission: checkPermissionCode,
      });
      checkResult = result.allowed;
    } catch {
      error = t('roles.checkFailed');
    } finally {
      checkingPermission = false;
    }
  };

  onMount(fetchAll);

  $: filteredRoles = roles.filter((role) => matchesRoleSearch(role, roleSearch));
  $: filteredPermissions = permissions.filter((permission) => matchesPermissionSearch(permission, permissionSearch));
  $: permissionTypeCount = new Set(permissions.map((permission) => permission.type).filter(Boolean)).size;
  $: selectedRolePermissionCount = rolePerms.length;
</script>

<svelte:head>
  <title>{t('roles.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <header class="flex items-center justify-end">
    <div class="flex gap-2">
      <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openPermForm}>{t('roles.addPermission')}</button>
      <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={openRoleForm}>{t('roles.addRole')}</button>
    </div>
  </header>

  {#if message}
    <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
  {/if}
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 p-4">
    <form class="grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto_auto]" on:submit|preventDefault={runPermissionCheck}>
      <label class="block">
        <span class="text-sm text-surface-500">{t('roles.userId')}</span>
        <input class="input w-full" type="text" bind:value={checkUserId} required />
      </label>
      <label class="block">
        <span class="text-sm text-surface-500">{t('roles.permission')}</span>
        <input class="input w-full" type="text" bind:value={checkPermissionCode} required />
      </label>
      <div class="flex items-end">
        <button class="btn preset-filled-primary-500" type="submit" disabled={checkingPermission || checkUserId.trim() === '' || checkPermissionCode.trim() === ''}>
          {checkingPermission ? t('common.loading') : t('roles.check')}
        </button>
      </div>
      <div class="flex items-end">
        {#if checkResult !== null}
          <span class={`badge ${checkResult ? 'preset-tonal-success' : 'preset-tonal-error'}`}>{checkResult ? t('roles.allowed') : t('roles.denied')}</span>
        {/if}
      </div>
    </form>
  </section>

  <div class="grid gap-4 xl:grid-cols-2">
    <section class="card bg-surface-50-950 border border-surface-200-800 p-4">
      <div class="mb-3 space-y-3">
        <h2 class="font-semibold">{t('roles.title')}</h2>
        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.searchRoles')}</span>
          <input class="input w-full" type="search" bind:value={roleSearch} placeholder={t('roles.searchRoles')} />
        </label>
        <div class="grid gap-3 text-sm sm:grid-cols-3">
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('roles.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{filteredRoles.length}</p></article>
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('dashboard.total')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{roles.length}</p></article>
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('roles.scopes')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{resourceScopes.length}</p></article>
        </div>
      </div>
      {#if roleLoading}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {:else}
        {#if roles.length === 0}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
        {:else if filteredRoles.length === 0}
          <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('roles.noRoleSearchResults')}</div>
        {:else}
          <div class="divide-y divide-surface-200-800">
            {#each filteredRoles as role (role.id)}
              <article class="py-3">
                <header class="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <h3 class="font-medium">{role.name}</h3>
                    <p class="text-xs text-surface-500">{role.code} · {role.description || '-'}</p>
                  </div>
                  <div class="flex flex-wrap gap-2">
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openRoleEdit(role)}>{t('common.edit')}</button>
                    <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => void openRolePermission(role)}>{t('roles.managePermissions')}</button>
                    <button
                      class="btn preset-tonal-error btn-xs"
                      type="button"
                      on:click={() => (pendingDeleteKey === `role:${role.id}` ? void deleteRole(role) : (pendingDeleteKey = `role:${role.id}`))}
                    >
                      {pendingDeleteKey === `role:${role.id}` ? t('common.confirmDelete') : t('common.delete')}
                    </button>
                  </div>
                </header>
              </article>
            {/each}
          </div>
        {/if}
      {/if}
    </section>

    <section class="card bg-surface-50-950 border border-surface-200-800 p-4">
      <div class="mb-3 space-y-3">
        <h2 class="font-semibold">{t('roles.permissions')}</h2>
        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.searchPermissions')}</span>
          <input class="input w-full" type="search" bind:value={permissionSearch} placeholder={t('roles.searchPermissions')} />
        </label>
        <div class="grid gap-3 text-sm sm:grid-cols-3">
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('roles.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{filteredPermissions.length}</p></article>
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('dashboard.total')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{permissions.length}</p></article>
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('roles.permissionTypes')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{permissionTypeCount}</p></article>
        </div>
      </div>
      {#if permLoading}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {:else if permissions.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.noData')}</div>
      {:else if filteredPermissions.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('roles.noPermissionSearchResults')}</div>
      {:else}
        <div class="divide-y divide-surface-200-800">
          {#each filteredPermissions as permission (permission.id)}
            <article class="py-3">
              <header class="flex flex-wrap items-start justify-between gap-2">
                <div>
                  <h3 class="font-medium">{permission.name}</h3>
                  <p class="text-xs text-surface-500">{permission.code}</p>
                </div>
                <div class="flex flex-wrap gap-2">
                  <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openPermissionEdit(permission)}>{t('common.edit')}</button>
                  <button
                    class="btn preset-tonal-error btn-xs"
                    type="button"
                    on:click={() => (pendingDeleteKey === `permission:${permission.id}` ? void deletePermission(permission.id) : (pendingDeleteKey = `permission:${permission.id}`))}
                  >
                    {pendingDeleteKey === `permission:${permission.id}` ? t('common.confirmDelete') : t('common.delete')}
                  </button>
                </div>
              </header>
            </article>
          {/each}
        </div>
      {/if}
    </section>
  </div>

  {#if selectedRole}
    <section class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="font-semibold">{t('roles.managePermissions')}: {selectedRole.name}</h2>
        <span class="badge preset-outlined-surface-500">
          {t('roles.assignedPermissions')}: {selectedRolePermissionCount}
        </span>
      </div>

      {#if rolePermLoading}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {:else}
        <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-2 p-3 md:grid-cols-[minmax(0,1fr)_auto_auto_auto]" on:submit|preventDefault={assignScopeToRole}>
          <label class="block">
            <span class="text-sm text-surface-500">{t('roles.scope')}</span>
            <select class="input w-full" bind:value={selectedScopeId} required>
              <option value="">{t('roles.noScopes')}</option>
              {#each resourceScopes as scope (scope.id)}
                <option value={scope.id}>{scope.name} · {scope.type}:{scope.key}</option>
              {/each}
            </select>
          </label>
          <label class="block">
            <span class="text-sm text-surface-500">{t('roles.effect')}</span>
            <select class="input w-full" bind:value={selectedScopeEffect}>
              <option value="allow">{t('assignments.allow')}</option>
              <option value="deny">{t('assignments.deny')}</option>
            </select>
          </label>
          <div class="flex items-end">
            <button class="btn preset-filled-primary-500" type="submit" disabled={!selectedScopeId}>{t('roles.assignScope')}</button>
          </div>
          <div class="flex items-end">
            <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void removeScopeFromRole()} disabled={!selectedScopeId}>{t('roles.removeScope')}</button>
          </div>
        </form>

        <div class="grid gap-2 md:grid-cols-2">
          {#each permissions as permission (permission.id)}
            <article class="card bg-surface-50-950 border border-surface-200-800 px-3 py-2">
              <label class="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={rolePerms.includes(permission.id)}
                  on:change={(event) => void onRolePermissionToggle(permission.id, (event.currentTarget as HTMLInputElement).checked)}
                />
                <span class="text-sm">{permission.name}</span>
              </label>
            </article>
          {/each}
          {#if permissions.length === 0}
            <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('roles.fetchPermissionFailed')}</div>
          {/if}
        </div>
      {/if}

      <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeRolePermission}>{t('common.close')}</button>
    </section>
  {/if}

  {#if roleOpen}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="role-dialog-title" tabindex="-1">
      <form class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] w-full max-w-md overflow-y-auto p-4 space-y-3" on:submit|preventDefault={saveRole}>
        <h2 id="role-dialog-title" class="font-semibold">{editingRole ? t('roles.editRole') : t('roles.createRole')}</h2>
        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.name')}</span>
          <input class="input w-full" type="text" bind:value={roleName} required />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.code')}</span>
          <input class="input w-full" type="text" bind:value={roleCode} required disabled={!!editingRole} />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.description')}</span>
          <input class="input w-full" type="text" bind:value={roleDescription} />
        </label>
        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={() => { roleOpen = false; editingRole = null; }}>{t('common.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={savingRole || roleName.trim() === '' || roleCode.trim() === ''}>
            {savingRole ? t('common.loading') : t('common.save')}
          </button>
        </div>
      </form>
    </div>
  {/if}

  {#if permOpen}
    <div class="fixed inset-0 z-20 flex items-start justify-center overflow-y-auto bg-surface-900/70 p-4 py-6 sm:items-center" role="dialog" aria-modal="true" aria-labelledby="permission-dialog-title" tabindex="-1">
      <form class="card bg-surface-50-950 border border-surface-200-800 max-h-[calc(100vh-3rem)] w-full max-w-md overflow-y-auto p-4 space-y-3" on:submit|preventDefault={savePermission}>
        <h2 id="permission-dialog-title" class="font-semibold">{editingPermission ? t('roles.editPermission') : t('roles.createPermission')}</h2>
        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.permCode')}</span>
          <input class="input w-full" type="text" bind:value={permissionCode} required disabled={!!editingPermission} />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.permName')}</span>
          <input class="input w-full" type="text" bind:value={permissionName} required />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.permType')}</span>
          <input class="input w-full" type="text" bind:value={permissionType} disabled={!!editingPermission} />
        </label>
        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={() => { permOpen = false; editingPermission = null; }}>{t('common.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={savingPermission || permissionCode.trim() === '' || permissionName.trim() === ''}>
            {savingPermission ? t('common.loading') : t('common.save')}
          </button>
        </div>
      </form>
    </div>
  {/if}
</section>
