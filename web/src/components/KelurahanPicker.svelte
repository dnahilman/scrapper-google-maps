<script lang="ts">
  import { untrack } from 'svelte';
  import { v1, type Kelurahan } from '../lib/api_v1.ts';
  import { notify } from '../lib/stores.ts';

  interface Props {
    cityId: string;
    onChange?: (names: string[]) => void;
  }
  let { cityId, onChange }: Props = $props();

  let items: Kelurahan[] = $state([]);
  let selected: string[] = $state([]);
  let loading = $state(false);
  let syncing = $state(false);
  let search = $state('');
  let error = $state('');

  let filtered = $derived(
    search.trim()
      ? items.filter((k) => (k.name + ' ' + k.kecamatan_name).toLowerCase().includes(search.toLowerCase()))
      : items
  );

  $effect(() => {
    if (cityId) void load(cityId);
  });

  // Sync selected to parent without tracking onChange as a reactive dep.
  // onChange is an inline arrow function recreated each parent render — tracking
  // it would create an infinite loop (effect → onChange → parent re-render →
  // new onChange → effect re-run → ...).
  $effect(() => {
    const s = selected;
    untrack(() => onChange?.(s));
  });

  async function load(id: string): Promise<void> {
    loading = true;
    error = '';
    items = [];
    selected = []; // triggers the effect above → onChange([])
    try {
      items = await v1.kelurahan(id);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function sync(): Promise<void> {
    if (!cityId || syncing) return;
    syncing = true;
    try {
      const res = await v1.syncKelurahan(cityId);
      notify(`Synced ${res.kelurahan} kelurahan for ${res.city}`, 'success');
      await load(cityId);
    } catch (e) {
      notify(`Sync failed: ${(e as Error).message}`, 'error');
    } finally {
      syncing = false;
    }
  }

  function toggle(name: string): void {
    selected = selected.includes(name)
      ? selected.filter((n) => n !== name)
      : [...selected, name];
  }

  function selectAll(): void {
    selected = filtered.map((k) => k.name);
  }

  function selectNone(): void {
    selected = [];
  }
</script>

<div class="kel-picker">
  <header class="row">
    <strong>Kelurahan</strong>
    <span class="muted">{selected.length} dari {items.length} dipilih</span>
    <div class="spacer"></div>
    <button type="button" class="ghost icon-btn" onclick={selectAll} disabled={loading || filtered.length === 0}>Pilih semua</button>
    <button type="button" class="ghost icon-btn" onclick={selectNone} disabled={loading || selected.length === 0}>Kosongkan</button>
    <button type="button" class="ghost icon-btn" onclick={sync} disabled={syncing || !cityId}>{syncing ? 'Syncing…' : '↻ Sync'}</button>
  </header>

  <input
    type="text"
    bind:value={search}
    placeholder="Cari kelurahan atau kecamatan…"
    disabled={loading || items.length === 0}
  />

  {#if loading}
    <p class="muted">Loading kelurahan…</p>
  {:else if error}
    <p class="error">⚠ {error}</p>
  {:else if items.length === 0}
    <p class="muted">Tidak ada kelurahan. Klik <kbd>↻ Sync</kbd> untuk fetch dari emsifa.</p>
  {:else}
    <ul class="grid">
      {#each filtered as k (k.id)}
        <li>
          <label>
            <input type="checkbox" checked={selected.includes(k.name)} onchange={() => toggle(k.name)} />
            <span class="kel-name">{k.name}</span>
            <span class="muted kec">· {k.kecamatan_name}</span>
          </label>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .kel-picker { display: grid; gap: 0.5rem; }
  .row { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
  .spacer { flex: 1; }
  ul.grid { list-style: none; padding: 0.5rem; margin: 0; max-height: 280px; overflow-y: auto; border: 1px solid var(--pico-form-element-border-color, #2a2c33); border-radius: 6px; display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 0.25rem 0.75rem; }
  ul.grid li label { display: flex; align-items: baseline; gap: 0.4rem; margin: 0; cursor: pointer; font-size: 0.9em; padding: 0.15rem 0.25rem; border-radius: 3px; }
  ul.grid li label:hover { background: var(--pico-secondary-background, rgba(255,255,255,.04)); }
  .kel-name { font-weight: 500; }
  .kec { font-size: 0.85em; }
  .muted { color: var(--pico-muted-color, #888); }
  .error { color: var(--pico-color-red-550, #c0392b); margin: 0.25rem 0; font-size: 0.85em; }
</style>
