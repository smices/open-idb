<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { Building2, ChevronRight, ChevronsDownUp, ChevronsUpDown, Copy, LoaderCircle, Network, RotateCcw, Search, UserRound } from 'lucide-svelte';
  import { TreeView, createTreeViewCollection, type TreeViewRootProps } from '@skeletonlabs/skeleton-svelte';
  import { api, type DirectoryUser, type OrganizationTreeNode, type OrganizationTreeNodeKind } from '$lib/api';
  import { t } from '$lib/i18n';
  import IdModal from '$lib/components/ui/IdModal.svelte';
  import { notifyError, notifySuccess } from '$lib/toast';

  type TreeNode = OrganizationTreeNode & {
    children?: TreeNode[];
    childrenCount?: number;
    search_text: string;
  };

  let loading = $state(true);
  let searchText = $state('');
  let rootNodes = $state<TreeNode[]>([]);
  let searchNodes = $state<TreeNode[]>([]);
  let expandedValue = $state<string[]>([]);
  let selectedValue = $state<string[]>([]);
  let searchLoading = $state(false);
  let searchTimer: ReturnType<typeof setTimeout> | undefined;
  let selectedUser = $state<DirectoryUser | null>(null);
  let detailOpen = $state(false);
  let detailLoading = $state(false);
  let copiedValue = $state('');

  const pageLimit = 100;
  const createOrganizationCollection = (children: TreeNode[]) =>
    createTreeViewCollection<TreeNode>({
      nodeToValue: (node) => node.id,
      nodeToString: (node) => node.name,
      rootNode: {
        id: 'root',
        kind: 'organization',
        name: '',
        has_children: true,
        search_text: '',
        children,
      } as TreeNode,
    });
  let collection = $state(createOrganizationCollection([]));
  const formatDateTime = (value?: string): string => (value ? new Date(value).toLocaleString() : '-');
  const userInlineLabel = (node: TreeNode): string => {
    const name = node.name || '-';
    const englishName = node.english_name || '';
    const email = node.email || '';
    return `${name}${englishName ? `（${englishName}）` : ''}${email ? ` ${email}` : ''}`;
  };
  const nodeSearchText = (node: OrganizationTreeNode): string =>
    [node.name, node.english_name, node.id, node.email, node.phone, node.status, node.external_department_id].filter(Boolean).join(' ');
  const showToast = (message: string, variant: 'success' | 'error' = 'success') => {
    if (variant === 'error') notifyError(message);
    else notifySuccess(message);
  };

  const toTreeNode = (node: OrganizationTreeNode, children?: TreeNode[]): TreeNode => ({
    ...node,
    search_text: nodeSearchText(node),
    children,
    childrenCount: children?.length || (node.has_children ? 1 : undefined),
  });

  const isBranch = (node: TreeNode): boolean => Boolean(node.has_children || node.children?.length || node.childrenCount);

  const loadChildren: TreeViewRootProps<TreeNode>['loadChildren'] = async (details) => {
    const node = details.node;
    if (node.kind === 'user' || !node.has_children) return [];
    try {
      const data = await api.listOrganizationTreeChildren({
        kind: node.kind as OrganizationTreeNodeKind,
        id: node.id,
        limit: pageLimit,
        offset: 0,
      });
      return ((data.items || []) as OrganizationTreeNode[]).map((child) => toTreeNode(child));
    } catch {
      showToast(t('organization.fetchFailed'), 'error');
      return [];
    }
  };

  const onLoadChildrenComplete: TreeViewRootProps<TreeNode>['onLoadChildrenComplete'] = (details) => {
    collection = details.collection;
    const nodes = (details.collection.rootNode.children || []) as TreeNode[];
    if (searchText.trim()) {
      searchNodes = nodes;
    } else {
      rootNodes = nodes;
    }
  };

  const expandAll = () => {
    expandedValue = collection.getBranchValues();
  };

  const collapseAll = () => {
    expandedValue = [];
  };

  const loadTree = async () => {
    loading = true;
    try {
      const data = await api.getOrganizationTreeRoot({ limit: pageLimit, offset: 0 });
      const children = (data.children || []).map((item) => toTreeNode(item));
      const root = toTreeNode(data.root, children);
      rootNodes = [root];
      searchNodes = [];
      collection = createOrganizationCollection(rootNodes);
      expandedValue = [root.id];
      selectedValue = [];
    } catch {
      showToast(t('organization.fetchFailed'), 'error');
    } finally {
      loading = false;
    }
  };

  const resetFilters = () => {
    searchText = '';
    searchNodes = [];
    collection = createOrganizationCollection(rootNodes);
    searchLoading = false;
    if (searchTimer) clearTimeout(searchTimer);
  };

  const searchOrganizationTree = async (query: string) => {
    const normalized = query.trim();
    if (!normalized) {
      searchNodes = [];
      searchLoading = false;
      return;
    }
    searchLoading = true;
    try {
      const data = await api.searchOrganizationTree({ q: normalized, limit: pageLimit, offset: 0 });
      searchNodes = (data.items || []).map((item) => toTreeNode(item));
      collection = createOrganizationCollection(searchNodes);
      expandedValue = collection.getBranchValues();
    } catch {
      showToast(t('organization.fetchFailed'), 'error');
    } finally {
      searchLoading = false;
    }
  };

  const onSearchInput = (event: Event) => {
    searchText = (event.currentTarget as HTMLInputElement).value;
    if (searchTimer) clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      void searchOrganizationTree(searchText);
    }, 250);
  };

  const openUserDetails = async (item: TreeNode) => {
    selectedValue = [item.id];
    detailOpen = true;
    detailLoading = true;
    selectedUser = {
      id: item.id,
      entity_id: '',
      source_id: item.source_id || '',
      external_user_id: '',
      name: item.name,
      english_name: item.english_name,
      employee_no: item.employee_no,
      job_title: item.job_title,
      email: item.email,
      phone: item.phone,
      status: item.status || '',
      last_synced_at: '',
      created_at: '',
      updated_at: item.updated_at || '',
    };
    try {
      selectedUser = await api.getDirectoryUser(item.id);
    } catch {
      showToast(t('directory.detailFetchFailed'), 'error');
    } finally {
      detailLoading = false;
    }
  };

  const copyValue = async (value?: string) => {
    if (!value || typeof navigator === 'undefined' || !navigator.clipboard) return;
    await navigator.clipboard.writeText(value);
    copiedValue = value;
    setTimeout(() => {
      if (copiedValue === value) copiedValue = '';
    }, 1200);
  };

  const copyIconLabel = (value?: string): string => (copiedValue === value ? t('common.copied') : t('common.copy'));

  const closeDetails = () => {
    detailOpen = false;
    detailLoading = false;
    selectedUser = null;
  };

  onMount(() => {
    void loadTree();
    return () => {
      if (searchTimer) clearTimeout(searchTimer);
    };
  });
