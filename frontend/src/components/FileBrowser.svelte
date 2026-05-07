<script>
  import { onMount } from 'svelte';
  import { api, fmtBytes, fmtDate } from '../lib/api.js';
  import { notify } from '../lib/stores.js';

  let keywords = [];
  let activeKw = null;
  let files = [];
  let loading = false;
  let activeFile = null;
  let activeContent = null;
  let loadingFile = false;

  onMount(async () => {
    keywords = await api.keywords();
    if (keywords.length) selectKw(keywords[0]);
  });

  async function selectKw(kw) {
    activeKw = kw;
    activeFile = null;
    activeContent = null;
    loading = true;
    try {
      files = await api.files(kw);
    } catch (e) {
      notify(`Gagal load files: ${e.message}`, 'error');
      files = [];
    } finally {
      loading = false;
    }
  }

  async function selectFile(name) {
    activeFile = name;
    loadingFile = true;
    try {
      activeContent = await api.file(activeKw, name);
    } catch (e) {
      notify(`Gagal load file: ${e.message}`, 'error');
      activeContent = null;
    } finally {
      loadingFile = false;
    }
  }
</script>

<div class="toolbar">
  <h3 style="margin:0">Files</h3>
  <div class="actions">
    {#if keywords.length > 0}
      <select bind:value={activeKw} on:change={(e) => selectKw(e.target.value)} style="margin:0">
        {#each keywords as k}
          <option value={k}>{k}</option>
        {/each}
      </select>
    {/if}
  </div>
</div>

{#if keywords.length === 0}
  <article>
    <p class="muted">Belum ada keyword folder di <code>data/</code>. Jalankan scrape dulu.</p>
  </article>
{:else}
  <div style="display:grid;grid-template-columns: 360px 1fr; gap: 1rem; align-items:start">
    <article style="padding:0">
      {#if loading}
        <p class="muted" style="padding:1rem">Loading…</p>
      {:else if files.length === 0}
        <p class="muted" style="padding:1rem">Belum ada file JSON untuk keyword <strong>{activeKw}</strong>.</p>
      {:else}
        <div style="max-height:70vh;overflow-y:auto">
          <table style="margin:0;font-size:.85rem">
            <tbody>
              {#each files as f (f.name)}
                <tr
                  on:click={() => selectFile(f.name)}
                  style="cursor:pointer"
                  class:active={activeFile === f.name}
                >
                  <td>
                    <div style="font-weight:600">{f.name.replace('.json','')}</div>
                    <div class="muted" style="font-size:.75rem">
                      {fmtBytes(f.size_bytes)} · {fmtDate(f.modified)}
                      {#if f.shop_count != null}· {f.shop_count} shops{/if}
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </article>

    <article>
      {#if !activeFile}
        <p class="muted">Pilih file untuk preview JSON.</p>
      {:else}
        <div class="toolbar" style="margin-bottom:.75rem">
          <strong>{activeFile}</strong>
          <div class="actions">
            <a
              class="icon-btn"
              role="button"
              href={api.fileDownloadUrl(activeKw, activeFile)}
              download
            >⤓ Download</a>
          </div>
        </div>

        {#if loadingFile}
          <p class="muted">Loading…</p>
        {:else if activeContent != null}
          <pre class="json-viewer">{JSON.stringify(activeContent, null, 2)}</pre>
        {/if}
      {/if}
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
  a.icon-btn {
    text-decoration: none;
    display: inline-flex;
    align-items: center;
    gap: .35rem;
    padding: 0.35rem 0.7rem;
    border-radius: var(--pico-border-radius);
    background: var(--pico-primary);
    color: var(--pico-primary-inverse);
    font-size: .8rem;
    font-weight: 500;
  }
</style>
