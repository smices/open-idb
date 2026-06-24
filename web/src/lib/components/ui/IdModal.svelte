<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Dialog } from '@skeletonlabs/skeleton-svelte';
  import { X } from 'lucide-svelte';
  import { t } from '$lib/i18n';

  let {
    open = false,
    title,
    subtitle = '',
    maxWidth = 'max-w-3xl',
    onClose,
    children,
    footer,
  }: {
    open?: boolean;
    title: string;
    subtitle?: string;
    maxWidth?: string;
    onClose: () => void;
    children?: Snippet;
    footer?: Snippet;
  } = $props();
</script>

<Dialog
  open={open}
  modal
  onOpenChange={(details) => {
    if (!details.open) onClose();
  }}
>
  {#if open}
    <Dialog.Backdrop class="fixed inset-0 z-50 bg-surface-950/45 backdrop-blur-sm" />
    <Dialog.Positioner class="fixed inset-0 z-50 overflow-y-auto p-4">
      <Dialog.Content class={`mx-auto mt-10 w-full ${maxWidth} rounded-container border border-surface-200-800 bg-surface-50-950 text-surface-950-50 shadow-xl`}>
        <header class="flex items-center justify-between gap-3 border-b border-surface-200-800 px-5 py-4">
          <div class="min-w-0">
            <Dialog.Title class="text-base font-semibold">{title}</Dialog.Title>
            {#if subtitle}
              <p class="mt-1 max-w-96 truncate text-xs text-surface-500">{subtitle}</p>
            {/if}
          </div>
          <Dialog.CloseTrigger
            class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
            type="button"
            aria-label={t('common.close')}
            title={t('common.close')}
          >
            <X class="size-4" aria-hidden="true" />
          </Dialog.CloseTrigger>
        </header>

        <div class="px-5 py-4">
          {@render children?.()}
        </div>

        {#if footer}
          <footer class="flex justify-end gap-2 border-t border-surface-200-800 bg-surface-100-900 px-5 py-4">
            {@render footer()}
          </footer>
        {/if}
      </Dialog.Content>
    </Dialog.Positioner>
  {/if}
</Dialog>
