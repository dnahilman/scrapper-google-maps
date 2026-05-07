<script lang="ts">
  import JobList from './JobList.svelte';
  import LogViewer from './LogViewer.svelte';
  import { api, type JobResponse } from '../lib/api.ts';

  let viewing: JobResponse | null = null;
  let url: string | null = null;

  function onView(e: CustomEvent<JobResponse>): void {
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
