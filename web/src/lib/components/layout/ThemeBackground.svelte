<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  let { intensity = 'standard' }: { intensity?: 'quiet' | 'standard' | 'strong' } = $props();
</script>

<div class="theme-background">
  <svg
    class="theme-background-svg {intensity}"
    viewBox="0 0 1920 1080"
    preserveAspectRatio="xMidYMid slice"
    aria-hidden="true"
  >
    <defs>
      <radialGradient id="idb-bg-grad-a" cx="30%" cy="30%" r="60%">
        <stop offset="0%" stop-color="var(--color-primary-400)" stop-opacity="0.6" />
        <stop offset="100%" stop-color="var(--color-primary-400)" stop-opacity="0" />
      </radialGradient>
      <radialGradient id="idb-bg-grad-b" cx="70%" cy="60%" r="55%">
        <stop offset="0%" stop-color="var(--color-secondary-400)" stop-opacity="0.5" />
        <stop offset="100%" stop-color="var(--color-secondary-400)" stop-opacity="0" />
      </radialGradient>
      <radialGradient id="idb-bg-grad-c" cx="50%" cy="80%" r="40%">
        <stop offset="0%" stop-color="var(--color-tertiary-400)" stop-opacity="0.4" />
        <stop offset="100%" stop-color="var(--color-tertiary-400)" stop-opacity="0" />
      </radialGradient>
    </defs>
    <rect width="1920" height="1080" fill="none" />
    <ellipse cx="500" cy="320" rx="520" ry="420" fill="url(#idb-bg-grad-a)" />
    <ellipse cx="1400" cy="600" rx="480" ry="380" fill="url(#idb-bg-grad-b)" />
    <ellipse cx="960" cy="860" rx="400" ry="300" fill="url(#idb-bg-grad-c)" />
  </svg>
</div>
