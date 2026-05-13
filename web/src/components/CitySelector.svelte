<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import { v1, type City } from '../lib/api_v1.ts';
  import { notify } from '../lib/stores.ts';

  export let selectedCityId: string = '';
  export let preferredSlug: string = 'bandung';

  const dispatch = createEventDispatcher<{ select: City }>();

  let cities: City[] = [];
  let loading = true;
  let syncing = false;
  let error = '';

  async function load(): Promise<void> {
    loading = true;
    error = '';
    try {
      cities = await v1.cities();
      if (!selectedCityId) {
        const preferred = cities.find((c) => c.slug === preferredSlug);
        if (preferred) {
          selectedCityId = preferred.id;
          dispatch('select', preferred);
        }
      }
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function sync(): Promise<void> {
    if (syncing) return;
    syncing = true;
    try {
      const res = await v1.syncCities();
      notify(`Synced ${res.synced} cities from emsifa`, 'success');
      await load();
    } catch (e) {
      notify(`Sync failed: ${(e as Error).message}`, 'error');
    } finally {
      syncing = false;
    }
  }

  function handleChange(ev: Event): void {
    const id = (ev.target as HTMLSelectElement).value;
    selectedCityId = id;
    const found = cities.find((c) => c.id === id);
    if (found) dispatch('select', found);
  }

  onMount(load);
</script>

<div class="city-selector">
  <label>
    Kota
    <select disabled={loading} value={selectedCityId} on:change={handleChange}>
      {#if loading}
        <option value="">Loading…</option>
      {:else if cities.length === 0}
        <option value="">No cities — click Sync</option>
      {:else}
        <option value="" disabled>Pilih kota…</option>
        {#each cities as c (c.id)}
          <option value={c.id}>{c.name} — {c.province_name}</option>
        {/each}
      {/if}
    </select>
  </label>
  <button type="button" class="ghost icon-btn sync-btn" on:click={sync} disabled={syncing || loading}>
    {syncing ? 'Syncing…' : '↻ Sync cities'}
  </button>
  {#if error}
    <p class="error">⚠ {error}</p>
  {/if}
</div>

<style>
  .city-selector {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.5rem;
    align-items: end;
  }
  .sync-btn {
    white-space: nowrap;
  }
  .error {
    grid-column: 1 / -1;
    color: var(--pico-color-red-550, #c0392b);
    margin: 0.25rem 0 0;
    font-size: 0.85em;
  }
</style>
