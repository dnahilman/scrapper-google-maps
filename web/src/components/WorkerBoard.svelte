<script lang="ts">
  import { onMount } from 'svelte';
  import { v1, type Worker, fmtTimeAgo } from '../lib/api_v1.ts';
  import { notify } from '../lib/stores.ts';
  import { ws } from '../lib/ws.ts';

  let workers: Worker[] = $state([]);
  let loading = $state(true);
  let error = $state('');

  async function refresh(): Promise<void> {
    try {
      workers = await v1.workers();
      error = '';
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function drain(id: string): Promise<void> {
    try {
      await v1.drainWorker(id);
      notify('Worker draining', 'info');
      await refresh();
    } catch (e) {
      notify(`Drain failed: ${(e as Error).message}`, 'error');
    }
  }

  async function remove(id: string): Promise<void> {
    if (!confirm('Hapus worker dari registry?')) return;
    try {
      await v1.deleteWorker(id);
      notify('Worker removed', 'info');
      await refresh();
    } catch (e) {
      notify(`Delete failed: ${(e as Error).message}`, 'error');
    }
  }

  onMount(() => {
    void refresh();
    const unsub = ws.events.subscribe((e) => {
      if (!e) return;
      if (['worker.registered', 'worker.offline'].includes(e.type)) {
        void refresh();
      }
    });
    return () => unsub();
  });
</script>

<section class="worker-board">
  <header>
    <h3>Workers</h3>
    <span class="muted">{workers.length} terdaftar · live (WS)</span>
  </header>

  {#if loading}
    <p class="muted">Loading…</p>
  {:else if error}
    <p class="error">⚠ {error}</p>
  {:else if workers.length === 0}
    <p class="muted">Belum ada worker. Jalankan binary worker dengan <code>MASTER_URL</code> menunjuk ke master ini.</p>
  {:else}
    <div class="grid">
      {#each workers as w (w.id)}
        <article class="card status-{w.status}">
          <header>
            <strong>{w.name}</strong>
            <span class="badge">{w.status}</span>
          </header>
          <dl>
            <dt>Host</dt><dd>{w.hostname || '–'}</dd>
            <dt>IP</dt><dd><code>{w.ip_addr || '–'}</code></dd>
            <dt>Concurrency</dt><dd>{w.max_concurrency}</dd>
            <dt>Heartbeat</dt><dd>{fmtTimeAgo(w.last_heartbeat)}</dd>
            <dt>Registered</dt><dd>{fmtTimeAgo(w.registered_at)}</dd>
          </dl>
          <footer>
            {#if w.status === 'online'}
              <button type="button" class="ghost icon-btn" onclick={() => drain(w.id)}>Drain</button>
            {/if}
            <button type="button" class="ghost icon-btn danger" onclick={() => remove(w.id)}>Remove</button>
          </footer>
        </article>
      {/each}
    </div>
  {/if}
</section>

<style>
  .worker-board { display: grid; gap: 0.75rem; }
  .worker-board > header { display: flex; align-items: baseline; justify-content: space-between; gap: 1rem; }
  .worker-board > header h3 { margin: 0; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 0.75rem; }
  .card { border: 1px solid var(--pico-form-element-border-color, #2a2c33); border-radius: 8px; padding: 0.85rem; margin: 0; display: grid; gap: 0.5rem; }
  .card.status-online  { border-left: 3px solid #2ecc71; }
  .card.status-offline { border-left: 3px solid #95a5a6; opacity: 0.7; }
  .card.status-draining { border-left: 3px solid #f39c12; }
  .card header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
  .badge { font-size: 0.75em; text-transform: uppercase; padding: 0.15rem 0.5rem; border-radius: 999px; background: var(--pico-secondary-background, rgba(255,255,255,.06)); }
  dl { margin: 0; display: grid; grid-template-columns: max-content 1fr; gap: 0.15rem 0.6rem; font-size: 0.85em; }
  dt { color: var(--pico-muted-color, #888); }
  dd { margin: 0; }
  footer { display: flex; gap: 0.5rem; justify-content: flex-end; }
  .danger { color: var(--pico-color-red-550, #c0392b); }
  .error { color: var(--pico-color-red-550, #c0392b); }
</style>
