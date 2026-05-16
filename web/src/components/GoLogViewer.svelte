<script lang="ts">
  import { onMount } from 'svelte';
  import { ws, type LogEntry } from '../lib/ws.ts';

  let filter = $state('');
  let levelFilter = $state('all');
  let autoScroll = $state(true);
  let viewer: HTMLDivElement | undefined = $state(undefined);
  let logEntries: LogEntry[] = $state([]);

  const LEVEL_ORDER: Record<string, number> = { debug: 0, info: 1, warn: 2, error: 3, fatal: 4 };

  onMount(() => {
    return ws.logEvents.subscribe((entries) => { logEntries = entries; });
  });

  let showHttp = $state(false);

  let filtered = $derived(
    logEntries.filter((e: LogEntry) => {
      if (!showHttp && e.message === 'http') return false;
      const minLevel = levelFilter === 'all' ? -1 : (LEVEL_ORDER[levelFilter] ?? 0);
      if ((LEVEL_ORDER[e.level] ?? 1) < minLevel) return false;
      if (!filter) return true;
      const fields = e.fields ? Object.entries(e.fields).map(([k, v]) => `${k}=${v}`).join(' ') : '';
      return (e.message + ' ' + fields).toLowerCase().includes(filter.toLowerCase());
    })
  );

  function lineClass(level: string): string {
    if (level === 'error' || level === 'fatal') return 'error';
    if (level === 'warn') return 'warn';
    if (level === 'debug') return 'debug';
    return 'info';
  }

  function fmtTime(t: string): string {
    if (!t) return '';
    try { return new Date(t).toLocaleTimeString(); } catch { return t; }
  }

  function fmtFields(fields?: Record<string, unknown>): string {
    if (!fields || Object.keys(fields).length === 0) return '';
    return Object.entries(fields).map(([k, v]) => `${k}=${typeof v === 'object' ? JSON.stringify(v) : v}`).join(' ');
  }

  $effect(() => {
    if (autoScroll && viewer && filtered.length > 0) {
      viewer.scrollTop = viewer.scrollHeight;
    }
  });
</script>

<div class="toolbar">
  <h3 style="margin:0">Go Live Logs <span class="badge muted">{logEntries.length}</span></h3>
  <div class="actions">
    <select bind:value={levelFilter} style="margin:0;font-size:.85rem">
      <option value="all">All levels</option>
      <option value="debug">Debug+</option>
      <option value="info">Info+</option>
      <option value="warn">Warn+</option>
      <option value="error">Error+</option>
    </select>
    <input type="text" placeholder="Filter…" bind:value={filter} style="margin:0;min-width:180px" />
    <label style="display:flex;align-items:center;gap:.4rem;font-size:.85rem;margin:0">
      <input type="checkbox" bind:checked={showHttp} style="margin:0" />
      HTTP logs
    </label>
    <label style="display:flex;align-items:center;gap:.4rem;font-size:.85rem;margin:0">
      <input type="checkbox" bind:checked={autoScroll} style="margin:0" />
      auto-scroll
    </label>
    <button class="ghost icon-btn" onclick={() => ws.clearLogs()}>Clear</button>
  </div>
</div>

<div class="log-viewer" bind:this={viewer}>
  {#each filtered as e, i (i)}
    <div class="line {lineClass(e.level)}">
      <span class="ts">{fmtTime(e.time)}</span>
      <span class="lvl {lineClass(e.level)}">{(e.level ?? 'info').toUpperCase()}</span>
      <span class="msg">{e.message}</span>
      {#if e.fields && Object.keys(e.fields).length > 0}
        <span class="fields muted">{fmtFields(e.fields)}</span>
      {/if}
    </div>
  {/each}
  {#if filtered.length === 0}
    <div class="muted">— belum ada log masuk —</div>
  {/if}
</div>

<style>
  .log-viewer {
    font-family: 'Courier New', Courier, monospace;
    font-size: 0.8rem;
    background: var(--pico-code-background-color, #0d1117);
    border-radius: 8px;
    padding: 0.75rem;
    max-height: 65vh;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .line { display: flex; gap: 0.6rem; align-items: baseline; padding: 1px 0; line-height: 1.4; white-space: pre-wrap; word-break: break-all; }
  .ts   { color: #555; flex-shrink: 0; font-size: 0.72rem; min-width: 8ch; }
  .lvl  { flex-shrink: 0; font-weight: 700; min-width: 5ch; font-size: 0.75rem; }
  .lvl.info  { color: #22c55e; }
  .lvl.debug { color: #64748b; }
  .lvl.warn  { color: #eab308; }
  .lvl.error { color: #ef4444; }
  .msg    { color: #e2e8f0; flex: 1; }
  .fields { font-size: 0.72rem; color: #888; }
  .line.error { background: rgba(239,68,68,.06); }
  .line.warn  { background: rgba(234,179,8,.06); }
  .line.debug { opacity: 0.65; }
  .badge { font-size: 0.7rem; background: rgba(255,255,255,.08); padding: 1px 6px; border-radius: 999px; }
  .muted { color: var(--pico-muted-color, #888); }
</style>
