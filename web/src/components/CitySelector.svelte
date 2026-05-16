<script lang="ts">
  import { v1, type City } from '../lib/api_v1.ts';
  import { notify } from '../lib/stores.ts';

  interface Props {
    selectedCityId?: string;
    preferredSlug?: string;
    onSelect?: (city: City) => void;
  }
  let { selectedCityId = $bindable(''), preferredSlug = 'bandung', onSelect }: Props = $props();

  let cities: City[] = $state([]);
  let loading = $state(true);
  let syncing = $state(false);
  let error = $state('');
  let search = $state('');

  let filtered = $derived(
    search.trim()
      ? cities.filter((c) =>
          (c.name + ' ' + c.province_name).toLowerCase().includes(search.toLowerCase())
        )
      : cities
  );

  let showDropdown = $state(false);
  let selectedCity = $derived(cities.find((c) => c.id === selectedCityId) ?? null);

  async function load(): Promise<void> {
    loading = true;
    error = '';
    try {
      cities = await v1.cities();
      if (!selectedCityId) {
        const preferred = cities.find((c) => c.slug === preferredSlug);
        if (preferred) {
          selectedCityId = preferred.id;
          onSelect?.(preferred);
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
      notify(`Synced ${res.synced} cities`, 'success');
      await load();
    } catch (e) {
      notify(`Sync failed: ${(e as Error).message}`, 'error');
    } finally {
      syncing = false;
    }
  }

  function selectCity(city: City): void {
    selectedCityId = city.id;
    search = '';
    showDropdown = false;
    onSelect?.(city);
  }

  $effect(() => { load(); });
</script>

<div class="city-selector">
  <label>
    Kota
    <div class="combo-wrap">
      <input
        type="text"
        placeholder={loading ? 'Loading…' : (selectedCity ? selectedCity.name + ' — ' + selectedCity.province_name : 'Cari kota…')}
        bind:value={search}
        onfocus={() => (showDropdown = true)}
        onblur={() => setTimeout(() => (showDropdown = false), 150)}
        disabled={loading}
        autocomplete="off"
      />
      {#if showDropdown && filtered.length > 0}
        <ul class="dropdown">
          {#each filtered.slice(0, 20) as c (c.id)}
            <li>
              <button type="button" onmousedown={() => selectCity(c)}>
                {c.name} <span class="muted small">— {c.province_name}</span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  </label>
  <button type="button" class="ghost icon-btn sync-btn" onclick={sync} disabled={syncing || loading}>
    {syncing ? 'Syncing…' : '↻ Sync'}
  </button>
  {#if error}
    <p class="error">⚠ {error}</p>
  {/if}
</div>

<style>
  .city-selector { display: grid; grid-template-columns: 1fr auto; gap: 0.5rem; align-items: end; }
  .combo-wrap { position: relative; }
  .combo-wrap input { width: 100%; }
  .dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: var(--pico-card-background-color, #1a1b22);
    border: 1px solid var(--pico-muted-border-color, #2a2c33);
    border-radius: 6px;
    z-index: 20;
    max-height: 260px;
    overflow-y: auto;
    list-style: none;
    padding: 0.25rem 0;
    margin: 0;
  }
  .dropdown li { margin: 0; }
  .dropdown button { display: block; width: 100%; text-align: left; padding: 0.35rem 0.75rem; background: none; border: none; cursor: pointer; color: var(--pico-color); font-size: 0.9em; }
  .dropdown button:hover { background: var(--pico-secondary-background, rgba(255,255,255,.06)); }
  .sync-btn { white-space: nowrap; }
  .error { grid-column: 1 / -1; color: var(--pico-color-red-550, #c0392b); margin: 0.25rem 0 0; font-size: 0.85em; }
  .small { font-size: 0.85em; }
  .muted { color: var(--pico-muted-color, #888); }
</style>
