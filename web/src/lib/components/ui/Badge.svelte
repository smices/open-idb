<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import type { Snippet } from 'svelte';

  type Tone = 'neutral' | 'primary' | 'success' | 'warning' | 'error';

  let {
    tone = 'neutral',
    class: className = '',
    children,
  }: {
    tone?: Tone;
    class?: string;
    children?: Snippet;
  } = $props();

  const toneClass = $derived.by(() => {
    if (tone === 'primary') return 'preset-tonal-primary';
    if (tone === 'success') return 'preset-tonal-success';
    if (tone === 'warning') return 'preset-tonal-warning';
    if (tone === 'error') return 'preset-tonal-error';
    return 'preset-outlined-surface-500';
  });
</script>

<span class={`badge ${toneClass} ${className}`.trim()}>
  {@render children?.()}
</span>
