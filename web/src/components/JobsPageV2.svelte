<script lang="ts">
  import { route, navigate } from '../lib/stores.ts';
  import JobListV2 from './JobListV2.svelte';
  import JobLauncherV2 from './JobLauncherV2.svelte';
  import JobDetailV2 from './JobDetailV2.svelte';
  import JobLogsView from './JobLogsView.svelte';

  let view = $derived(
    $route.sub
      ? ('params' in $route && 'logs' in $route.params ? 'job-logs' : 'job-detail')
      : ('params' in $route && 'create' in $route.params ? 'create' : 'list')
  );

  let jobId = $derived($route.sub ?? '');
</script>

{#if view === 'list'}
  <div class="header">
    <h2>Jobs</h2>
    <button class="icon-btn primary" onclick={() => navigate('#jobs?create')}>▶ Start scrape job</button>
  </div>
  <JobListV2 />

{:else if view === 'create'}
  <div class="header">
    <h2>Start scrape job</h2>
    <button class="ghost icon-btn" onclick={() => navigate('#jobs')}>← Kembali</button>
  </div>
  <JobLauncherV2 onStarted={() => navigate('#jobs')} />

{:else if view === 'job-detail'}
  <div class="header">
    <h2>Job Detail</h2>
    <div class="header-actions">
      <button class="ghost icon-btn" onclick={() => navigate(`#jobs/${jobId}?logs`)}>≡ Logs</button>
      <button class="ghost icon-btn" onclick={() => navigate('#jobs')}>← Kembali</button>
    </div>
  </div>
  <JobDetailV2 {jobId} />

{:else if view === 'job-logs'}
  <div class="header">
    <h2>Job Logs</h2>
    <div class="header-actions">
      <button class="ghost icon-btn" onclick={() => navigate(`#jobs/${jobId}`)}>◆ Detail</button>
      <button class="ghost icon-btn" onclick={() => navigate('#jobs')}>← Kembali</button>
    </div>
  </div>
  <JobLogsView {jobId} />
{/if}

<style>
  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
  }
  .header h2 { margin: 0; }
  .header-actions { display: flex; gap: 0.5rem; }
</style>
