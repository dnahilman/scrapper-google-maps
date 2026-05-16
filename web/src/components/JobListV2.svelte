<script lang="ts">
  import { onMount } from 'svelte';
  import { v1, type JobV1, fmtTimeAgo } from '../lib/api_v1.ts';
  import { notify, activeJobsCount, navigate } from '../lib/stores.ts';
  import { ws } from '../lib/ws.ts';

  let jobs: JobV1[] = $state([]);
  let loading = $state(true);
  let error = $state('');

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

  async function deleteJob(id: string): Promise<void> {
    if (!confirm('Hapus job ini beserta semua task-nya? Data places tetap tersimpan.')) return;
    try {
      await v1.deleteJob(id);
      notify('Job dihapus', 'success');
      await refresh();
    } catch (e) {
      notify(`Delete failed: ${(e as Error).message}`, 'error');
    }
  }

  function downloadJob(id: string, format: 'json' | 'csv' | 'xlsx'): void {
    window.open(v1.exportJobURL(id, format), '_blank');
  }

  function progress(j: JobV1): string {
    if (j.total_tasks === 0) return '0%';
    return `${(((j.done_count + j.failed_count) / j.total_tasks) * 100).toFixed(0)}%`;
  }

  onMount(() => {
    void refresh();
    const unsub = ws.events.subscribe((e) => {
      if (!e) return;
      if (['task.completed','task.failed','task.claimed','task.requeued','job.updated'].includes(e.type)) {
        void refresh();
      }
    });
    return () => unsub();
  });
</script>

<section class="jobs">
  <header>
    <h3>Jobs</h3>
    <span class="muted">{jobs.length} total · live (WS)</span>
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
              <button type="button" class="ghost icon-btn small" onclick={() => navigate(`#jobs/${j.id}`)}>Detail</button>
              <button type="button" class="ghost icon-btn small" onclick={() => navigate(`#jobs/${j.id}?logs`)}>Logs</button>
              <div class="dropdown">
                <button type="button" class="ghost icon-btn small">↓ Export</button>
                <div class="dropdown-menu">
                  <button type="button" onclick={() => downloadJob(j.id, 'json')}>JSON</button>
                  <button type="button" onclick={() => downloadJob(j.id, 'csv')}>CSV</button>
                  <button type="button" onclick={() => downloadJob(j.id, 'xlsx')}>Excel</button>
                </div>
              </div>
              {#if j.status === 'running' || j.status === 'pending'}
                <button type="button" class="ghost icon-btn small" onclick={() => cancel(j.id)}>Cancel</button>
              {/if}
              {#if j.failed_count > 0 && j.status !== 'running'}
                <button type="button" class="ghost icon-btn small" onclick={() => retry(j.id)}>Retry</button>
              {/if}
              <button type="button" class="ghost icon-btn small danger" onclick={() => deleteJob(j.id)}>Hapus</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</section>

<style>
  .jobs > header { display: flex; align-items: baseline; justify-content: space-between; gap: 1rem; margin-bottom: 0.5rem; }
  .jobs > header h3 { margin: 0; }
  table { width: 100%; border-collapse: collapse; font-size: 0.9em; }
  th, td { text-align: left; padding: 0.5rem 0.6rem; border-bottom: 1px solid var(--pico-muted-border-color, #2a2c33); vertical-align: top; }
  th { font-weight: 600; color: var(--pico-muted-color, #888); }
  .small { font-size: 0.85em; }
  .actions { display: flex; gap: 0.3rem; justify-content: flex-end; flex-wrap: wrap; align-items: center; }
  .badge { font-size: 0.75em; text-transform: uppercase; padding: 0.15rem 0.5rem; border-radius: 999px; background: var(--pico-secondary-background, rgba(255,255,255,.06)); }
  .status-running, .status-pending { color: #3b82f6; }
  .status-completed { color: #22c55e; }
  .status-failed, .status-cancelled { color: #ef4444; }
  .progress-bar { height: 4px; background: var(--pico-secondary-background, rgba(255,255,255,.08)); border-radius: 2px; margin-top: 4px; overflow: hidden; }
  .progress-fill { height: 100%; background: #3b82f6; transition: width .3s ease; }
  .danger { color: var(--pico-color-red-550, #c0392b); }
  .error { color: var(--pico-color-red-550, #c0392b); }

  .dropdown { position: relative; display: inline-block; }
  .dropdown-menu {
    display: none;
    position: absolute;
    right: 0;
    top: 100%;
    background: var(--pico-card-background-color, #1a1b22);
    border: 1px solid var(--pico-muted-border-color, #2a2c33);
    border-radius: 6px;
    z-index: 10;
    min-width: 80px;
    padding: 0.25rem 0;
  }
  .dropdown:hover .dropdown-menu { display: block; }
  .dropdown-menu button {
    display: block;
    width: 100%;
    text-align: left;
    padding: 0.3rem 0.75rem;
    font-size: 0.85em;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--pico-color);
  }
  .dropdown-menu button:hover { background: var(--pico-secondary-background, rgba(255,255,255,.06)); }
</style>
