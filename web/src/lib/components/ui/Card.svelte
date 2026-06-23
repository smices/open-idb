<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    as = 'article',
    padded = true,
    class: className = '',
    children,
  }: {
    as?: 'article' | 'section' | 'div' | 'form';
    padded?: boolean;
    class?: string;
    children?: Snippet;
  } = $props();

  const classes = $derived(`idb-card ${padded ? 'p-4' : ''} ${className}`.trim());
</script>

{#if as === 'section'}
  <section class={classes}>
    {@render children?.()}
  </section>
{:else if as === 'div'}
  <div class={classes}>
    {@render children?.()}
  </div>
{:else if as === 'form'}
  <form class={classes}>
    {@render children?.()}
  </form>
{:else}
  <article class={classes}>
    {@render children?.()}
  </article>
{/if}
