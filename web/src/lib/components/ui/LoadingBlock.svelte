<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  let {
    rows = 3,
    label = 'Loading',
    class: className = '',
  }: {
    rows?: number;
    label?: string;
    class?: string;
  } = $props();
</script>

<div class={`idb-loading-block ${className}`.trim()} aria-label={label} aria-busy="true">
  {#each Array.from({ length: rows }) as _, index}
    <div class="idb-loading-row" class:short={index === rows - 1}></div>
  {/each}
</div>
