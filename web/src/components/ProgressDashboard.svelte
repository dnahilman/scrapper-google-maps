<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { v1, type HealthV1, type JobV1, type Worker, fmtTimeAgo } from '../lib/api_v1.ts';
  import { fmtUptime } from '../lib/api.ts';
  import { ws } from '../lib/ws.ts';

  let health: HealthV1 | null = null;
  let jobs: JobV1[] = [];
  let workers: Worker[] = [];
  let placesByKeyword: { keyword: string; count: number }[] = [];
  let loading = true;
  let timer: ReturnType<typeof setInterval> | null = null;
  let unsubEvents: (() => void) | null = null;

  async function refresh(): Promise<void> {
    try {
      const [h, js, ws] = await Promise.all([v1.health(), v1.jobs(), v1.workers()]);
      health = h;
      jobs = js;
      workers = ws;
      placesByKeyword = aggregatePlacesByKeyword(js);
    } catch (e) {
      console.error('dashboard refresh failed', e);
    } finally {
      loading = false;
    }
  }

  // Derive places count per keyword from job totals; cheaper than a separate
  // /api/v1/places aggregate query and stays consistent with the jobs view.
  function aggregatePlacesByKeyword(js: JobV1[]): { keyword: string; count: number }[] {
    const map = new Map<string, number>();
    for (const j of js) {
      map.set(j.keyword, (map.get(j.keyword) ?? 0) + j.done_count);
    }
    return Array.from(map.entries())
      .map(([keyword, count]) => ({ keyword, count }))
      .sort((a, b) => b.count - a.count);
  }

  onMount(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 10_000);
    unsubEvents = ws.events.subscribe(() => void refresh());
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
    if (unsubEvents) unsubEvents();
  });

  $: jobStats = aggregateJobStatus(jobs);
  function aggregateJobStatus(js: JobV1[]) {
    const out = { pending: 0, running: 0, completed: 0, failed: 0, cancelled: 0 };
    for (const j of js) {
      out[j.status] = (out[j.status] ?? 0) + 1;
    }
    return out;
  }
</script>

<div class="toolbar">
  <h2>Dashboard</h2>
  <button class="ghost icon-btn" on:click={() => void refresh()}>⟳ Refresh</button>
</div>

{#if loading}
  <p class="muted">Loading…</p>
{:else}
  <!-- KPI cards -->
  <div class="card-grid kpis">
    <article>
      <span class="label">Workers</span>
      <span class="value">
        {health?.workers_online ?? 0}
        <small class="muted">/ {health?.workers_total ?? 0}</small>
      </span>
    </article>
    <article>
      <span class="label">Jobs running</span>
      <span class="value">{jobStats.running}</span>
      <small class="muted">{jobStats.completed} done · {jobStats.failed} failed</small>
    </article>
    <article>
      <span class="label">Database</span>
      <span class="value db-{health?.db}">{health?.db ?? '–'}</span>
    </article>
    <article>
      <span class="label">Uptime</span>
      <span class="value uptime">{fmtUptime(health?.uptime_seconds)}</span>
    </article>
  </div>

  <!-- Places by keyword -->
  <section class="dash-section">
    <h3>Places per keyword</h3>
    {#if placesByKeyword.length === 0}
      <p class="muted">Belum ada place ter-scrape. Mulai job dari tab Jobs.</p>
    {:else}
      <div class="card-grid">
        {#each placesByKeyword.slice(0, 6) as row (row.keyword)}
          <article class="kw-card">
            <div class="kw-name">{row.keyword}</div>
            <div class="kw-count">{row.count}</div>
            <div class="muted small">places scraped</div>
          </article>
        {/each}
      </div>
    {/if}
  </section>

  <!-- Recent jobs -->
  <section class="dash-section">
    <h3>Recent jobs</h3>
    {#if jobs.length === 0}
      <p class="muted">Belum ada job.</p>
    {:else}
      <table>
        <thead>
          <tr>
            <th>Keyword</th>
            <th>Status</th>
            <th>Progress</th>
            <th>Created</th>
          </tr>
        </thead>
        <tbody>
          {#each jobs.slice(0, 5) as j (j.id)}
            <tr>
              <td><strong>{j.keyword}</strong></td>
              <td><span class="badge status-{j.status}">{j.status}</span></td>
              <td>
                {j.done_count}/{j.total_tasks}
                {#if j.failed_count > 0}<span class="muted small">({j.failed_count} fail)</span>{/if}
              </td>
              <td class="muted small">{fmtTimeAgo(j.created_at)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </section>

  <!-- Workers status -->
  <section class="dash-section">
    <h3>Workers status</h3>
    {#if workers.length === 0}
      <p class="muted">Belum ada worker terdaftar.</p>
    {:else}
      <ul class="worker-mini-list">
        {#each workers as w (w.id)}
          <li>
            <span class="status-dot status-{w.status}"></span>
            <span class="worker-name">{w.name}</span>
            <span class="muted small">last heartbeat {fmtTimeAgo(w.last_heartbeat)}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
{/if}

<style>
  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }
  .toolbar h2 { margin: 0; }
  .card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 0.75rem;
  }
  .kpis article {
    display: grid;
    gap: 0.2rem;
    padding: 0.85rem;
    border: 1px solid var(--pico-muted-border-color, #2a2c33);
    border-radius: 8px;
    margin: 0;
  }
  .kpis .label {
    font-size: 0.75em;
    text-transform: uppercase;
    color: var(--pico-muted-color, #888);
    letter-spacing: 0.05em;
  }
  .kpis .value {
    font-size: 1.6em;
    font-weight: 600;
  }
  .kpis .value small { font-size: 0.65em; font-weight: 400; }
  .db-ok { color: #22c55e; }
  .db-down { color: #ef4444; }
  .uptime { font-size: 1.1em; }

  .dash-section { margin-top: 1.5rem; }
  .dash-section h3 { margin: 0 0 0.5rem; }

  .kw-card {
    padding: 1rem;
    text-align: center;
    border: 1px solid var(--pico-muted-border-color, #2a2c33);
    border-radius: 8px;
    margin: 0;
  }
  .kw-name {
    font-size: 0.85em;
    color: var(--pico-muted-color, #888);
    text-transform: capitalize;
  }
  .kw-count {
    font-size: 2em;
    font-weight: 600;
    color: var(--pico-color);
  }

  table { width: 100%; border-collapse: collapse; font-size: 0.9em; }
  th, td { text-align: left; padding: 0.5rem 0.6rem; border-bottom: 1px solid var(--pico-muted-border-color, #2a2c33); }
  th { font-weight: 600; color: var(--pico-muted-color, #888); }
  .small { font-size: 0.85em; }
  .badge {
    font-size: 0.75em;
    text-transform: uppercase;
    padding: 0.15rem 0.5rem;
    border-radius: 999px;
    background: var(--pico-secondary-background, rgba(255,255,255,.06));
  }
  .status-running, .status-pending { color: #3b82f6; }
  .status-completed { color: #22c55e; }
  .status-failed, .status-cancelled { color: #ef4444; }

  .worker-mini-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.3rem;
  }
  .worker-mini-list li {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.35rem 0.5rem;
    border: 1px solid var(--pico-muted-border-color, #2a2c33);
    border-radius: 6px;
  }
  .worker-name { font-weight: 500; }
  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .status-online   { background: #22c55e; box-shadow: 0 0 6px #22c55e; }
  .status-offline  { background: #888; }
  .status-draining { background: #f59e0b; }
</style>
