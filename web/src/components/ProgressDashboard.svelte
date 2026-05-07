<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { api, type HealthResponse, type ProgressCounts } from '../lib/api.ts';
  import { notify } from '../lib/stores.ts';
  import JobLauncher from './JobLauncher.svelte';

  interface Summary {
    counts: ProgressCounts;
    total: number;
  }

  let keywords: string[] = [];
  let summaries: Record<string, Summary> = {};
  let health: HealthResponse | null = null;
  let timer: ReturnType<typeof setInterval>;

  async function refresh(): Promise<void> {
    try {
      health = await api.health();
      keywords = health.keywords || [];
      const next: Record<string, Summary> = {};
      for (const kw of keywords) {
        try {
          const p = await api.progress(kw);
          next[kw] = { counts: p.counts, total: p.total };
        } catch {
          next[kw] = { counts: { done: 0, in_progress: 0, failed: 0 }, total: 0 };
        }
      }
      summaries = next;
    } catch (e) {
      console.error(e);
    }
  }

  onMount(() => {
    refresh();
    timer = setInterval(refresh, 8000);
  });

  onDestroy(() => clearInterval(timer));

  function pct(c: number, t: number): number {
    if (!t) return 0;
    return Math.round((c / t) * 100);
  }
</script>

<div class="toolbar">
  <h2 style="margin:0">Dashboard</h2>
  <div class="actions">
    <button class="ghost icon-btn" on:click={refresh}>⟳ Refresh</button>
  </div>
</div>

<div class="card-grid" style="margin-bottom:1.5rem">
  <article>
    <div class="kpi">
      <span class="label">Active jobs</span>
      <span class="value">{health?.active_jobs ?? '-'}</span>
    </div>
  </article>
  <article>
    <div class="kpi">
      <span class="label">Keywords</span>
      <span class="value">{keywords.length}</span>
    </div>
  </article>
  <article>
    <div class="kpi">
      <span class="label">Data dir</span>
      <span class="value">{health?.data_dir_size_mb ?? '-'} <small style="font-size:.85rem;color:var(--pico-muted-color)">MB</small></span>
    </div>
  </article>
  <article>
    <div class="kpi">
      <span class="label">Backend</span>
      <span class="value" style="font-size:1.1rem">v{health?.version ?? '-'}</span>
    </div>
  </article>
</div>

<h3>Keyword progress</h3>

{#if keywords.length === 0}
  <article>
    <p class="muted">Belum ada keyword. Jalankan scrape pertama untuk membuat folder.</p>
    <JobLauncher initialKeyword="cafe" on:started={refresh} />
  </article>
{:else}
  <div class="card-grid">
    {#each keywords as kw}
      {@const s = summaries[kw] || { counts: { done: 0, in_progress: 0, failed: 0 }, total: 0 }}
      <article>
        <header style="display:flex;justify-content:space-between;align-items:center">
          <strong style="text-transform:capitalize">{kw}</strong>
          <JobLauncher initialKeyword={kw} on:started={refresh} />
        </header>
        <div style="margin-top:.5rem">
          <div style="display:flex;justify-content:space-between;font-size:.85rem;margin-bottom:.3rem">
            <span class="muted">{s.counts.done} / {s.total} done</span>
            <strong style="color:var(--accent)">{pct(s.counts.done, s.total)}%</strong>
          </div>
          <progress value={s.counts.done} max={s.total || 1}></progress>
        </div>
        <div style="display:flex;gap:.4rem;flex-wrap:wrap;margin-top:.6rem">
          <span class="pill done"><span class="dot"></span>done {s.counts.done}</span>
          {#if s.counts.in_progress}
            <span class="pill in_progress"><span class="dot"></span>{s.counts.in_progress}</span>
          {/if}
          {#if s.counts.failed}
            <span class="pill failed"><span class="dot"></span>failed {s.counts.failed}</span>
          {/if}
        </div>
      </article>
    {/each}
  </div>
{/if}
