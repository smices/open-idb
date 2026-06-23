<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import type { Snippet } from 'svelte';

  type Tone = 'info' | 'success' | 'warning' | 'error';

  let {
    tone = 'info',
    role,
    class: className = '',
    children,
  }: {
    tone?: Tone;
    role?: 'alert' | 'status';
    class?: string;
    children?: Snippet;
  } = $props();

  const toneClass = $derived.by(() => {
    if (tone === 'success') return 'preset-tonal-success';
    if (tone === 'warning') return 'preset-tonal-warning';
    if (tone === 'error') return 'preset-tonal-error';
    return 'preset-tonal-primary';
  });

  const resolvedRole = $derived(role ?? (tone === 'error' ? 'alert' : 'status'));
</script>

<aside class={`alert ${toneClass} ${className}`.trim()} role={resolvedRole}>
  {@render children?.()}
</aside>
