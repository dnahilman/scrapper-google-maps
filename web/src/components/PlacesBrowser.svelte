<script lang="ts">
  import { v1, type Kelurahan, fmtTimeAgo } from '../lib/api_v1.ts';
  import { notify } from '../lib/stores.ts';
  import CitySelector from './CitySelector.svelte';
  import PlaceDetail from './PlaceDetail.svelte';

  interface PlaceRow {
    id: string; place_id: string; title: string; category?: string; address?: string;
    review_rating?: number; review_count?: number; keyword: string;
    kelurahan_id?: string; scraped_at: string;
  }

  let cityId = $state('');
  let kelurahanList: Kelurahan[] = $state([]);
  let kelurahanId = $state('');
  let keyword = $state('');
  let limit = $state(50);
  let offset = $state(0);
  let places: PlaceRow[] = $state([]);
  let loading = $state(false);
  let error = $state('');
  let selectedPlaceId: string | null = $state(null);
  let didInitialLoad = $state(false);

  async function loadKelurahan(id: string): Promise<void> {
    try { kelurahanList = await v1.kelurahan(id); } catch { kelurahanList = []; }
  }

  async function search(reset = true): Promise<void> {
    loading = true;
    error = '';
    if (reset) offset = 0;
    try {
      const params = new URLSearchParams();
      if (keyword) params.set('keyword', keyword);
      if (kelurahanId) params.set('kelurahan_id', kelurahanId);
      params.set('limit', String(limit));
      params.set('offset', String(offset));
      const res = await fetch(`/api/v1/places?${params.toString()}`);
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
      places = await res.json();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
      didInitialLoad = true;
    }
  }

  function copyId(placeId: string): void {
    void navigator.clipboard.writeText(placeId);
    notify('Place ID disalin', 'info', 1500);
  }
</script>

<div class="toolbar">
  <h2>Places</h2>
  <div class="muted small">{places.length} hasil · offset {offset}</div>
</div>

<section class="filter">
  <div class="filter-row">
    <div style="flex: 0 0 260px">
      <CitySelector onSelect={(c) => { cityId = c.id; kelurahanId = ''; void loadKelurahan(c.id); }} />
    </div>
    <label class="grow">
      Keyword
      <input type="text" bind:value={keyword} placeholder="cafe / barbershop / …"
        onkeydown={(e) => e.key === 'Enter' && search()} />
    </label>
    <label class="grow">
      Kelurahan
      <select bind:value={kelurahanId} disabled={!cityId || kelurahanList.length === 0}>
        <option value="">— semua —</option>
        {#each kelurahanList as k (k.id)}
          <option value={k.id}>{k.name} · {k.kecamatan_name}</option>
        {/each}
      </select>
    </label>
    <label style="flex: 0 0 90px">
      Limit
      <input type="number" bind:value={limit} min="1" max="500" />
    </label>
    <button type="button" class="primary" onclick={() => search()} disabled={loading}>
      {loading ? 'Loading…' : 'Search'}
    </button>
  </div>
</section>

{#if error}
  <p class="error">⚠ {error}</p>
{:else if !didInitialLoad}
  <p class="muted">Klik <kbd>Search</kbd> untuk memuat places. Filter optional.</p>
{:else if places.length === 0}
  <p class="muted">Tidak ada place. Coba ubah filter atau jalankan job baru.</p>
{:else}
  <div class="table-wrap">
    <table>
      <thead>
        <tr><th>Title</th><th>Category</th><th>Rating</th><th>Address</th><th>Keyword</th><th>Scraped</th><th></th></tr>
      </thead>
      <tbody>
        {#each places as p (p.id)}
          <tr>
            <td><strong>{p.title}</strong></td>
            <td class="muted small">{p.category ?? '–'}</td>
            <td>
              {#if p.review_rating}
                <span class="rating">★ {p.review_rating.toFixed(1)}</span>
                <span class="muted small">({p.review_count ?? 0})</span>
              {:else}<span class="muted">–</span>{/if}
            </td>
            <td class="muted small ellipsis" title={p.address ?? ''}>{p.address ?? '–'}</td>
            <td><span class="badge">{p.keyword}</span></td>
            <td class="muted small">{fmtTimeAgo(p.scraped_at)}</td>
            <td class="actions">
              <button type="button" class="ghost icon-btn small" onclick={() => (selectedPlaceId = p.place_id)}>Detail</button>
              <button type="button" class="ghost icon-btn small" onclick={() => copyId(p.place_id)} title="copy place_id">⎘</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  <div class="pagination">
    <button type="button" class="ghost icon-btn" onclick={() => { offset = Math.max(0, offset - limit); void search(false); }} disabled={offset === 0 || loading}>← Prev</button>
    <span class="muted small">page {Math.floor(offset / limit) + 1}</span>
    <button type="button" class="ghost icon-btn" onclick={() => { offset += limit; void search(false); }} disabled={places.length < limit || loading}>Next →</button>
  </div>
{/if}

{#if selectedPlaceId}
  <PlaceDetail placeId={selectedPlaceId} onClose={() => (selectedPlaceId = null)} />
{/if}

<style>
  .toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
  .toolbar h2 { margin: 0; }
  .filter { margin-bottom: 1rem; padding: 0.75rem 1rem; border: 1px solid var(--pico-muted-border-color, #2a2c33); border-radius: 8px; }
  .filter-row { display: flex; gap: 0.75rem; align-items: flex-end; flex-wrap: wrap; }
  .filter-row label { display: grid; gap: 0.2rem; margin: 0; }
  .grow { flex: 1; min-width: 160px; }
  .table-wrap { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; font-size: 0.9em; }
  th, td { text-align: left; padding: 0.5rem 0.6rem; border-bottom: 1px solid var(--pico-muted-border-color, #2a2c33); vertical-align: top; }
  th { font-weight: 600; color: var(--pico-muted-color, #888); }
  .small { font-size: 0.85em; }
  .rating { color: #eab308; }
  .badge { font-size: 0.7em; text-transform: uppercase; padding: 0.1rem 0.4rem; border-radius: 999px; background: var(--pico-secondary-background, rgba(255,255,255,.06)); }
  .ellipsis { max-width: 320px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .actions { display: flex; gap: 0.3rem; justify-content: flex-end; }
  .error { color: var(--pico-color-red-550, #c0392b); }
  .muted { color: var(--pico-muted-color, #888); }
  .pagination { display: flex; align-items: center; justify-content: center; gap: 1rem; margin: 1rem 0; }
</style>
