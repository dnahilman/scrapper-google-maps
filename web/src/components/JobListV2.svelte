<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { v1, type JobV1, fmtTimeAgo } from '../lib/api_v1.ts';
  import { notify, activeJobsCount } from '../lib/stores.ts';
  import { ws } from '../lib/ws.ts';

  let jobs: JobV1[] = [];
  let loading = true;
  let error = '';
  let timer: ReturnType<typeof setInterval> | null = null;
  let unsubEvents: (() => void) | null = null;

  async function refresh(): Promise<void> {
    try {
      jobs = await v1.jobs();
      const active = jobs.filter((j) => j.status === 'running' || j.status === 'pending').length;
      activeJobsCount.set(active);
      error = '';
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function cancel(id: string): Promise<void> {
    if (!confirm('Cancel job ini? Task running akan dibatalkan.')) return;
    try {
      await v1.cancelJob(id);
      notify('Job cancelled', 'info');
      await refresh();
    } catch (e) {
      notify(`Cancel failed: ${(e as Error).message}`, 'error');
    }
  }

  async function retry(id: string): Promise<void> {
    try {
      const res = await v1.retryFailed(id);
      notify(`Re-queued ${res.requeued} failed tasks`, 'success');
      await refresh();
    } catch (e) {
      notify(`Retry failed: ${(e as Error).message}`, 'error');
    }
  }

  function progress(j: JobV1): string {
    if (j.total_tasks === 0) return '0%';
    const pct = ((j.done_count + j.failed_count) / j.total_tasks) * 100;
    return `${pct.toFixed(0)}%`;
  }

  onMount(() => {
    void refresh();
    // Cheap fallback polling — WS push will refresh on transitions but we
    // still want periodic sync in case a message was dropped.
    timer = setInterval(() => void refresh(), 8_000);

    const wsHandler = (ev: CustomEvent | null): void => {
      if (!ev) return;
      const ce = ev as unknown as { type: string };
      if (['task.completed', 'task.failed', 'task.claimed', 'task.requeued'].includes(ce.type)) {
        void refresh();
      }
    };
    const sub = ws.events.subscribe((e) => wsHandler(e as unknown as CustomEvent));
    unsubEvents = sub;
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
    if (unsubEvents) unsubEvents();
  });
</script>

<section class="jobs">
  <header>
    <h3>Jobs</h3>
    <span class="muted">{jobs.length} total · live (WS + 8s poll)</span>
  </header>

  {#if loading}
    <p class="muted">Loading…</p>
  {:else if error}
    <p class="error">⚠ {error}</p>
  {:else if jobs.length === 0}
    <p class="muted">Belum ada job. Klik <kbd>▶ Start scrape job</kbd>.</p>
  {:else}
    <table>
      <thead>
        <tr>
          <th>Keyword</th>
          <th>Status</th>
          <th>Progress</th>
          <th>Created</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each jobs as j (j.id)}
          <tr>
            <td>
              <strong>{j.keyword}</strong>
              <div class="muted small">{j.id.slice(0, 8)}…</div>
            </td>
            <td><span class="badge status-{j.status}">{j.status}</span></td>
            <td>
              {j.done_count}/{j.total_tasks}
              {#if j.failed_count > 0}<span class="muted small">({j.failed_count} failed)</span>{/if}
              <div class="progress-bar"><div class="progress-fill" style="width: {progress(j)}"></div></div>
            </td>
            <td><span class="muted small">{fmtTimeAgo(j.created_at)}</span></td>
            <td class="actions">
              {#if j.status === 'running' || j.status === 'pending'}
                <button type="button" class="ghost icon-btn small" on:click={() => cancel(j.id)}>Cancel</button>
              {/if}
              {#if j.failed_count > 0 && j.status !== 'running'}
                <button type="button" class="ghost icon-btn small" on:click={() => retry(j.id)}>Retry failed</button>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</section>

<style>
  .jobs > header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 0.5rem;
  }
  .jobs > header h3 { margin: 0; }
  table { width: 100%; border-collapse: collapse; font-size: 0.9em; }
  th, td { text-align: left; padding: 0.5rem 0.6rem; border-bottom: 1px solid var(--pico-muted-border-color, #2a2c33); vertical-align: top; }
  th { font-weight: 600; color: var(--pico-muted-color, #888); }
  .small { font-size: 0.85em; }
  .actions { display: flex; gap: 0.4rem; justify-content: flex-end; }
  .badge {
    font-size: 0.75em;
    text-transform: uppercase;
    padding: 0.15rem 0.5rem;
    border-radius: 999px;
    background: var(--pico-secondary-background, rgba(255, 255, 255, 0.06));
  }
  .status-running, .status-pending { color: #3b82f6; }
  .status-completed { color: #22c55e; }
  .status-failed, .status-cancelled { color: #ef4444; }
  .progress-bar { height: 4px; background: var(--pico-secondary-background, rgba(255,255,255,.08)); border-radius: 2px; margin-top: 4px; overflow: hidden; }
  .progress-fill { height: 100%; background: #3b82f6; transition: width .3s ease; }
  .error { color: var(--pico-color-red-550, #c0392b); }
</style>
