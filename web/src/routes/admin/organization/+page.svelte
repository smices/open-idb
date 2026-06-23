<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { Building2, ChevronRight, ChevronsDownUp, ChevronsUpDown, Copy, LoaderCircle, Network, RotateCcw, Search, UserRound, X } from 'lucide-svelte';
  import { api, type DirectoryUser, type OrganizationTreeNode, type OrganizationTreeNodeKind } from '$lib/api';
  import { t } from '$lib/i18n';
  import Toast from '$lib/components/ui/Toast.svelte';

  type TreeNode = OrganizationTreeNode & {
    children?: TreeNode[];
    childrenCount?: number;
    search_text: string;
  };

  let loading = true;
  let toastMessage = '';
  let toastVariant: 'success' | 'error' = 'success';
  let searchText = '';
  let rootNodes: TreeNode[] = [];
  let searchNodes: TreeNode[] = [];
  let visibleNodes: TreeNode[] = [];
  let expandedValue: string[] = [];
  let searchLoading = false;
  let searchTimer: ReturnType<typeof setTimeout> | undefined;
  let loadingNodeIds: string[] = [];
  let selectedUser: DirectoryUser | null = null;
  let detailOpen = false;
  let detailLoading = false;
  let copiedValue = '';

  const pageLimit = 100;
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
    toastMessage = message;
    toastVariant = variant;
    setTimeout(() => {
      if (toastMessage === message) toastMessage = '';
    }, 2400);
  };

  const toTreeNode = (node: OrganizationTreeNode, children?: TreeNode[]): TreeNode => ({
    ...node,
    search_text: nodeSearchText(node),
    children,
    childrenCount: children?.length || (node.has_children ? 1 : undefined),
  });

  const isExpanded = (node: TreeNode): boolean => expandedValue.includes(node.id);
  const isLoadingNode = (node: TreeNode): boolean => loadingNodeIds.includes(node.id);
  const isBranch = (node: TreeNode): boolean => Boolean(node.has_children || node.children?.length || node.childrenCount);
  const collectExpandableIds = (nodes: TreeNode[]): string[] =>
    nodes.flatMap((node) => (isBranch(node) ? [node.id, ...collectExpandableIds(node.children || [])] : []));
  const updateNode = (nodes: TreeNode[], id: string, updater: (node: TreeNode) => TreeNode): TreeNode[] =>
    nodes.map((node) => {
      if (node.id === id) return updater(node);
      if (node.children?.length) return { ...node, children: updateNode(node.children, id, updater) };
      return node;
    });

  const loadNodeChildren = async (node: TreeNode) => {
    if (node.kind === 'user' || !node.has_children || node.children?.length || isLoadingNode(node)) return;
    loadingNodeIds = [...loadingNodeIds, node.id];
    try {
      const data = await api.listOrganizationTreeChildren({
        kind: node.kind as OrganizationTreeNodeKind,
        id: node.id,
        limit: pageLimit,
        offset: 0,
      });
      const children = ((data.items || []) as OrganizationTreeNode[]).map((child) => toTreeNode(child));
      rootNodes = updateNode(rootNodes, node.id, (item) => ({ ...item, children, childrenCount: children.length || item.childrenCount }));
    } catch {
      showToast(t('organization.fetchFailed'), 'error');
    } finally {
      loadingNodeIds = loadingNodeIds.filter((id) => id !== node.id);
    }
  };

  const toggleNode = async (node: TreeNode) => {
    if (!isBranch(node)) {
      if (node.kind === 'user') void openUserDetails(node);
      return;
    }
    if (isExpanded(node)) {
      expandedValue = expandedValue.filter((id) => id !== node.id);
      return;
    }
    expandedValue = [...expandedValue, node.id];
    await loadNodeChildren(node);
  };

  const loadTree = async () => {
    loading = true;
    try {
      const data = await api.getOrganizationTreeRoot({ limit: pageLimit, offset: 0 });
      const children = (data.children || []).map((item) => toTreeNode(item));
      const root = toTreeNode(data.root, children);
      rootNodes = [root];
      searchNodes = [];
      expandedValue = [root.id];
    } catch {
      showToast(t('organization.fetchFailed'), 'error');
    } finally {
      loading = false;
    }
  };

  const resetFilters = () => {
    searchText = '';
    searchNodes = [];
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

  $: visibleNodes = searchText.trim() ? searchNodes : rootNodes;

  const openUserDetails = async (item: TreeNode) => {
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

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && detailOpen) closeDetails();
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

<svelte:window on:keydown={handleDialogKeydown} />

<Toast message={toastMessage} variant={toastVariant} />

<section class="space-y-4">
  <section class="card bg-surface-50-950 border border-surface-200-800 flex flex-wrap items-center justify-between gap-2 p-3">
    <label class="relative min-w-56 flex-1 sm:max-w-md">
      <span class="sr-only">{t('organization.search')}</span>
      <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-500" aria-hidden="true" />
      <input class="input h-8 w-full pl-9 text-sm" type="search" value={searchText} on:input={onSearchInput} placeholder={t('organization.searchPlaceholder')} />
    </label>

    <div class="flex items-center gap-2">
      <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => (expandedValue = collectExpandableIds(visibleNodes))} aria-label={t('common.expand')} title={t('common.expand')}>
        <ChevronsUpDown class="size-4" aria-hidden="true" />
      </button>
      <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => (expandedValue = [])} aria-label={t('common.collapse')} title={t('common.collapse')}>
        <ChevronsDownUp class="size-4" aria-hidden="true" />
      </button>
      <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => void loadTree()} aria-label={t('common.retry')} title={t('common.retry')}>
        <RotateCcw class="size-4" aria-hidden="true" />
      </button>
      <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-8 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={resetFilters} aria-label={t('common.reset')} title={t('common.reset')}>
        <RotateCcw class="size-4" aria-hidden="true" />
      </button>
    </div>
  </section>

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    {#if loading || searchLoading}
      <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if rootNodes.length === 0}
      <div class="m-3 rounded-container border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noDepartments')}</div>
    {:else if searchText.trim() && visibleNodes.length === 0}
      <div class="m-3 rounded-container border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('organization.noSearchResults')}</div>
    {:else}
      <div class="organization-tree space-y-0.5 p-3 text-sm" role="tree" aria-label={t('organization.title')}>
        {#each visibleNodes as node (node.id)}
          {@render treeNode(node, 0)}
        {/each}
      </div>
    {/if}
  </section>
</section>

{#snippet treeNode(node: TreeNode, depth: number)}
  {#if isBranch(node)}
    <div>
      <button
        class="organization-tree-row flex min-h-7 w-full min-w-0 items-center gap-1.5 rounded px-1.5 py-0.5 text-left"
        style={`padding-left: ${depth * 16 + 6}px`}
        type="button"
        role="treeitem"
        aria-selected="false"
        aria-expanded={isExpanded(node)}
        on:click={() => void toggleNode(node)}
      >
        {#if isLoadingNode(node)}
          <span class="organization-tree-muted shrink-0 animate-spin">
            <LoaderCircle class="size-4" aria-hidden="true" />
          </span>
        {:else}
          <ChevronRight class="organization-tree-muted size-4 shrink-0 transition-transform {isExpanded(node) ? 'rotate-90' : ''}" aria-hidden="true" />
        {/if}
        {#if node.kind === 'company' || node.kind === 'organization'}
          <Building2 class="size-4 shrink-0 text-primary-600-400" aria-hidden="true" />
        {:else}
          <Network class="size-4 shrink-0 text-primary-600-400" aria-hidden="true" />
        {/if}
        <span class="truncate text-sm font-medium">{node.name || '-'}</span>
      </button>
      {#if isExpanded(node) && node.children?.length}
        <div class="organization-tree-line border-l">
          {#each node.children as childNode (childNode.id)}
            {@render treeNode(childNode, depth + 1)}
          {/each}
        </div>
      {/if}
    </div>
  {:else}
    <button
      class="organization-tree-row flex min-h-7 w-full min-w-0 items-center gap-1.5 rounded px-1.5 py-0.5 text-left"
      style={`padding-left: ${depth * 16 + 22}px`}
      type="button"
      role="treeitem"
      aria-selected="false"
      on:click={() => void openUserDetails(node)}
    >
      <UserRound class="organization-tree-muted size-4 shrink-0" aria-hidden="true" />
      <span class="min-w-0 flex-1 truncate text-sm font-medium">{userInlineLabel(node)}</span>
    </button>
  {/if}
{/snippet}

{#if detailOpen && selectedUser}
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="organization-user-dialog-title">
    <div class="mx-auto mt-10 max-w-3xl rounded-container bg-surface-50-950 border border-surface-200-800 p-5 shadow-xl">
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 id="organization-user-dialog-title" class="font-semibold">{t('directory.details')}</h2>
        <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={closeDetails} aria-label={t('common.close')} title={t('common.close')}>
          <X class="size-4" aria-hidden="true" />
        </button>
      </div>

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
              <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.id)} aria-label={copyIconLabel(selectedUser.id)} title={copyIconLabel(selectedUser.id)}>
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
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.external_user_id)} aria-label={copyIconLabel(selectedUser.external_user_id)} title={copyIconLabel(selectedUser.external_user_id)}>
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
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.email)} aria-label={copyIconLabel(selectedUser.email)} title={copyIconLabel(selectedUser.email)}>
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
                <button class="inline-grid size-4 shrink-0 place-items-center text-surface-400 hover:text-surface-950-50" type="button" on:click={() => void copyValue(selectedUser?.phone)} aria-label={copyIconLabel(selectedUser.phone)} title={copyIconLabel(selectedUser.phone)}>
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
    </div>
  </div>
{/if}

<style>
  .organization-tree {
    --org-tree-row-text: var(--color-surface-800);
    --org-tree-row-muted: var(--color-surface-500);
    --org-tree-row-hover-bg: color-mix(in srgb, var(--idb-accent) 7%, var(--idb-panel-bg));
    --org-tree-row-active-bg: color-mix(in srgb, var(--idb-accent) 11%, var(--idb-panel-bg));
    --org-tree-row-active-text: var(--color-surface-950);
    --org-tree-line: color-mix(in srgb, var(--idb-border) 78%, transparent);
  }

  :global(:root[data-mode='dark']) .organization-tree {
    --org-tree-row-text: var(--color-surface-200);
    --org-tree-row-muted: var(--color-surface-400);
    --org-tree-row-hover-bg: color-mix(in srgb, var(--idb-accent) 12%, var(--idb-panel-bg));
    --org-tree-row-active-bg: color-mix(in srgb, var(--idb-accent) 18%, var(--idb-panel-bg));
    --org-tree-row-active-text: var(--color-surface-50);
    --org-tree-line: color-mix(in srgb, var(--idb-border) 82%, transparent);
  }

  .organization-tree :global(.organization-tree-row) {
    color: var(--org-tree-row-text);
  }

  .organization-tree :global(.organization-tree-muted) {
    color: var(--org-tree-row-muted);
  }

  .organization-tree :global(.organization-tree-line) {
    border-color: var(--org-tree-line);
  }

  .organization-tree :global(.organization-tree-row:hover),
  .organization-tree :global(.organization-tree-row[aria-expanded='true']),
  .organization-tree :global(.organization-tree-row[data-selected]),
  .organization-tree :global(.organization-tree-row[data-state='open']),
  .organization-tree :global(.organization-tree-row[data-state='selected']) {
    background: var(--org-tree-row-active-bg);
    color: var(--org-tree-row-active-text);
  }

  .organization-tree :global(.organization-tree-row:hover) {
    background: var(--org-tree-row-hover-bg);
  }
</style>
