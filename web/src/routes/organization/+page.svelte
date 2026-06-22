<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { Popover } from '@skeletonlabs/skeleton-svelte';
  import { onMount } from 'svelte';
  import { Building2, ChevronDown, GitBranch, Network } from 'lucide-svelte';
  import { api, type Department, type Group, type GroupMember, type Organization } from '$lib/api';
  import { t } from '$lib/i18n';

  type Tab = 'organizations' | 'departments' | 'groups';
  type TreeItem = { id: string; name: string; parent_id?: string | null };
  type TreeRow<T extends TreeItem> = { item: T; depth: number };

  let activeTab: Tab = 'organizations';
  let organizations: Organization[] = [];
  let departments: Department[] = [];
  let groups: Group[] = [];
  let total = 0;
  let limit = 25;
  let offset = 0;
  let organizationIdFilter = '';
  let groupTypeFilter = '';
  let searchTerm = '';
  let loading = true;
  let error = '';
  let message = '';
  let saving = false;
  let organizationDialogOpen = false;
  let organizationParentSelectOpen = false;
  let editingOrganization: Organization | null = null;
  let organizationName = '';
  let organizationParentId = '';
  let pendingDeleteId = '';
  let departmentDialogOpen = false;
  let departmentOrganizationSelectOpen = false;
  let departmentParentSelectOpen = false;
  let organizationFilterSelectOpen = false;
  let editingDepartment: Department | null = null;
  let departmentOrganizationId = '';
  let departmentName = '';
  let departmentParentId = '';
  let departmentSourceId = '';
  let departmentExternalId = '';
  let pendingDepartmentDeleteId = '';
  let groupDialogOpen = false;
  let editingGroup: Group | null = null;
  let groupName = '';
  let groupType = 'manual';
  let pendingGroupDeleteId = '';
  let selectedGroup: Group | null = null;
  let groupMembers: GroupMember[] = [];
  let groupMemberTotal = 0;
  let groupMembersLoading = false;
  let memberSearch = '';
  let memberUserId = '';
  let pendingMemberDeleteId = '';

  const treeLimit = 1000;
  const formatDateTime = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');

  const flattenTree = <T extends TreeItem>(items: T[], excludeId = ''): TreeRow<T>[] => {
    const byId = new Map<string, T>();
    const children = new Map<string, T[]>();
    const rows: TreeRow<T>[] = [];

    for (const item of items) {
      if (item.id === excludeId) continue;
      byId.set(item.id, item);
    }

    for (const item of byId.values()) {
      const parentId = item.parent_id || '';
      if (!parentId || !byId.has(parentId)) continue;
      const siblings = children.get(parentId) || [];
      siblings.push(item);
      children.set(parentId, siblings);
    }

    const roots = Array.from(byId.values()).filter((item) => !item.parent_id || !byId.has(item.parent_id));
    const sortByName = (values: T[]) => values.sort((a, b) => a.name.localeCompare(b.name));
    const visit = (item: T, depth: number, path: Set<string>) => {
      if (path.has(item.id)) return;
      rows.push({ item, depth });
      const nextPath = new Set(path);
      nextPath.add(item.id);
      for (const child of sortByName(children.get(item.id) || [])) {
        visit(child, depth + 1, nextPath);
      }
    };

    for (const root of sortByName(roots)) {
      visit(root, 0, new Set());
    }
    return rows;
  };

  const withTreeSearchContext = <T extends TreeItem>(items: T[], matcher: (item: T, query: string) => boolean, query: string): T[] => {
    const normalized = query.trim();
    if (!normalized) return items;
    const byId = new Map(items.map((item) => [item.id, item]));
    const children = new Map<string, T[]>();
    const included = new Set<string>();

    for (const item of items) {
      if (!item.parent_id) continue;
      const siblings = children.get(item.parent_id) || [];
      siblings.push(item);
      children.set(item.parent_id, siblings);
    }

    const includeBranch = (item: T) => {
      if (included.has(item.id)) return;
      included.add(item.id);
      for (const child of children.get(item.id) || []) includeBranch(child);
    };

    const includeAncestors = (item: T) => {
      let current: T | undefined = item;
      const seen = new Set<string>();
      while (current && !seen.has(current.id)) {
        included.add(current.id);
        seen.add(current.id);
        current = current.parent_id ? byId.get(current.parent_id) : undefined;
      }
    };

    for (const item of items) {
      if (!matcher(item, normalized)) continue;
      includeBranch(item);
      includeAncestors(item);
    }

    return items.filter((item) => included.has(item.id));
  };

  const getItemName = <T extends { id: string; name: string }>(items: T[], id: string, fallback = ''): string => {
    if (!id) return fallback;
    return items.find((item) => item.id === id)?.name || id;
  };

  const getDepartmentOrganizationName = (department: Department): string =>
    organizations.find((organization) => organization.id === department.organization_id)?.name || department.organization_id;

  const matchesOrganizationSearch = (item: Organization, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [item.name, item.parent_id, item.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const matchesDepartmentSearch = (item: Department, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [item.name, item.organization_id, item.parent_id, item.source_id, item.external_department_id, item.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const matchesGroupSearch = (item: Group, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [item.name, item.type, item.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const matchesMemberSearch = (item: GroupMember, query: string): boolean => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return true;
    return [item.user_id, item.username, item.display_name, item.email, item.lifecycle_status]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalized));
  };

  const resetPage = () => {
    offset = 0;
    total = 0;
  };

  const loadData = async () => {
    loading = true;
    error = '';
    try {
      if (activeTab === 'organizations') {
        const data = await api.listOrganizations({ limit: treeLimit, offset: 0 });
        organizations = data.items || [];
        total = data.total || 0;
      } else if (activeTab === 'departments') {
        if (organizations.length === 0) {
          const organizationData = await api.listOrganizations({ limit: treeLimit, offset: 0 });
          organizations = organizationData.items || [];
        }
        const data = await api.listDepartments({ organization_id: organizationIdFilter, limit: treeLimit, offset: 0 });
        departments = data.items || [];
        total = data.total || 0;
      } else {
        const data = await api.listGroups({ type: groupTypeFilter, limit, offset });
        groups = data.items || [];
        total = data.total || 0;
      }
    } catch {
      error = t('organization.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const openCreateOrganization = () => {
    editingOrganization = null;
    organizationName = '';
    organizationParentId = '';
    organizationDialogOpen = true;
  };

  const openEditOrganization = (organization: Organization) => {
    editingOrganization = organization;
    organizationName = organization.name;
    organizationParentId = organization.parent_id || '';
    organizationDialogOpen = true;
  };

  const closeOrganizationDialog = () => {
    organizationDialogOpen = false;
    editingOrganization = null;
    saving = false;
  };

  const saveOrganization = async () => {
    saving = true;
    error = '';
    message = '';
    const payload = {
      name: organizationName,
      parent_id: organizationParentId || undefined,
    };
    try {
      if (editingOrganization) {
        await api.updateOrganization(editingOrganization.id, payload);
        message = t('common.updateSuccess');
      } else {
        await api.createOrganization(payload);
        message = t('common.createSuccess');
      }
      closeOrganizationDialog();
      await loadData();
    } catch {
      error = t(editingOrganization ? 'organization.updateFailed' : 'organization.createFailed');
    } finally {
      saving = false;
    }
  };

  const deleteOrganization = async (organization: Organization) => {
    if (pendingDeleteId !== organization.id) {
      pendingDeleteId = organization.id;
      return;
    }
    error = '';
    message = '';
    try {
      await api.deleteOrganization(organization.id);
      pendingDeleteId = '';
      message = t('common.deleteSuccess');
      await loadData();
    } catch {
      error = t('organization.deleteFailed');
    }
  };

  const openCreateDepartment = () => {
    editingDepartment = null;
    departmentOrganizationId = organizationIdFilter;
    departmentName = '';
    departmentParentId = '';
    departmentSourceId = '';
    departmentExternalId = '';
    departmentDialogOpen = true;
  };

  const openEditDepartment = (department: Department) => {
    editingDepartment = department;
    departmentOrganizationId = department.organization_id;
    departmentName = department.name;
    departmentParentId = department.parent_id || '';
    departmentSourceId = department.source_id || '';
    departmentExternalId = department.external_department_id || '';
    departmentDialogOpen = true;
  };

  const closeDepartmentDialog = () => {
    departmentDialogOpen = false;
    editingDepartment = null;
    saving = false;
  };

  const saveDepartment = async () => {
    saving = true;
    error = '';
    message = '';
    try {
      if (editingDepartment) {
        await api.updateDepartment(editingDepartment.id, {
          name: departmentName,
          parent_id: departmentParentId || undefined,
        });
        message = t('common.updateSuccess');
      } else {
        await api.createDepartment({
          organization_id: departmentOrganizationId,
          name: departmentName,
          parent_id: departmentParentId || undefined,
          source_id: departmentSourceId || undefined,
          external_department_id: departmentExternalId || undefined,
        });
        message = t('common.createSuccess');
      }
      closeDepartmentDialog();
      await loadData();
    } catch {
      error = t(editingDepartment ? 'organization.departmentUpdateFailed' : 'organization.departmentCreateFailed');
    } finally {
      saving = false;
    }
  };

  const deleteDepartment = async (department: Department) => {
    if (pendingDepartmentDeleteId !== department.id) {
      pendingDepartmentDeleteId = department.id;
      return;
    }
    error = '';
    message = '';
    try {
      await api.deleteDepartment(department.id);
      pendingDepartmentDeleteId = '';
      message = t('common.deleteSuccess');
      await loadData();
    } catch {
      error = t('organization.departmentDeleteFailed');
    }
  };

  const openCreateGroup = () => {
    editingGroup = null;
    groupName = '';
    groupType = groupTypeFilter || 'manual';
    groupDialogOpen = true;
  };

  const openEditGroup = (group: Group) => {
    editingGroup = group;
    groupName = group.name;
    groupType = group.type || 'manual';
    groupDialogOpen = true;
  };

  const closeGroupDialog = () => {
    groupDialogOpen = false;
    editingGroup = null;
    saving = false;
  };

  const saveGroup = async () => {
    saving = true;
    error = '';
    message = '';
    try {
      if (editingGroup) {
        await api.updateGroup(editingGroup.id, { name: groupName });
        message = t('common.updateSuccess');
      } else {
        await api.createGroup({ name: groupName, type: groupType || undefined });
        message = t('common.createSuccess');
      }
      closeGroupDialog();
      await loadData();
    } catch {
      error = t(editingGroup ? 'organization.groupUpdateFailed' : 'organization.groupCreateFailed');
    } finally {
      saving = false;
    }
  };

  const deleteGroup = async (group: Group) => {
    if (pendingGroupDeleteId !== group.id) {
      pendingGroupDeleteId = group.id;
      return;
    }
    error = '';
    message = '';
    try {
      await api.deleteGroup(group.id);
      pendingGroupDeleteId = '';
      if (selectedGroup?.id === group.id) selectedGroup = null;
      message = t('common.deleteSuccess');
      await loadData();
    } catch {
      error = t('organization.groupDeleteFailed');
    }
  };

  const loadGroupMembers = async (group: Group) => {
    selectedGroup = group;
    memberSearch = '';
    groupMembersLoading = true;
    error = '';
    try {
      const data = await api.listGroupMembers(group.id, { limit: 200 });
      groupMembers = data.items || [];
      groupMemberTotal = data.total || 0;
    } catch {
      error = t('organization.fetchFailed');
    } finally {
      groupMembersLoading = false;
    }
  };

  const addGroupMember = async () => {
    if (!selectedGroup || !memberUserId) return;
    saving = true;
    error = '';
    message = '';
    try {
      await api.addGroupMember(selectedGroup.id, memberUserId);
      memberUserId = '';
      message = t('common.createSuccess');
      await loadGroupMembers(selectedGroup);
    } catch {
      error = t('organization.memberAddFailed');
    } finally {
      saving = false;
    }
  };

  const removeGroupMember = async (member: GroupMember) => {
    if (!selectedGroup) return;
    if (pendingMemberDeleteId !== member.user_id) {
      pendingMemberDeleteId = member.user_id;
      return;
    }
    error = '';
    message = '';
    try {
      await api.removeGroupMember(selectedGroup.id, member.user_id);
      pendingMemberDeleteId = '';
      message = t('common.deleteSuccess');
      await loadGroupMembers(selectedGroup);
    } catch {
      error = t('organization.memberRemoveFailed');
    }
  };

  const selectTab = (tab: Tab) => {
    activeTab = tab;
    searchTerm = '';
    resetPage();
    void loadData();
  };

  const applyFilters = () => {
    resetPage();
    void loadData();
  };

  const resetFilters = () => {
    organizationIdFilter = '';
    organizationFilterSelectOpen = false;
    groupTypeFilter = '';
    resetPage();
    void loadData();
  };

  const previousPage = () => {
    if (offset === 0) return;
    offset = Math.max(0, offset - limit);
    void loadData();
  };

  const nextPage = () => {
    if (offset + limit >= total) return;
    offset += limit;
    void loadData();
  };

  onMount(() => {
    void loadData();
  });

  $: organizationRows = flattenTree(organizations, editingOrganization?.id || '');
  $: departmentParentRows = flattenTree(
    departments.filter((department) => department.organization_id === departmentOrganizationId),
    editingDepartment?.id || ''
  );
  $: filteredOrganizations = withTreeSearchContext(organizations, matchesOrganizationSearch, searchTerm);
  $: filteredDepartments = withTreeSearchContext(departments, matchesDepartmentSearch, searchTerm);
  $: organizationTreeRows = flattenTree(filteredOrganizations);
  $: departmentTreeRows = flattenTree(filteredDepartments);
  $: filteredGroups = groups.filter((item) => matchesGroupSearch(item, searchTerm));
  $: visibleRows =
    activeTab === 'organizations'
      ? filteredOrganizations.length
      : activeTab === 'departments'
        ? filteredDepartments.length
        : filteredGroups.length;
  $: pageStart = total === 0 ? 0 : activeTab === 'groups' ? offset + 1 : 1;
  $: pageEnd = activeTab === 'groups' ? Math.min(offset + limit, total) : Math.min(total, treeLimit);
  $: uniqueGroupTypeCount = new Set(groups.map((group) => group.type).filter(Boolean)).size;
  $: filteredGroupMembers = groupMembers.filter((item) => matchesMemberSearch(item, memberSearch));
  $: activeGroupMemberCount = groupMembers.filter((item) => item.lifecycle_status === 'active').length;
  $: memberWithEmailCount = groupMembers.filter((item) => Boolean(item.email)).length;
</script>

<svelte:head>
  <title>{t('organization.title')}</title>
</svelte:head>

<section class="space-y-4">
  <header class="flex flex-wrap items-end justify-between gap-3">
    <span aria-hidden="true"></span>
    <div class="flex gap-2">
      <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => void loadData()}>{t('common.retry')}</button>
      {#if activeTab === 'organizations'}
        <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openCreateOrganization}>{t('organization.createOrganization')}</button>
      {:else if activeTab === 'departments'}
        <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openCreateDepartment}>{t('organization.createDepartment')}</button>
      {:else}
        <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openCreateGroup}>{t('organization.createGroup')}</button>
      {/if}
    </div>
  </header>

  <div class="flex flex-wrap gap-2" aria-label={t('organization.title')}>
    <button class={`btn btn-sm ${activeTab === 'organizations' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" aria-pressed={activeTab === 'organizations'} on:click={() => selectTab('organizations')}>
      {t('organization.organizations')}
    </button>
    <button class={`btn btn-sm ${activeTab === 'departments' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" aria-pressed={activeTab === 'departments'} on:click={() => selectTab('departments')}>
      {t('organization.departments')}
    </button>
    <button class={`btn btn-sm ${activeTab === 'groups' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" aria-pressed={activeTab === 'groups'} on:click={() => selectTab('groups')}>
      {t('organization.groups')}
    </button>
  </div>

  {#if activeTab !== 'organizations'}
    <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_auto]" on:submit|preventDefault={applyFilters}>
      {#if activeTab === 'departments'}
        <div class="block">
          <span class="mb-1 block text-sm text-surface-500">{t('organization.organization')}</span>
          <Popover open={organizationFilterSelectOpen} onOpenChange={(event) => (organizationFilterSelectOpen = event.open)}>
            <Popover.Trigger class="btn btn-sm preset-outlined-surface-500 w-full justify-between">
              <span class="truncate">{organizationIdFilter ? getItemName(organizations, organizationIdFilter) : t('organization.allOrganizations')}</span>
              <ChevronDown class="size-4" aria-hidden="true" />
            </Popover.Trigger>
            <Popover.Positioner>
              <Popover.Content class="w-80 max-w-[calc(100vw-2rem)] rounded-container bg-surface-50-950 border border-surface-200-800 p-2 shadow-xl">
                <div class="max-h-80 space-y-1 overflow-y-auto" role="tree" aria-label={t('organization.selectOrganization')}>
                  <button class={`btn btn-sm w-full justify-start ${organizationIdFilter === '' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" on:click={() => { organizationIdFilter = ''; organizationFilterSelectOpen = false; }}>
                    {t('organization.allOrganizations')}
                  </button>
                  {#each organizationRows as row (row.item.id)}
                    <button class={`btn btn-sm w-full justify-start ${organizationIdFilter === row.item.id ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" role="treeitem" aria-selected={organizationIdFilter === row.item.id} style={`padding-left: ${0.75 + row.depth * 1.25}rem`} on:click={() => { organizationIdFilter = row.item.id; organizationFilterSelectOpen = false; }}>
                      <Building2 class="size-4 shrink-0" aria-hidden="true" />
                      <span class="truncate">{row.item.name}</span>
                    </button>
                  {/each}
                </div>
              </Popover.Content>
            </Popover.Positioner>
          </Popover>
        </div>
      {:else}
        <label class="block">
          <span class="text-sm text-surface-500">{t('organization.groupType')}</span>
          <input class="input w-full" type="text" bind:value={groupTypeFilter} />
        </label>
      {/if}
      <div class="flex flex-wrap items-end gap-2">
        <button class="btn preset-filled-primary-500" type="submit">{t('organization.filter')}</button>
        <button class="btn preset-outlined-surface-500" type="button" on:click={resetFilters}>{t('organization.reset')}</button>
      </div>
    </form>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_repeat(3,minmax(0,10rem))]">
    <label class="block">
      <span class="text-sm text-surface-500">{t('organization.search')}</span>
      <input class="input w-full" type="search" bind:value={searchTerm} placeholder={t('organization.searchPlaceholder')} />
    </label>
    <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('organization.pageRange')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${pageStart}-${pageEnd}`}</p></article>
    <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('organization.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{visibleRows}</p></article>
    <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{activeTab === 'groups' ? t('organization.groupTypes') : t('dashboard.total')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeTab === 'groups' ? uniqueGroupTypeCount : total}</p></article>
  </section>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}
  {#if message}
    <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if activeTab === 'organizations'}
      {#if organizations.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noOrganizations')}</div>
      {:else if filteredOrganizations.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noSearchResults')}</div>
      {:else}
        <div class="divide-y divide-surface-200-800" role="tree" aria-label={t('organization.organizationTree')}>
          {#each organizationTreeRows as row (row.item.id)}
            <div class="grid gap-3 p-3 md:grid-cols-[minmax(0,1fr)_minmax(0,12rem)_minmax(0,12rem)_auto]" role="treeitem" aria-level={row.depth + 1} aria-selected="false">
              <div class="flex min-w-0 items-center gap-2" style={`padding-left: ${row.depth * 1.25}rem`}>
                <Building2 class="size-4 shrink-0 text-primary-500" aria-hidden="true" />
                <div class="min-w-0">
                  <div class="truncate text-sm font-medium">{row.item.name}</div>
                  <div class="truncate text-xs text-surface-500">{row.item.id}</div>
                </div>
              </div>
              <div class="min-w-0 text-xs text-surface-500">
                <span class="block">{t('organization.parent')}</span>
                <span class="block truncate">{row.item.parent_id ? getItemName(organizations, row.item.parent_id) : t('organization.noParent')}</span>
              </div>
              <div class="text-xs text-surface-500">
                <span class="block">{t('organization.updatedAt')}</span>
                <span class="block whitespace-nowrap">{formatDateTime(row.item.updated_at)}</span>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openEditOrganization(row.item)}>{t('common.edit')}</button>
                <button class="btn preset-tonal-error btn-xs" type="button" on:click={() => void deleteOrganization(row.item)}>
                  {pendingDeleteId === row.item.id ? t('organization.deleteConfirm') : t('common.delete')}
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {:else if activeTab === 'departments'}
      {#if departments.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noDepartments')}</div>
      {:else if filteredDepartments.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noSearchResults')}</div>
      {:else}
        <div class="divide-y divide-surface-200-800" role="tree" aria-label={t('organization.departmentTree')}>
          {#each departmentTreeRows as row (row.item.id)}
            <div class="grid gap-3 p-3 md:grid-cols-[minmax(0,1fr)_minmax(0,12rem)_minmax(0,14rem)_auto]" role="treeitem" aria-level={row.depth + 1} aria-selected="false">
              <div class="flex min-w-0 items-center gap-2" style={`padding-left: ${row.depth * 1.25}rem`}>
                <Network class="size-4 shrink-0 text-primary-500" aria-hidden="true" />
                <div class="min-w-0">
                  <div class="truncate text-sm font-medium">{row.item.name}</div>
                  <div class="truncate text-xs text-surface-500">{row.item.id}</div>
                </div>
              </div>
              <div class="min-w-0 text-xs text-surface-500">
                <span class="block">{t('organization.organization')}</span>
                <span class="block truncate">{getDepartmentOrganizationName(row.item)}</span>
              </div>
              <div class="min-w-0 text-xs text-surface-500">
                <span class="block">{t('organization.externalDepartmentId')}</span>
                <span class="block truncate">{row.item.external_department_id || row.item.source_id || '-'}</span>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openEditDepartment(row.item)}>{t('common.edit')}</button>
                <button class="btn preset-tonal-error btn-xs" type="button" on:click={() => void deleteDepartment(row.item)}>
                  {pendingDepartmentDeleteId === row.item.id ? t('organization.deleteConfirm') : t('common.delete')}
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {:else}
      {#if groups.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noGroups')}</div>
      {:else if filteredGroups.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noSearchResults')}</div>
      {:else}
        <div class="overflow-x-auto">
          <table class="table min-w-full">
            <thead>
              <tr>
                <th>{t('organization.name')}</th>
                <th>{t('organization.type')}</th>
                <th>{t('organization.createdAt')}</th>
                <th>{t('organization.updatedAt')}</th>
                <th>{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {#each filteredGroups as item (item.id)}
                <tr>
                  <td>
                    <div class="space-y-1">
                      <div class="font-medium">{item.name}</div>
                      <div class="text-xs text-surface-500">{item.id}</div>
                    </div>
                  </td>
                  <td>{item.type || '-'}</td>
                  <td class="whitespace-nowrap">{formatDateTime(item.created_at)}</td>
                  <td class="whitespace-nowrap">{formatDateTime(item.updated_at)}</td>
                  <td>
                    <div class="flex flex-wrap gap-2">
                      <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => openEditGroup(item)}>{t('common.edit')}</button>
                      <button class="btn preset-outlined-surface-500 btn-xs" type="button" on:click={() => void loadGroupMembers(item)}>{t('organization.manageMembers')}</button>
                      <button class="btn preset-tonal-error btn-xs" type="button" on:click={() => void deleteGroup(item)}>
                        {pendingGroupDeleteId === item.id ? t('organization.deleteConfirm') : t('common.delete')}
                      </button>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    {/if}

    <div class="flex flex-wrap items-center justify-between gap-3 border-t border-surface-200-800 p-3">
      <span class="text-xs text-surface-500">{t('dashboard.total')}: {total}</span>
      <div class="flex gap-2">
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={previousPage} disabled={offset === 0}>{t('common.previous')}</button>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={nextPage} disabled={offset + limit >= total}>{t('common.next')}</button>
      </div>
    </div>
  </section>

  {#if activeTab === 'groups' && selectedGroup}
    <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
      <div class="flex flex-wrap items-end justify-between gap-3 border-b border-surface-200-800 p-4">
        <div>
          <h2 class="font-semibold">{t('organization.manageMembers')}: {selectedGroup.name}</h2>
          <p class="text-xs text-surface-500">{t('dashboard.total')}: {groupMemberTotal}</p>
        </div>
        <form class="grid w-full gap-3 sm:w-auto sm:grid-cols-[minmax(0,18rem)_auto] sm:items-end" on:submit|preventDefault={addGroupMember}>
          <label class="block">
            <span class="text-sm text-surface-500">{t('organization.userId')}</span>
            <input class="input w-full" type="text" bind:value={memberUserId} required />
          </label>
          <button class="btn preset-filled-primary-500" type="submit" disabled={saving || memberUserId.trim() === ''}>{t('organization.addMember')}</button>
        </form>
      </div>

      <div class="grid gap-3 border-b border-surface-200-800 p-4 text-sm md:grid-cols-[minmax(0,1fr)_repeat(3,minmax(0,9rem))]">
        <label class="block">
          <span class="text-sm text-surface-500">{t('organization.searchMembers')}</span>
          <input class="input w-full" type="search" bind:value={memberSearch} placeholder={t('organization.searchMembersPlaceholder')} />
        </label>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('organization.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredGroupMembers.length} / ${groupMembers.length}`}</p></article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('users.status.active')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeGroupMemberCount}</p></article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('directory.withEmail')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{memberWithEmailCount}</p></article>
      </div>

      {#if groupMembersLoading}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {:else if groupMembers.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noMembers')}</div>
      {:else if filteredGroupMembers.length === 0}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noMemberSearchResults')}</div>
      {:else}
        <div class="overflow-x-auto">
          <table class="table min-w-full">
            <thead>
              <tr>
                <th>{t('organization.username')}</th>
                <th>{t('organization.displayName')}</th>
                <th>{t('organization.email')}</th>
                <th>{t('users.status')}</th>
                <th>{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {#each filteredGroupMembers as member (member.user_id)}
                <tr>
                  <td>
                    <div class="space-y-1">
                      <div class="font-medium">{member.username}</div>
                      <div class="text-xs text-surface-500">{member.user_id}</div>
                    </div>
                  </td>
                  <td>{member.display_name || '-'}</td>
                  <td>{member.email || '-'}</td>
                  <td>{t(`users.status.${member.lifecycle_status}`, member.lifecycle_status)}</td>
                  <td>
                    <button class="btn preset-tonal-error btn-xs" type="button" on:click={() => void removeGroupMember(member)}>
                      {pendingMemberDeleteId === member.user_id ? t('organization.deleteConfirm') : t('common.delete')}
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </section>
  {/if}
</section>

{#if organizationDialogOpen}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="organization-dialog-title" tabindex="-1" on:keydown={(event) => event.key === 'Escape' && closeOrganizationDialog()}>
    <div class="mx-auto mt-10 max-w-lg rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="organization-dialog-title" class="font-semibold">{editingOrganization ? t('organization.editOrganization') : t('organization.createOrganization')}</h2>
        <button class="btn preset-outlined-surface-500" type="button" on:click={closeOrganizationDialog}>{t('common.cancel')}</button>
      </div>

      <form class="space-y-4" on:submit|preventDefault={saveOrganization}>
        <label class="block">
          <span class="text-sm text-surface-500">{t('organization.name')}</span>
          <input class="input w-full" type="text" bind:value={organizationName} required />
        </label>
        <div class="block">
          <span class="mb-1 block text-sm text-surface-500">{t('organization.parent')}</span>
          <Popover open={organizationParentSelectOpen} onOpenChange={(event) => (organizationParentSelectOpen = event.open)}>
            <Popover.Trigger class="btn btn-sm preset-outlined-surface-500 w-full justify-between">
              <span class="truncate">{organizationParentId ? getItemName(organizations, organizationParentId) : t('organization.noParent')}</span>
              <ChevronDown class="size-4" aria-hidden="true" />
            </Popover.Trigger>
            <Popover.Positioner>
              <Popover.Content class="w-80 max-w-[calc(100vw-2rem)] rounded-container bg-surface-50-950 border border-surface-200-800 p-2 shadow-xl">
                <div class="max-h-80 space-y-1 overflow-y-auto" role="tree" aria-label={t('organization.selectParent')}>
                  <button class={`btn btn-sm w-full justify-start ${organizationParentId === '' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" on:click={() => { organizationParentId = ''; organizationParentSelectOpen = false; }}>
                    {t('organization.noParent')}
                  </button>
                  {#each organizationRows as row (row.item.id)}
                    <button class={`btn btn-sm w-full justify-start ${organizationParentId === row.item.id ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" role="treeitem" aria-selected={organizationParentId === row.item.id} style={`padding-left: ${0.75 + row.depth * 1.25}rem`} on:click={() => { organizationParentId = row.item.id; organizationParentSelectOpen = false; }}>
                      <Building2 class="size-4 shrink-0" aria-hidden="true" />
                      <span class="truncate">{row.item.name}</span>
                    </button>
                  {/each}
                </div>
              </Popover.Content>
            </Popover.Positioner>
          </Popover>
        </div>
        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeOrganizationDialog}>{t('common.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={saving || organizationName.trim() === ''}>
            {saving ? t('common.loading') : t('common.save')}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

{#if groupDialogOpen}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="group-dialog-title" tabindex="-1" on:keydown={(event) => event.key === 'Escape' && closeGroupDialog()}>
    <div class="mx-auto mt-10 max-w-lg rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="group-dialog-title" class="font-semibold">{editingGroup ? t('organization.editGroup') : t('organization.createGroup')}</h2>
        <button class="btn preset-outlined-surface-500" type="button" on:click={closeGroupDialog}>{t('common.cancel')}</button>
      </div>

      <form class="space-y-4" on:submit|preventDefault={saveGroup}>
        <label class="block">
          <span class="text-sm text-surface-500">{t('organization.name')}</span>
          <input class="input w-full" type="text" bind:value={groupName} required />
        </label>
        <label class="block">
          <span class="text-sm text-surface-500">{t('organization.groupType')}</span>
          <input class="input w-full" type="text" bind:value={groupType} required disabled={!!editingGroup} />
        </label>
        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeGroupDialog}>{t('common.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={saving || groupName.trim() === '' || groupType.trim() === ''}>
            {saving ? t('common.loading') : t('common.save')}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

{#if departmentDialogOpen}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="department-dialog-title" tabindex="-1" on:keydown={(event) => event.key === 'Escape' && closeDepartmentDialog()}>
    <div class="mx-auto mt-10 max-w-lg rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="department-dialog-title" class="font-semibold">{editingDepartment ? t('organization.editDepartment') : t('organization.createDepartment')}</h2>
        <button class="btn preset-outlined-surface-500" type="button" on:click={closeDepartmentDialog}>{t('common.cancel')}</button>
      </div>

      <form class="space-y-4" on:submit|preventDefault={saveDepartment}>
        <div class="block">
          <span class="mb-1 block text-sm text-surface-500">{t('organization.organization')}</span>
          <Popover open={departmentOrganizationSelectOpen} onOpenChange={(event) => (departmentOrganizationSelectOpen = event.open)}>
            <Popover.Trigger class="btn btn-sm preset-outlined-surface-500 w-full justify-between" disabled={!!editingDepartment}>
              <span class="truncate">{departmentOrganizationId ? getItemName(organizations, departmentOrganizationId) : t('organization.selectOrganization')}</span>
              <ChevronDown class="size-4" aria-hidden="true" />
            </Popover.Trigger>
            <Popover.Positioner>
              <Popover.Content class="w-80 max-w-[calc(100vw-2rem)] rounded-container bg-surface-50-950 border border-surface-200-800 p-2 shadow-xl">
                <div class="max-h-80 space-y-1 overflow-y-auto" role="tree" aria-label={t('organization.selectOrganization')}>
                  {#each organizationRows as row (row.item.id)}
                    <button class={`btn btn-sm w-full justify-start ${departmentOrganizationId === row.item.id ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" role="treeitem" aria-selected={departmentOrganizationId === row.item.id} style={`padding-left: ${0.75 + row.depth * 1.25}rem`} on:click={() => { departmentOrganizationId = row.item.id; departmentParentId = ''; departmentOrganizationSelectOpen = false; }}>
                      <Building2 class="size-4 shrink-0" aria-hidden="true" />
                      <span class="truncate">{row.item.name}</span>
                    </button>
                  {/each}
                </div>
              </Popover.Content>
            </Popover.Positioner>
          </Popover>
        </div>
        <label class="block">
          <span class="text-sm text-surface-500">{t('organization.name')}</span>
          <input class="input w-full" type="text" bind:value={departmentName} required />
        </label>
        <div class="block">
          <span class="mb-1 block text-sm text-surface-500">{t('organization.parentDepartment')}</span>
          <Popover open={departmentParentSelectOpen} onOpenChange={(event) => (departmentParentSelectOpen = event.open)}>
            <Popover.Trigger class="btn btn-sm preset-outlined-surface-500 w-full justify-between" disabled={!departmentOrganizationId}>
              <span class="truncate">{departmentParentId ? getItemName(departments, departmentParentId) : t('organization.noParent')}</span>
              <ChevronDown class="size-4" aria-hidden="true" />
            </Popover.Trigger>
            <Popover.Positioner>
              <Popover.Content class="w-80 max-w-[calc(100vw-2rem)] rounded-container bg-surface-50-950 border border-surface-200-800 p-2 shadow-xl">
                <div class="max-h-80 space-y-1 overflow-y-auto" role="tree" aria-label={t('organization.selectParentDepartment')}>
                  <button class={`btn btn-sm w-full justify-start ${departmentParentId === '' ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" on:click={() => { departmentParentId = ''; departmentParentSelectOpen = false; }}>
                    {t('organization.noParent')}
                  </button>
                  {#each departmentParentRows as row (row.item.id)}
                    <button class={`btn btn-sm w-full justify-start ${departmentParentId === row.item.id ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`} type="button" role="treeitem" aria-selected={departmentParentId === row.item.id} style={`padding-left: ${0.75 + row.depth * 1.25}rem`} on:click={() => { departmentParentId = row.item.id; departmentParentSelectOpen = false; }}>
                      <GitBranch class="size-4 shrink-0" aria-hidden="true" />
                      <span class="truncate">{row.item.name}</span>
                    </button>
                  {/each}
                </div>
              </Popover.Content>
            </Popover.Positioner>
          </Popover>
        </div>
        {#if !editingDepartment}
          <label class="block">
            <span class="text-sm text-surface-500">{t('organization.sourceId')}</span>
            <input class="input w-full" type="text" bind:value={departmentSourceId} />
          </label>
          <label class="block">
            <span class="text-sm text-surface-500">{t('organization.externalDepartmentId')}</span>
            <input class="input w-full" type="text" bind:value={departmentExternalId} />
          </label>
        {/if}
        <div class="flex justify-end gap-2">
          <button class="btn preset-outlined-surface-500" type="button" on:click={closeDepartmentDialog}>{t('common.cancel')}</button>
          <button class="btn preset-filled-primary-500" type="submit" disabled={saving || departmentName.trim() === '' || departmentOrganizationId.trim() === ''}>
            {saving ? t('common.loading') : t('common.save')}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
