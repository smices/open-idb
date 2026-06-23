<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { Check, KeyRound, Plus, Settings, Trash2, X } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { api, type Permission, type Role } from '$lib/api';
  import Toast from '$lib/components/ui/Toast.svelte';

  let roles: Role[] = [];
  let permissions: Permission[] = [];
  let rolePermissions: string[] = [];
  let loading = false;
  let permissionLoading = false;
  let drawerOpen = false;
  let editingRole: Role | null = null;
  let roleName = '';
  let roleCode = '';
  let roleDescription = '';
  let saving = false;
  let pendingDeleteKey = '';
  let error = '';
  let message = '';

  const loadRoles = async () => {
    loading = true;
    error = '';
    try {
      const data = await api.listRoles({ limit: 200 });
      roles = data.items || [];
    } catch {
      error = t('roles.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const loadPermissions = async () => {
    try {
      const data = await api.listPermissions({ limit: 500 });
      permissions = data.items || [];
    } catch {
      error = t('roles.fetchPermissionFailed');
    }
  };

  const openCreateRole = () => {
    pendingDeleteKey = '';
    editingRole = null;
    roleName = '';
    roleCode = '';
    roleDescription = '';
    rolePermissions = [];
    error = '';
    message = '';
    drawerOpen = true;
  };

  const openRoleDrawer = async (role: Role) => {
    pendingDeleteKey = '';
    editingRole = role;
    roleName = role.name;
    roleCode = role.code;
    roleDescription = role.description || '';
    rolePermissions = [];
    error = '';
    message = '';
    drawerOpen = true;
    permissionLoading = true;
    try {
      const assigned = await api.listRolePermissions(role.id);
      rolePermissions = (assigned || []).map((permission) => permission.id);
    } catch {
      error = t('roles.permissionFetchFailed');
    } finally {
      permissionLoading = false;
    }
  };

  const closeDrawer = () => {
    drawerOpen = false;
    editingRole = null;
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && drawerOpen) {
      closeDrawer();
    }
  };

  const saveRole = async () => {
    const name = roleName.trim();
    const code = roleCode.trim();
    if (!name || !code) return;

    saving = true;
    error = '';
    message = '';
    try {
      if (editingRole) {
        await api.updateRole(editingRole.id, {
          name,
          description: roleDescription.trim(),
        });
      } else {
        await api.createRole({
          name,
          code,
          description: roleDescription.trim(),
        });
      }
      message = t('roles.saveSuccess');
      drawerOpen = false;
      editingRole = null;
      await loadRoles();
    } catch {
      error = t(editingRole ? 'roles.updateFailed' : 'roles.createFailed');
    } finally {
      saving = false;
    }
  };

  const deleteRole = async (role: Role) => {
    error = '';
    message = '';
    try {
      await api.deleteRole(role.id);
      pendingDeleteKey = '';
      message = t('roles.deleteSuccess');
      await loadRoles();
    } catch {
      error = t('roles.deleteFailed');
    }
  };

  const toggleRolePermission = async (permission: Permission, checked: boolean) => {
    if (!editingRole) return;
    error = '';
    message = '';
    try {
      if (checked) {
        await api.assignPermissionToRole(editingRole.id, permission.id);
        rolePermissions = [...rolePermissions, permission.id];
      } else {
        await api.removePermissionFromRole(editingRole.id, permission.id);
        rolePermissions = rolePermissions.filter((id) => id !== permission.id);
      }
      message = t('roles.permissionSaveSuccess');
    } catch {
      error = t('roles.assignFailed');
    }
  };

  onMount(() => {
    void loadRoles();
    void loadPermissions();
  });
</script>

<svelte:head>
  <title>{t('roles.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <div class="flex items-center justify-end">
    <button class="btn btn-sm preset-filled-primary-500 gap-1.5" type="button" on:click={openCreateRole}>
      <Plus class="size-4" aria-hidden="true" />
      {t('roles.addRole')}
    </button>
  </div>

  <Toast {message} />
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    {#if loading}
      <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if roles.length === 0}
      <div class="p-6 text-sm text-surface-500">{t('common.noData')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th scope="col">{t('roles.name')}</th>
              <th scope="col">{t('roles.code')}</th>
              <th scope="col">{t('roles.description')}</th>
              <th scope="col" class="w-28 !text-right">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each roles as role (role.id)}
              <tr>
                <td>
                  <p class="font-medium">{role.name}</p>
                  <p class="max-w-64 truncate text-xs text-surface-500">{role.id}</p>
                </td>
                <td class="whitespace-nowrap font-mono text-xs text-surface-700-300">{role.code}</td>
                <td class="max-w-md truncate text-sm text-surface-600-400">{role.description || '-'}</td>
                <td class="!text-right">
                  <div class="relative inline-flex items-center gap-1">
                    <button
                      class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                      type="button"
                      on:click={() => void openRoleDrawer(role)}
                      aria-label={t('roles.manageRole')}
                      title={t('roles.manageRole')}
                    >
                      <Settings class="size-4" aria-hidden="true" />
                    </button>
                    <button
                      class="btn btn-xs preset-outlined-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                      type="button"
                      on:click={() => (pendingDeleteKey = `role:${role.id}`)}
                      aria-label={t('common.delete')}
                      title={t('common.delete')}
                    >
                      <Trash2 class="size-4" aria-hidden="true" />
                    </button>
                    {#if pendingDeleteKey === `role:${role.id}`}
                      <div class="absolute right-full top-1/2 z-10 mr-1 flex -translate-y-1/2 items-center gap-1 rounded-container border border-surface-200-800 bg-surface-50-950 p-1 shadow-lg">
                        <button
                          class="btn btn-xs preset-filled-error-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                          type="button"
                          on:click={() => void deleteRole(role)}
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
  <div class="fixed inset-y-0 right-0 z-50 flex w-full justify-end" role="dialog" aria-modal="true" aria-labelledby="role-drawer-title" tabindex="-1">
    <form
      class="flex h-full w-full max-w-lg flex-col border-l border-surface-200-800 bg-surface-50-950 text-surface-950-50 shadow-2xl"
      on:submit|preventDefault={saveRole}
    >
      <header class="flex items-center justify-between gap-3 border-b border-surface-200-800 px-5 py-4">
        <div>
          <h2 id="role-drawer-title" class="text-base font-semibold">{editingRole ? t('roles.editRole') : t('roles.createRole')}</h2>
          {#if editingRole}
            <p class="mt-1 max-w-72 truncate text-xs text-surface-500">{editingRole.id}</p>
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

      <div class="flex-1 space-y-5 overflow-y-auto px-5 py-5">
        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.name')}</span>
          <input class="input h-9 w-full bg-surface-50-950 text-sm" type="text" bind:value={roleName} required />
        </label>

        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.code')}</span>
          <input class="input h-9 w-full bg-surface-50-950 font-mono text-sm" type="text" bind:value={roleCode} required disabled={!!editingRole} />
        </label>

        <label class="block">
          <span class="text-sm text-surface-500">{t('roles.description')}</span>
          <textarea class="textarea w-full bg-surface-50-950 text-sm" rows="3" bind:value={roleDescription}></textarea>
        </label>

        {#if editingRole}
          <section class="space-y-3 border-t border-surface-200-800 pt-4">
            <div class="flex items-center gap-2">
              <KeyRound class="size-4 text-surface-500" aria-hidden="true" />
              <h3 class="text-sm font-semibold">{t('roles.permissions')}</h3>
            </div>

            {#if permissionLoading}
              <div class="p-4 text-sm text-surface-500">{t('common.loading')}</div>
            {:else if permissions.length === 0}
              <div class="p-4 text-sm text-surface-500">{t('common.noData')}</div>
            {:else}
              <div class="divide-y divide-surface-200-800 rounded-container border border-surface-200-800">
                {#each permissions as permission (permission.id)}
                  <label class="flex items-center gap-3 px-3 py-2">
                    <input
                      type="checkbox"
                      class="checkbox"
                      checked={rolePermissions.includes(permission.id)}
                      on:change={(event) => void toggleRolePermission(permission, (event.currentTarget as HTMLInputElement).checked)}
                    />
                    <span class="min-w-0 flex-1">
                      <span class="block truncate text-sm">{permission.name}</span>
                      <span class="block truncate font-mono text-xs text-surface-500">{permission.code}</span>
                    </span>
                    <span class="text-xs text-surface-500">{permission.type}</span>
                  </label>
                {/each}
              </div>
            {/if}
          </section>
        {/if}
      </div>

      <footer class="flex justify-end gap-2 border-t border-surface-200-800 bg-surface-100-900 px-5 py-4">
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeDrawer}>{t('common.cancel')}</button>
        <button class="btn btn-sm preset-filled-primary-500" type="submit" disabled={saving || roleName.trim() === '' || roleCode.trim() === ''}>
          {saving ? t('common.loading') : t('common.save')}
        </button>
      </footer>
    </form>
  </div>
{/if}
