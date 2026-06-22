<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import {
    api,
    type Application,
    type ApplicationAssignment,
    type Department,
    type Group,
    type Role,
    type User,
  } from '$lib/api';

  let applications: Application[] = [];
  let users: User[] = [];
  let groups: Group[] = [];
  let departments: Department[] = [];
  let roles: Role[] = [];
  let selectedAppId = '';
  let assignments: ApplicationAssignment[] = [];
  let loading = true;
  let assignmentLoading = false;
  let subjectLoading = false;
  let saving = false;
  let pendingDeleteId = '';
  let message = '';
  let error = '';
  let assignmentSearch = '';

  let formSubjectType = 'user';
  let formSubjectId = '';
  let formEffect = 'allow';

  const subjectTypeLabel = (value: string): string => {
    const labels: Record<string, string> = {
      user: t('assignments.subjectUser'),
      group: t('assignments.subjectGroup'),
      department: t('assignments.subjectDept'),
      role: t('assignments.subjectRole'),
    };
    return labels[value] || value;
  };

  const effectLabel = (value: string): string => {
    const labels: Record<string, string> = {
      allow: t('assignments.allow'),
      deny: t('assignments.deny'),
    };
    return labels[value] || value;
  };

  const subjectOptions = (type: string): { id: string; label: string; meta: string }[] => {
    if (type === 'user') {
      return users.map((user) => ({
        id: user.id,
        label: user.display_name || user.username,
        meta: user.username,
      }));
    }
    if (type === 'group') {
      return groups.map((group) => ({
        id: group.id,
        label: group.name,
        meta: group.type,
      }));
    }
    if (type === 'department') {
      return departments.map((department) => ({
        id: department.id,
        label: department.name,
        meta: department.organization_id,
      }));
    }
    if (type === 'role') {
      return roles.map((role) => ({
        id: role.id,
        label: role.name,
        meta: role.code,
      }));
    }
    return [];
  };

  const subjectName = (type: string, id: string): string => {
    const match = subjectOptions(type).find((option) => option.id === id);
    return match ? match.label : id;
  };

  const subjectMeta = (type: string, id: string): string => {
    const match = subjectOptions(type).find((option) => option.id === id);
    return match?.meta || id;
  };

  const includesQuery = (value: unknown, query: string): boolean => String(value ?? '').toLowerCase().includes(query.trim().toLowerCase());

  const matchesAssignmentSearch = (item: ApplicationAssignment, query: string): boolean => {
    if (!query.trim()) return true;
    return [
      item.id,
      item.subject_type,
      item.subject_id,
      subjectTypeLabel(item.subject_type),
      subjectName(item.subject_type, item.subject_id),
      subjectMeta(item.subject_type, item.subject_id),
      item.effect,
      effectLabel(item.effect),
    ].some((value) => includesQuery(value, query));
  };

  const loadApplications = async () => {
    try {
      const data = await api.listApplications({ limit: 200 });
      applications = data.applications || [];
    } catch {
      error = t('applications.fetchFailed');
    }
  };

  const loadSubjectCatalogs = async () => {
    subjectLoading = true;
    try {
      const [userData, groupData, departmentData, roleData] = await Promise.all([
        api.listUsers({ limit: 200 }),
        api.listGroups({ limit: 200 }),
        api.listDepartments({ limit: 200 }),
        api.listRoles({ limit: 200 }),
      ]);
      users = userData.items || [];
      groups = groupData.items || [];
      departments = departmentData.items || [];
      const normalizedRoles = roleData as { roles?: Role[]; items?: Role[] };
      roles = normalizedRoles.roles || normalizedRoles.items || [];
    } catch {
      error = t('assignments.subjectsFetchFailed');
    } finally {
      subjectLoading = false;
    }
  };

  const changeSubjectType = (value: string) => {
    formSubjectType = value;
    formSubjectId = '';
  };

  const loadAssignments = async () => {
    if (!selectedAppId) {
      assignments = [];
      return;
    }

    assignmentLoading = true;
    try {
      const data = await api.listAssignments(selectedAppId, { limit: 200 });
      assignments = data.assignments || [];
    } catch {
      error = t('assignments.fetchFailed');
    } finally {
      assignmentLoading = false;
    }
  };

  const selectApp = (id: string) => {
    selectedAppId = id;
    assignmentSearch = '';
    void loadAssignments();
  };

  const createAssignment = async () => {
    if (!selectedAppId || !formSubjectId) return;
    saving = true;
    error = '';
    message = '';
    try {
      await api.createAssignment(selectedAppId, {
        subject_type: formSubjectType,
        subject_id: formSubjectId,
        effect: formEffect,
      });
      message = t('assignments.createSuccess');
      formSubjectId = '';
      await loadAssignments();
    } catch {
      error = t('assignments.createFailed');
    } finally {
      saving = false;
    }
  };

  const deleteAssignment = async (id: string) => {
    error = '';
    message = '';
    try {
      await api.deleteAssignment(id);
      pendingDeleteId = '';
      message = t('assignments.deleteSuccess');
      await loadAssignments();
    } catch {
      error = t('assignments.deleteFailed');
    }
  };

  onMount(async () => {
    loading = true;
    try {
      await Promise.all([loadApplications(), loadSubjectCatalogs()]);
      if (applications[0]) {
        selectedAppId = applications[0].id;
        await loadAssignments();
      }
    } finally {
      loading = false;
    }
  });

  $: filteredAssignments = assignments.filter((item) => matchesAssignmentSearch(item, assignmentSearch));
  $: allowAssignmentCount = assignments.filter((item) => item.effect === 'allow').length;
  $: denyAssignmentCount = assignments.filter((item) => item.effect === 'deny').length;
  $: subjectTypeCount = new Set(assignments.map((item) => item.subject_type).filter(Boolean)).size;
