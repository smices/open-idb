<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type AdminRoleOption, type AdminUser, type Entity } from '$lib/api';
  import { t } from '$lib/i18n';
  import IdConfirmDialog from '$lib/components/ui/IdConfirmDialog.svelte';
  import IdModal from '$lib/components/ui/IdModal.svelte';
  import { notifyError, notifySuccess } from '$lib/toast';
  import { KeyRound, Pencil, Plus, Trash2 } from 'lucide-svelte';

  type DialogMode = 'create' | 'edit' | 'password';

  let admins: AdminUser[] = [];
  let roles: AdminRoleOption[] = [];
  let entities: Entity[] = [];
  let loading = false;
  let saving = false;
  let error = '';
  let dialogOpen = false;
  let dialogMode: DialogMode = 'create';
  let editing: AdminUser | null = null;
  let confirmDeleteId = '';

  let formUsername = '';
  let formDisplayName = '';
  let formEmail = '';
  let formRole = 'enterprise_admin';
  let formEntityId = '';
  let formStatus = 'active';
  let formPassword = '';

  const roleLabel = (role: string) => roles.find((item) => item.value === role)?.label || role;
  const roleRequiresEntity = (role: string) => Boolean(roles.find((item) => item.value === role)?.requires_entity);
  const entityName = (id?: string) => entities.find((item) => item.id === id)?.name || '';
  const statusLabel = (value: string) => {
    return t(`adminUsers.status.${value}`, value);
  };
  const actionErrorMessage = (err: unknown, fallback: string): string => {
    const message = err instanceof Error ? err.message : '';
    if (message.includes('password does not meet minimum strength requirements')) {
      return t('adminUsers.passwordTooWeak');
    }
    return message || fallback;
  };
  const isStrongPassword = (value: string): boolean =>
    value.length >= 6 && /[A-Za-z]/.test(value) && /\d/.test(value);

  const loadData = async () => {
    loading = true;
    error = '';
    try {
      const [adminData, roleData, entityData] = await Promise.all([
        api.listAdminUsers(),
        api.listAdminRoles(),
        api.listEntities({ limit: 200 }),
      ]);
      admins = adminData.items || [];
      roles = roleData || [];
      entities = entityData.items || [];
      if (!formRole && roles.length) formRole = roles[0].value;
    } catch (err) {
      error = err instanceof Error && err.message !== 'unauthorized' ? err.message : t('adminUsers.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const resetForm = () => {
    formUsername = '';
    formDisplayName = '';
    formEmail = '';
    formRole = roles.find((item) => item.value === 'enterprise_admin')?.value || roles[0]?.value || 'enterprise_admin';
    formEntityId = entities[0]?.id || '';
    formStatus = 'active';
    formPassword = '';
  };

  const openCreate = () => {
    resetForm();
    editing = null;
    dialogMode = 'create';
    dialogOpen = true;
    error = '';
  };

  const openEdit = (admin: AdminUser) => {
    if (admin.protected) return;
    editing = admin;
    formUsername = admin.username;
    formDisplayName = admin.display_name;
    formEmail = admin.email || '';
    formRole = admin.role;
    formEntityId = admin.entity_id || entities[0]?.id || '';
    formStatus = admin.status;
    formPassword = '';
    dialogMode = 'edit';
    dialogOpen = true;
    error = '';
  };

  const openPassword = (admin: AdminUser) => {
    editing = admin;
    formPassword = '';
    dialogMode = 'password';
    dialogOpen = true;
    error = '';
  };

  const closeDialog = () => {
    dialogOpen = false;
    editing = null;
    formPassword = '';
  };

  const saveAdmin = async () => {
    if ((dialogMode === 'create' || dialogMode === 'password') && !isStrongPassword(formPassword)) {
      notifyError(t('adminUsers.passwordTooWeak'));
      return;
    }

    saving = true;
    error = '';
    try {
      if (dialogMode === 'password' && editing) {
        await api.setAdminUserPassword(editing.id, formPassword);
        notifySuccess(t('adminUsers.passwordUpdated'));
        closeDialog();
        return;
      }

      const payload = {
        display_name: formDisplayName,
        email: formEmail,
        role: formRole,
        entity_id: roleRequiresEntity(formRole) ? formEntityId : '',
        status: formStatus,
      };

      if (dialogMode === 'edit' && editing) {
        await api.updateAdminUser(editing.id, payload);
        notifySuccess(t('adminUsers.updated'));
      } else {
        await api.createAdminUser({
          username: formUsername,
          password: formPassword,
          ...payload,
        });
        notifySuccess(t('adminUsers.created'));
      }
      closeDialog();
      await loadData();
    } catch (err) {
      notifyError(actionErrorMessage(err, t('adminUsers.saveFailed')));
    } finally {
      saving = false;
    }
  };

  const deleteAdmin = async (admin: AdminUser) => {
    if (admin.protected) return;
    saving = true;
    error = '';
    try {
      await api.deleteAdminUser(admin.id);
      confirmDeleteId = '';
      notifySuccess(t('adminUsers.deleted'));
      await loadData();
    } catch (err) {
      notifyError(actionErrorMessage(err, t('adminUsers.deleteFailed')));
    } finally {
      saving = false;
    }
  };

  onMount(loadData);
</script>

<svelte:head>
  <title>{t('adminUsers.title')}</title>
</svelte:head>

<section class="space-y-4">
  <div class="flex justify-end">
    <button class="btn btn-sm preset-filled-primary-500 gap-1.5" type="button" onclick={openCreate}>
      <Plus class="size-4" aria-hidden="true" />
      {t('common.add')}
    </button>
  </div>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card overflow-hidden border border-surface-200-800 bg-surface-50-950">
    {#if loading}
      <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if admins.length === 0}
      <div class="p-6 text-sm text-surface-500">{t('adminUsers.noData')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th scope="col">{t('adminUsers.account')}</th>
              <th scope="col">{t('adminUsers.role')}</th>
              <th scope="col">{t('adminUsers.company')}</th>
              <th scope="col" class="text-center">{t('adminUsers.status')}</th>
              <th scope="col">{t('adminUsers.updatedAt')}</th>
              <th scope="col" class="text-right">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each admins as admin}
              <tr>
                <td>
                  <div class="font-medium">{admin.display_name || admin.username}</div>
                  <div class="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-surface-500">
                    <span>{admin.username}</span>
                    {#if admin.email}
                      <span>{admin.email}</span>
                    {/if}
                  </div>
                </td>
                <td>{roleLabel(admin.role)}</td>
                <td>{admin.entity_name || entityName(admin.entity_id) || '-'}</td>
                <td class="text-center">
                  <span class="inline-flex items-center justify-center gap-1.5 text-sm">
                    <span class={`size-2 rounded-full ${admin.status === 'active' ? 'bg-success-500' : 'bg-error-500'}`}></span>
                    {statusLabel(admin.status)}
                  </span>
                </td>
                <td class="whitespace-nowrap text-sm">{new Date(admin.updated_at).toLocaleString()}</td>
                <td class="text-right">
                  <div class="relative inline-flex items-center justify-end gap-1">
                    <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" onclick={() => openPassword(admin)} aria-label={t('adminUsers.changePassword')} title={t('adminUsers.changePassword')}>
                      <KeyRound class="size-4" aria-hidden="true" />
                    </button>
                    {#if !admin.protected}
                      <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" onclick={() => openEdit(admin)} aria-label={t('common.edit')} title={t('common.edit')}>
                        <Pencil class="size-4" aria-hidden="true" />
                      </button>
                      <IdConfirmDialog
                        open={confirmDeleteId === admin.id}
                        triggerLabel={t('common.delete')}
                        confirmLabel={t('common.confirmDelete')}
                        triggerClass="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
                        disabled={saving}
                        onOpenChange={(open) => (confirmDeleteId = open ? admin.id : '')}
                        onConfirm={() => deleteAdmin(admin)}
                      >
                        {#snippet trigger()}
                          <Trash2 class="size-4" aria-hidden="true" />
                        {/snippet}
                      </IdConfirmDialog>
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

<IdModal
  open={dialogOpen}
  title={dialogMode === 'password' ? t('adminUsers.changePassword') : dialogMode === 'edit' ? t('adminUsers.editTitle') : t('adminUsers.createTitle')}
  subtitle={editing?.username || t('adminUsers.loginAccount')}
  onClose={closeDialog}
>
  <form
    id="admin-user-dialog-form"
    class="space-y-4"
    onsubmit={(event) => {
      event.preventDefault();
      saveAdmin();
    }}
  >
    {#if dialogMode !== 'password'}
      <label class="block">
        <span class="text-sm text-surface-500">{t('adminUsers.loginAccount')}</span>
        <input class="input h-9 w-full bg-surface-50-950" type="text" bind:value={formUsername} required disabled={dialogMode === 'edit'} />
      </label>

      <label class="block">
        <span class="text-sm text-surface-500">{t('adminUsers.displayName')}</span>
        <input class="input h-9 w-full bg-surface-50-950" type="text" bind:value={formDisplayName} required />
      </label>

      <label class="block">
        <span class="text-sm text-surface-500">{t('adminUsers.email')}</span>
        <input class="input h-9 w-full bg-surface-50-950" type="email" bind:value={formEmail} />
      </label>

      <label class="block">
        <span class="text-sm text-surface-500">{t('adminUsers.role')}</span>
        <select class="input h-9 w-full bg-surface-50-950" bind:value={formRole}>
          {#each roles as role}
            <option value={role.value}>{role.label}</option>
          {/each}
        </select>
      </label>

      {#if roleRequiresEntity(formRole)}
        <label class="block">
          <span class="text-sm text-surface-500">{t('adminUsers.company')}</span>
          <select class="input h-9 w-full bg-surface-50-950" bind:value={formEntityId} required>
            {#each entities as entity}
              <option value={entity.id}>{entity.name}</option>
            {/each}
          </select>
        </label>
      {/if}

      {#if dialogMode === 'edit'}
        <label class="block">
          <span class="text-sm text-surface-500">{t('adminUsers.status')}</span>
          <select class="input h-9 w-full bg-surface-50-950" bind:value={formStatus}>
            <option value="active">{t('adminUsers.status.active')}</option>
            <option value="disabled">{t('adminUsers.status.disabled')}</option>
            <option value="locked">{t('adminUsers.status.locked')}</option>
          </select>
        </label>
      {/if}
    {/if}

    {#if dialogMode === 'create' || dialogMode === 'password'}
      <label class="block">
        <span class="text-sm text-surface-500">{t('adminUsers.newPassword')}</span>
        <input class="input h-9 w-full bg-surface-50-950" type="password" bind:value={formPassword} required autocomplete="new-password" />
      </label>
      <p class="text-xs text-surface-500">{t('adminUsers.passwordHelp')}</p>
    {/if}
  </form>

  {#snippet footer()}
    <button class="btn btn-sm preset-outlined-surface-500" type="button" onclick={closeDialog}>{t('common.cancel')}</button>
    <button class="btn btn-sm preset-filled-primary-500" type="button" disabled={saving} onclick={saveAdmin}>{saving ? t('common.loading') : t('common.save')}</button>
  {/snippet}
</IdModal>
