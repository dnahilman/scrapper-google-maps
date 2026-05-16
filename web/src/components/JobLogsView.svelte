<script lang="ts">
  import { onMount } from 'svelte';
  import { v1, type TaskV1 } from '../lib/api_v1.ts';
  import { ws, type LogEntry } from '../lib/ws.ts';

  interface Props { jobId: string; }
  let { jobId }: Props = $props();

  let taskIds = $state<Set<string>>(new Set());
  let filter = $state('');
  let levelFilter = $state('all');
  let logEntries: LogEntry[] = $state([]);
  let autoScroll = $state(true);
  let viewer: HTMLDivElement | undefined = $state(undefined);

  const LEVEL_ORDER: Record<string, number> = { debug: 0, info: 1, warn: 2, error: 3, fatal: 4 };

  // Load task IDs when jobId changes
  $effect(() => {
    v1.tasks({ jobId, limit: 200 }).then((tasks: TaskV1[]) => {
      taskIds = new Set(tasks.map((t) => t.id));
    }).catch(() => {});
  });

  // Subscribe to log store once on mount — filter via $derived below
  onMount(() => {
    return ws.logEvents.subscribe((entries: LogEntry[]) => { logEntries = entries; });
  });

  // Reactively filter when logEntries OR taskIds changes
  let allLogs = $derived(logEntries.filter((e) => {
    if (!e.fields) return false;
    const tid = e.fields['task_id'];
    return typeof tid === 'string' && taskIds.has(tid);
  }));

  let filtered = $derived(allLogs.filter((e: LogEntry) => {
    const minLevel = levelFilter === 'all' ? -1 : (LEVEL_ORDER[levelFilter] ?? 0);
    if ((LEVEL_ORDER[e.level] ?? 1) < minLevel) return false;
    if (!filter) return true;
    const fields = e.fields ? Object.entries(e.fields).map(([k, v]) => `${k}=${v}`).join(' ') : '';
    return (e.message + ' ' + fields).toLowerCase().includes(filter.toLowerCase());
  }));

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

  $effect(() => {
    if (autoScroll && viewer && filtered.length > 0) {
      viewer.scrollTop = viewer.scrollHeight;
    }
  });
</script>

<div class="toolbar-bar">
  <span class="muted small">{taskIds.size} tasks · {allLogs.length} entries · {filtered.length} tampil</span>
  <div class="controls">
    <select bind:value={levelFilter} style="margin:0;font-size:.85rem">
      <option value="all">All levels</option>
      <option value="debug">Debug+</option>
      <option value="info">Info+</option>
      <option value="warn">Warn+</option>
      <option value="error">Error+</option>
    </select>
    <input type="text" placeholder="Filter…" bind:value={filter} style="margin:0;min-width:180px" />
    <label style="display:flex;align-items:center;gap:.4rem;font-size:.85rem;margin:0">
      <input type="checkbox" bind:checked={autoScroll} style="margin:0" />
      auto-scroll
    </label>
  </div>
</div>

<div class="log-viewer" bind:this={viewer}>
  {#each filtered as e, i (i)}
    <div class="line {lineClass(e.level)}">
      <span class="ts">{fmtTime(e.time)}</span>
      <span class="lvl {lineClass(e.level)}">{(e.level ?? 'info').toUpperCase()}</span>
      <span class="msg">{e.message}</span>
      {#if e.fields && Object.keys(e.fields).length > 0}
        <span class="fields muted">{Object.entries(e.fields).map(([k,v]) => `${k}=${typeof v==='object'?JSON.stringify(v):v}`).join(' ')}</span>
      {/if}
    </div>
  {/each}
  {#if filtered.length === 0}
    <div class="muted">— belum ada log untuk job ini. Pastikan task sedang berjalan. —</div>
  {/if}
</div>

<style>
  .toolbar-bar { display: flex; justify-content: space-between; align-items: center; gap: 1rem; margin-bottom: 0.5rem; flex-wrap: wrap; }
  .controls { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
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
  .small { font-size: 0.85em; }
  .muted { color: var(--pico-muted-color, #888); }
</style>
