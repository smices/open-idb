<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  type Tone = 'neutral' | 'primary' | 'success' | 'warning' | 'error';

  let {
    label,
    value,
    hint,
    tone = 'neutral',
    href,
    class: className = '',
  }: {
    label: string;
    value: string | number;
    hint?: string;
    tone?: Tone;
    href?: string;
    class?: string;
  } = $props();

  const contentClass = 'idb-card p-4 transition hover:border-primary-500 hover:shadow-sm';
  const toneDotClass = $derived.by(() => {
    if (tone === 'primary') return 'bg-primary-500';
    if (tone === 'success') return 'bg-green-500';
    if (tone === 'warning') return 'bg-amber-500';
    if (tone === 'error') return 'bg-red-500';
    return 'bg-surface-500';
  });
</script>

{#if href}
  <a class={`${contentClass} ${className}`.trim()} href={href}>
    <div class="flex items-center justify-between gap-3">
      <p class="text-xs font-medium text-surface-600">{label}</p>
      <span class={`size-2.5 rounded-full ${toneDotClass}`} aria-hidden="true"></span>
    </div>
    <p class="mt-3 text-2xl font-semibold tabular-nums text-surface-950-50">{value}</p>
    {#if hint}
      <p class="mt-2 text-xs text-surface-500">{hint}</p>
    {/if}
  </a>
{:else}
  <article class={`${contentClass} ${className}`.trim()}>
    <div class="flex items-center justify-between gap-3">
      <p class="text-xs text-surface-600">{label}</p>
      <span class={`size-2.5 rounded-full ${toneDotClass}`} aria-hidden="true"></span>
    </div>
    <p class="mt-2 text-2xl font-semibold tabular-nums text-surface-950-50">{value}</p>
    {#if hint}
      <p class="mt-2 text-xs text-surface-500">{hint}</p>
    {/if}
  </article>
{/if}
