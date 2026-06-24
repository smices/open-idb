<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { Pagination } from '@skeletonlabs/skeleton-svelte';
  import { t } from '$lib/i18n';

  let {
    total = 0,
    offset = 0,
    pageSize = 20,
    onPage,
  }: {
    total: number;
    offset: number;
    pageSize?: number;
    onPage: (offset: number) => void;
  } = $props();

  const page = $derived(Math.max(1, Math.floor(offset / pageSize) + 1));
  const pageCount = $derived(Math.max(1, Math.ceil(total / pageSize)));
  const start = $derived(total === 0 ? 0 : offset + 1);
  const end = $derived(Math.min(offset + pageSize, total));
</script>

{#if total > pageSize}
  <Pagination
    class="flex items-center justify-between gap-3"
    count={total}
    {pageSize}
    {page}
    siblingCount={1}
    boundaryCount={1}
    onPageChange={(details) => onPage((details.page - 1) * pageSize)}
    translations={{
      rootLabel: 'Pagination',
      prevTriggerLabel: t('common.previous'),
      nextTriggerLabel: t('common.next'),
    }}
  >
    <span class="text-xs tabular-nums text-surface-500">{start}-{end} / {total}</span>
    <div class="flex items-center gap-1">
      <Pagination.PrevTrigger class="btn btn-sm preset-outlined-surface-500" type="button">
        {t('common.previous')}
      </Pagination.PrevTrigger>
      <span class="px-2 text-xs tabular-nums text-surface-500">{page} / {pageCount}</span>
      <Pagination.NextTrigger class="btn btn-sm preset-outlined-surface-500" type="button">
        {t('common.next')}
      </Pagination.NextTrigger>
    </div>
  </Pagination>
{/if}
