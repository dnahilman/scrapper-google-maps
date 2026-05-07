<script>
  import { onMount } from 'svelte';
  import { api, fmtBytes, fmtDate } from '../lib/api.js';
  import LogViewer from './LogViewer.svelte';

  let files = [];
  let activeFile = null;
  let url = null;
  let loading = true;

  async function refresh() {
    files = await api.listLogFiles();
    loading = false;
    if (!activeFile && files.length > 0) {
      pick(files[0].name);
    }
  }

  function pick(name) {
    activeFile = name;
    url = api.fileLogStreamUrl(name, 300);
  }

  onMount(refresh);
</script>

<div class="toolbar">
  <h2 style="margin:0">Logs</h2>
  <div class="actions">
    <button class="ghost icon-btn" on:click={refresh}>⟳ Refresh files</button>
  </div>
</div>

{#if loading}
  <p class="muted">Loading…</p>
{:else if files.length === 0}
  <article><p class="muted">Belum ada logfile di <code>logs/</code>.</p></article>
{:else}
  <div style="display:grid;grid-template-columns:300px 1fr;gap:1rem;align-items:start">
    <article style="padding:0">
      <div style="max-height:70vh;overflow-y:auto">
        <table style="margin:0;font-size:.85rem">
          <tbody>
            {#each files as f (f.name)}
              <tr
                on:click={() => pick(f.name)}
                style="cursor:pointer"
                class:active={activeFile === f.name}
              >
                <td>
                  <div style="font-weight:600">{f.name}</div>
                  <div class="muted" style="font-size:.72rem">
                    {f.kind} · {fmtBytes(f.size_bytes)} · {fmtDate(f.modified)}
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </article>
    <article>
      <LogViewer {url} title={activeFile || 'Log viewer'} />
    </article>
  </div>
{/if}

<style>
  tr.active {
    background: rgba(59, 130, 246, 0.12);
  }
  tr:hover {
    background: rgba(59, 130, 246, 0.06);
  }
</style>
