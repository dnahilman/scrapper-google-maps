<script>
  import { onDestroy, afterUpdate } from 'svelte';
  import { api, streamLogs } from '../lib/api.js';

  export let url = null;        // SSE URL (ditentukan oleh parent)
  export let title = 'Log';

  let lines = [];
  let es = null;
  let viewer;
  let autoScroll = true;
  let filter = '';

  function classify(l) {
    if (/error|exception|traceback|failed|captcha/i.test(l)) return 'error';
    if (/warn/i.test(l)) return 'warn';
    if (/debug/i.test(l)) return 'debug';
    return 'info';
  }

  function connect() {
    if (es) {
      es.close();
      es = null;
    }
    lines = [];
    if (!url) return;
    es = streamLogs(
      url,
      (line) => {
        lines = [...lines.slice(-1500), line];
      },
      (err) => {
        console.warn('SSE error', err);
      }
    );
  }

  $: if (url) connect();

  onDestroy(() => {
    if (es) es.close();
  });

  afterUpdate(() => {
    if (autoScroll && viewer) {
      viewer.scrollTop = viewer.scrollHeight;
    }
  });

  $: filtered = filter
    ? lines.filter((l) => l.toLowerCase().includes(filter.toLowerCase()))
    : lines;
</script>

<div class="toolbar">
  <h3 style="margin:0">{title}</h3>
  <div class="actions">
    <input
      type="text"
      placeholder="Filter (substring)…"
      bind:value={filter}
      style="margin:0;min-width:200px"
    />
    <label style="display:flex;align-items:center;gap:.4rem;font-size:.85rem;margin:0">
      <input type="checkbox" bind:checked={autoScroll} style="margin:0" />
      auto-scroll
    </label>
    <button class="ghost icon-btn" on:click={() => (lines = [])}>Clear</button>
  </div>
</div>

{#if !url}
  <p class="muted">Pilih log file atau job untuk live tail.</p>
{:else}
  <div class="log-viewer" bind:this={viewer}>
    {#each filtered as line, i (i + ':' + line)}
      <div class="line {classify(line)}">{line || ' '}</div>
    {/each}
    {#if filtered.length === 0}
      <div class="muted">— belum ada output —</div>
    {/if}
  </div>
{/if}
