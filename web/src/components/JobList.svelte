<script lang="ts">
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import { api, fmtDate, type JobResponse } from '../lib/api.ts';
  import { activeJobsCount, notify } from '../lib/stores.ts';
  import JobLauncher from './JobLauncher.svelte';

  export let keywordFilter: string | null = null;

  let jobs: JobResponse[] = [];
  let loading = true;
  let timer: ReturnType<typeof setInterval>;
  const dispatch = createEventDispatcher<{ view: JobResponse }>();

  async function refresh(): Promise<void> {
    try {
      jobs = await api.listJobs(keywordFilter);
      activeJobsCount.set(jobs.filter((j) => j.status === 'running').length);
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  }

  async function stopJob(j: JobResponse): Promise<void> {
    if (!confirm(`Stop job ${j.job_id} (${j.keyword} ${j.shard ?? ''})?`)) return;
    try {
      await api.stopJob(j.job_id);
      notify(`Job ${j.job_id} stopped`, 'success');
      await refresh();
    } catch (e) {
      notify(`Gagal stop: ${(e as Error).message}`, 'error');
    }
  }

  function uptime(started: string): string {
    const ms = Date.now() - new Date(started).getTime();
    const s = Math.floor(ms / 1000);
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    if (h) return `${h}h ${m}m`;
    if (m) return `${m}m`;
    return `${s}s`;
  }

  onMount(() => {
    refresh();
    timer = setInterval(refresh, 4000);
  });

  onDestroy(() => clearInterval(timer));
</script>

<div class="toolbar">
  <h3 style="margin:0">Jobs</h3>
  <div class="actions">
    <button class="ghost icon-btn" on:click={refresh}>⟳ Refresh</button>
    <JobLauncher initialKeyword={keywordFilter || 'cafe'} on:started={refresh} />
  </div>
</div>

<article style="padding:0">
  {#if loading}
    <p class="muted" style="padding:1rem">Loading…</p>
  {:else if jobs.length === 0}
    <p class="muted" style="padding:1.5rem;text-align:center">
      Belum ada job. Klik <strong>Start scrape</strong> untuk launching pertama.
    </p>
  {:else}
    <div style="overflow-x:auto">
      <table style="margin:0">
        <thead>
          <tr>
            <th>Job</th>
            <th>Keyword</th>
            <th>Shard</th>
            <th>Filter</th>
            <th>PID</th>
            <th>Started</th>
            <th>Uptime</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each jobs as j (j.job_id)}
            <tr>
              <td><code>{j.job_id}</code></td>
              <td>{j.keyword}</td>
              <td>{j.shard ?? '-'}</td>
              <td>{j.kelurahan ?? '-'}{j.limit ? ` · limit ${j.limit}` : ''}</td>
              <td>{j.pid}</td>
              <td>{fmtDate(j.started_at)}</td>
              <td>{uptime(j.started_at)}</td>
              <td><span class="pill {j.status}"><span class="dot"></span>{j.status}</span></td>
              <td style="text-align:right">
                <button
                  class="ghost icon-btn"
                  on:click={() => dispatch('view', j)}
                  title="View logs"
                >
                  ≡ Logs
                </button>
                {#if j.status === 'running'}
                  <button class="danger icon-btn" on:click={() => stopJob(j)}>■ Stop</button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</article>
