<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Dialog } from '@skeletonlabs/skeleton-svelte';
  import { X } from 'lucide-svelte';
  import { t } from '$lib/i18n';

  let {
    open = false,
    triggerLabel,
    triggerTitle = triggerLabel,
    title = triggerLabel,
    message = t('common.confirmAction'),
    confirmLabel = t('common.confirm'),
    triggerClass,
    confirmClass = 'preset-filled-error-500',
    disabled = false,
    onOpenChange,
    onConfirm,
    trigger,
  }: {
    open?: boolean;
    triggerLabel: string;
    triggerTitle?: string;
    title?: string;
    message?: string;
    confirmLabel?: string;
    triggerClass: string;
    confirmClass?: string;
    disabled?: boolean;
    onOpenChange: (open: boolean) => void;
    onConfirm: () => void;
    trigger?: Snippet;
  } = $props();
</script>

<Dialog
  open={open}
  modal
  onOpenChange={(details) => {
    onOpenChange(details.open);
  }}
>
  <Dialog.Trigger class={triggerClass} type="button" aria-label={triggerLabel} title={triggerTitle}>
    {@render trigger?.()}
  </Dialog.Trigger>
  {#if open}
    <Dialog.Backdrop class="fixed inset-0 z-50 bg-surface-950/45 backdrop-blur-sm" />
    <Dialog.Positioner class="fixed inset-0 z-50 grid place-items-center p-4">
      <Dialog.Content class="w-full max-w-sm rounded-container border border-surface-200-800 bg-surface-50-950 text-surface-950-50 shadow-xl">
        <header class="flex items-center justify-between gap-3 border-b border-surface-200-800 px-4 py-3">
          <Dialog.Title class="text-sm font-semibold">{title}</Dialog.Title>
          <Dialog.CloseTrigger
            class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0"
            type="button"
            aria-label={t('common.close')}
            title={t('common.close')}
          >
            <X class="size-4" aria-hidden="true" />
          </Dialog.CloseTrigger>
        </header>
        <div class="px-4 py-3 text-sm leading-6 text-surface-600-400">
          {message}
        </div>
        <footer class="flex justify-end gap-2 border-t border-surface-200-800 bg-surface-100-900 px-4 py-3">
          <Dialog.CloseTrigger class="btn btn-sm preset-outlined-surface-500" type="button">
            {t('common.cancel')}
          </Dialog.CloseTrigger>
          <button class={`btn btn-sm ${confirmClass}`} type="button" disabled={disabled} onclick={onConfirm}>
            {confirmLabel}
          </button>
        </footer>
      </Dialog.Content>
    </Dialog.Positioner>
  {/if}
</Dialog>