</script>

<svelte:head>
  <title>{t('assignments.title')}</title>
</svelte:head>

<section class="space-y-4">
  {#if message}
    <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
  {/if}
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <label class="block">
    <span class="text-sm text-surface-500">{t('assignments.selectApp')}</span>
    <select class="input max-w-md" value={selectedAppId} on:change={(event) => selectApp((event.currentTarget as HTMLSelectElement).value)}>
      {#each applications as app}
        <option value={app.id}>{app.name}</option>
      {/each}
    </select>
  </label>

  <section class="card bg-surface-50-950 border border-surface-200-800 p-4">
    <h2 class="font-semibold mb-3">{t('assignments.create')}</h2>
    <form class="grid gap-2 md:grid-cols-4" on:submit|preventDefault={createAssignment}>
      <label>
        <span class="text-sm text-surface-500">{t('assignments.subjectType')}</span>
        <select class="input w-full" value={formSubjectType} on:change={(event) => changeSubjectType((event.currentTarget as HTMLSelectElement).value)}>
          <option value="user">{t('assignments.subjectUser')}</option>
          <option value="group">{t('assignments.subjectGroup')}</option>
          <option value="department">{t('assignments.subjectDept')}</option>
          <option value="role">{t('assignments.subjectRole')}</option>
        </select>
      </label>
      <label>
        <span class="text-sm text-surface-500">{t('assignments.subjectId')}</span>
        {#if subjectOptions(formSubjectType).length}
          <select class="input w-full" bind:value={formSubjectId} required disabled={subjectLoading}>
            <option value="">{subjectLoading ? t('common.loading') : t('assignments.selectSubject')}</option>
            {#each subjectOptions(formSubjectType) as option}
              <option value={option.id}>{option.label} · {option.meta}</option>
            {/each}
          </select>
        {:else}
          <input class="input w-full" type="text" bind:value={formSubjectId} required placeholder={t('assignments.subjectIdPlaceholder')} disabled={subjectLoading} />
        {/if}
      </label>
      <label>
        <span class="text-sm text-surface-500">{t('assignments.effect')}</span>
        <select class="input w-full" bind:value={formEffect}>
          <option value="allow">{t('assignments.allow')}</option>
          <option value="deny">{t('assignments.deny')}</option>
        </select>
      </label>
      <div class="flex items-end">
        <button class="btn preset-filled-primary-500 w-full" type="submit" disabled={saving || !selectedAppId || !formSubjectId}>
          {saving ? t('common.loading') : t('assignments.createTitle')}
        </button>
      </div>
    </form>
  </section>

  <section class="card bg-surface-50-950 border border-surface-200-800 p-4">
    {#if loading || assignmentLoading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if !assignments.length}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('assignments.noRoles')}</div>
    {:else}
      <div class="mb-4 space-y-3">
        <h2 class="font-semibold">{t('assignments.title')}</h2>
        <label class="block">
          <span class="text-sm text-surface-500">{t('assignments.search')}</span>
          <input class="input w-full" type="search" bind:value={assignmentSearch} placeholder={t('assignments.searchPlaceholder')} />
        </label>
        <div class="grid gap-3 text-sm sm:grid-cols-4">
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('applications.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredAssignments.length} / ${assignments.length}`}</p></article>
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('assignments.allow')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{allowAssignmentCount}</p></article>
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('assignments.deny')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{denyAssignmentCount}</p></article>
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('assignments.subjectTypes')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{subjectTypeCount}</p></article>
        </div>
      </div>
      {#if filteredAssignments.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('assignments.noSearchResults')}</div>
      {:else}
        <div class="divide-y divide-surface-200-800">
          {#each filteredAssignments as item}
            <article class="py-3">
              <header class="flex flex-wrap justify-between gap-2">
                <div>
                  <div class="font-medium">{subjectTypeLabel(item.subject_type)}</div>
                  <div class="text-sm">{subjectName(item.subject_type, item.subject_id)}</div>
                  <div class="text-xs text-surface-500">{subjectMeta(item.subject_type, item.subject_id)}</div>
                </div>
                <span class={`badge ${item.effect === 'allow' ? 'preset-tonal-success' : 'preset-tonal-error'}`}>{effectLabel(item.effect)}</span>
              </header>
            <div class="mt-3 flex justify-end">
              <button
                class="btn preset-tonal-error btn-xs"
                type="button"
                on:click={() => (pendingDeleteId === item.id ? void deleteAssignment(item.id) : (pendingDeleteId = item.id))}
              >
                {pendingDeleteId === item.id ? t('common.confirmDelete') : t('common.delete')}
              </button>
            </div>
          </article>
        {/each}
      </div>
      {/if}
    {/if}
  </section>
</section>
