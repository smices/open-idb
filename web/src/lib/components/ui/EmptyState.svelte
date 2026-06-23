<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    description,
    class: className = '',
    action,
  }: {
    title: string;
    description?: string;
    class?: string;
    action?: Snippet;
  } = $props();
</script>

<div class={`idb-empty-state ${className}`.trim()}>
  <p class="font-medium text-surface-800-200">{title}</p>
  {#if description}
    <p class="mt-1 text-sm text-surface-500">{description}</p>
  {/if}
  {#if action}
    <div class="mt-4 flex justify-center">
      {@render action()}
    </div>
  {/if}
</div>
