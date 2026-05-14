<script lang="ts">
  import JobLauncherV2 from './JobLauncherV2.svelte';
  import JobListV2 from './JobListV2.svelte';
  import type { JobV1 } from '../lib/api_v1.ts';

  let listKey = 0;

  function onStarted(_e: CustomEvent<JobV1>): void {
    // Force JobList to refresh by remounting via key change.
    listKey++;
  }
</script>

<header class="header">
  <h2>Jobs</h2>
  <JobLauncherV2 on:started={onStarted} />
</header>

{#key listKey}
  <JobListV2 />
{/key}

<style>
  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
  }
  .header h2 { margin: 0; }
</style>
