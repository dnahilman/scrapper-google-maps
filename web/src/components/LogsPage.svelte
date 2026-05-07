<script lang="ts">
  import { onMount } from 'svelte';
  import { api, fmtBytes, fmtDate, type LogFile } from '../lib/api.ts';
  import LogViewer from './LogViewer.svelte';

  let files: LogFile[] = [];
  let activeFile: string | null = null;
  let url: string | null = null;
  let loading = true;

  async function refresh(): Promise<void> {
    files = await api.listLogFiles();
    loading = false;
    if (!activeFile && files.length > 0) {
      pick(files[0].name);
    }
  }

  function pick(name: string): void {
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
              <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
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
    background: rgba(34, 197, 94, 0.1);
  }
  tr:hover {
    background: rgba(34, 197, 94, 0.05);
  }
</style>