</script>

<svelte:head>
  <title>{t('organization.title')}</title>
</svelte:head>

<section class="space-y-4">
  <section class="card bg-surface-50-950 border border-surface-200-800 flex flex-wrap items-center justify-between gap-2 p-3">
    <label class="relative min-w-56 flex-1 sm:max-w-md">
      <span class="sr-only">{t('organization.search')}</span>
      <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
      <input class="input h-8 w-full pl-9 text-sm" type="search" value={searchText} oninput={onSearchInput} placeholder={t('organization.searchPlaceholder')} />
    </label>

    <div class="flex items-center gap-2">
      <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0" type="button" onclick={expandAll} aria-label={t('common.expand')} title={t('common.expand')}>
        <ChevronsUpDown class="size-4" aria-hidden="true" />
      </button>
      <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0" type="button" onclick={collapseAll} aria-label={t('common.collapse')} title={t('common.collapse')}>
        <ChevronsDownUp class="size-4" aria-hidden="true" />
      </button>
      <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0" type="button" onclick={() => void loadTree()} aria-label={t('common.retry')} title={t('common.retry')}>
        <RotateCcw class="size-4" aria-hidden="true" />
      </button>
      <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0" type="button" onclick={resetFilters} aria-label={t('common.reset')} title={t('common.reset')}>
        <RotateCcw class="size-4" aria-hidden="true" />
      </button>
    </div>
  </section>

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    {#if loading || searchLoading}
      <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if rootNodes.length === 0}
      <div class="m-3 rounded-container border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noDepartments')}</div>
    {:else if searchText.trim() && searchNodes.length === 0}
      <div class="m-3 rounded-container border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noSearchResults')}</div>
    {:else}
      <TreeView
        {collection}
        {expandedValue}
        {selectedValue}
        {loadChildren}
        {onLoadChildrenComplete}
        onExpandedChange={(details) => (expandedValue = details.expandedValue)}
        onSelectionChange={(details) => {
          selectedValue = details.selectedValue;
          const node = details.selectedNodes[0];
          if (node?.kind === 'user') void openUserDetails(node);
        }}
      >
        <TreeView.Tree class="organization-tree space-y-0.5 p-3 text-sm" aria-label={t('organization.title')}>
          {#each collection.rootNode.children || [] as node, index (node.id)}
            {@render treeNode(node, [index])}
          {/each}
        </TreeView.Tree>
      </TreeView>
    {/if}
  </section>
</section>

{#snippet treeNode(node: TreeNode, indexPath: number[])}
  <TreeView.NodeProvider value={{ node, indexPath }}>
  {#if isBranch(node)}
    <TreeView.Branch>
      <TreeView.BranchControl class="organization-tree-row flex min-h-7 w-full min-w-0 items-center gap-1.5 rounded px-1.5 py-0.5 text-left">
        <TreeView.BranchIndicator class="organization-tree-muted data-loading:hidden">
          <ChevronRight class="size-4 shrink-0 transition-transform" aria-hidden="true" />
        </TreeView.BranchIndicator>
        <TreeView.BranchIndicator class="organization-tree-muted hidden animate-spin data-loading:inline">
          <LoaderCircle class="size-4 shrink-0" aria-hidden="true" />
        </TreeView.BranchIndicator>
        <TreeView.BranchText class="flex min-w-0 items-center gap-1.5">
        {#if node.kind === 'company' || node.kind === 'organization'}
          <Building2 class="size-4 shrink-0 text-primary-600-400" aria-hidden="true" />
        {:else}
          <Network class="size-4 shrink-0 text-primary-600-400" aria-hidden="true" />
        {/if}
        <span class="truncate text-sm font-medium">{node.name || '-'}</span>
        </TreeView.BranchText>
      </TreeView.BranchControl>
      <TreeView.BranchContent>
        <TreeView.BranchIndentGuide class="organization-tree-line ml-3 border-l pl-3" />
          {#each node.children || [] as childNode, childIndex (childNode.id)}
            {@render treeNode(childNode, [...indexPath, childIndex])}
          {/each}
      </TreeView.BranchContent>
    </TreeView.Branch>
  {:else}
    <TreeView.Item class="organization-tree-row flex min-h-7 w-full min-w-0 items-center gap-1.5 rounded px-1.5 py-0.5 text-left" onclick={() => void openUserDetails(node)}>
      <UserRound class="organization-tree-muted size-4 shrink-0" aria-hidden="true" />
      <span class="min-w-0 flex-1 truncate text-sm font-medium">{userInlineLabel(node)}</span>
    </TreeView.Item>
  {/if}
  </TreeView.NodeProvider>
{/snippet}

<IdModal open={detailOpen && Boolean(selectedUser)} title={t('directory.details')} onClose={closeDetails}>
  {#if selectedUser}
      {#if detailLoading}
        <div class="rounded-container border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {:else}
        <dl class="grid gap-3 text-sm md:grid-cols-2">
          <div>
            <dt class="text-surface-500">{t('directory.name')}</dt>
            <dd class="font-medium">{selectedUser.name || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.englishName')}</dt>
            <dd class="font-medium">{selectedUser.english_name || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.employeeNo')}</dt>
            <dd class="font-medium">{selectedUser.employee_no || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.jobTitle')}</dt>
            <dd class="font-medium">{selectedUser.job_title || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.userId')}</dt>
            <dd class="flex min-w-0 items-center gap-2 font-mono text-xs">
              <span class="break-all">{selectedUser.id}</span>
              <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" onclick={() => void copyValue(selectedUser?.id)} aria-label={copyIconLabel(selectedUser.id)} title={copyIconLabel(selectedUser.id)}>
                <Copy class="size-3" aria-hidden="true" />
              </button>
            </dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.status')}</dt>
            <dd class="font-medium">{selectedUser.status || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.externalUserId')}</dt>
            <dd class="flex min-w-0 items-center gap-2 font-mono text-xs">
              <span class="break-all">{selectedUser.external_user_id || '-'}</span>
              {#if selectedUser.external_user_id}
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" onclick={() => void copyValue(selectedUser?.external_user_id)} aria-label={copyIconLabel(selectedUser.external_user_id)} title={copyIconLabel(selectedUser.external_user_id)}>
                  <Copy class="size-3" aria-hidden="true" />
                </button>
              {/if}
            </dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.email')}</dt>
            <dd class="flex min-w-0 items-center gap-2">
              <span class="break-all">{selectedUser.email || '-'}</span>
              {#if selectedUser.email}
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" onclick={() => void copyValue(selectedUser?.email)} aria-label={copyIconLabel(selectedUser.email)} title={copyIconLabel(selectedUser.email)}>
                  <Copy class="size-3" aria-hidden="true" />
                </button>
              {/if}
            </dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.phone')}</dt>
            <dd class="flex min-w-0 items-center gap-2">
              <span class="break-all">{selectedUser.phone || '-'}</span>
              {#if selectedUser.phone}
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" onclick={() => void copyValue(selectedUser?.phone)} aria-label={copyIconLabel(selectedUser.phone)} title={copyIconLabel(selectedUser.phone)}>
                  <Copy class="size-3" aria-hidden="true" />
                </button>
              {/if}
            </dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.externalUnionId')}</dt>
            <dd class="break-all font-mono text-xs">{selectedUser.external_union_id || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.externalOpenId')}</dt>
            <dd class="break-all font-mono text-xs">{selectedUser.external_open_id || '-'}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.lastSyncedAt')}</dt>
            <dd class="font-medium">{formatDateTime(selectedUser.last_synced_at)}</dd>
          </div>
          <div>
            <dt class="text-surface-500">{t('directory.updatedAt')}</dt>
            <dd class="font-medium">{formatDateTime(selectedUser.updated_at)}</dd>
          </div>
        </dl>
      {/if}
  {/if}
</IdModal>
