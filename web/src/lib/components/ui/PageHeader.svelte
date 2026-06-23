<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    description,
    actions,
    class: className = '',
  }: {
    title: string;
    description?: string;
    actions?: Snippet;
    class?: string;
  } = $props();
</script>

<header class={`idb-page-header ${className}`.trim()}>
  <div>
    <h1>{title}</h1>
    {#if description}
      <p>{description}</p>
    {/if}
  </div>
  {#if actions}
    <div class="idb-page-header-actions">
      {@render actions()}
    </div>
  {/if}
</header>
