<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { CircleAlert, CircleCheck } from 'lucide-svelte';

  export let message = '';
  export let variant: 'success' | 'error' = 'success';
</script>

{#if message}
  <div
    class={`fixed left-1/2 top-4 z-50 flex max-w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 items-center gap-2 rounded-container border bg-surface-50-950 px-3 py-2 text-sm text-surface-950-50 shadow-lg ${variant === 'error' ? 'border-error-200-800' : 'border-success-200-800'}`}
    role="status"
    aria-live="polite"
  >
    {#if variant === 'error'}
      <CircleAlert class="size-4 shrink-0 text-error-600-400" aria-hidden="true" />
    {:else}
      <CircleCheck class="size-4 shrink-0 text-success-600-400" aria-hidden="true" />
    {/if}
    <span class="truncate">{message}</span>
  </div>
{/if}
