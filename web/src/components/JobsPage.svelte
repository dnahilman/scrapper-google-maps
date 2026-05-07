<script>
  import JobList from './JobList.svelte';
  import LogViewer from './LogViewer.svelte';
  import { api } from '../lib/api.js';

  let viewing = null;
  let url = null;

  function onView(e) {
    viewing = e.detail;
    url = api.jobLogStreamUrl(viewing.job_id, 300);
  }
</script>

<JobList on:view={onView} />

{#if viewing}
  <article style="margin-top:1rem">
    <LogViewer
      {url}
      title={`Job ${viewing.job_id} · ${viewing.keyword}${viewing.shard ? ' · ' + viewing.shard : ''}`}
    />
  </article>
{/if}
