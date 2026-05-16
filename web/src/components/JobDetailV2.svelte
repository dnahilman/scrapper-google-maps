<script lang="ts">
  import { v1, type Place, type PlacesPage, type Review, fmtTimeAgo } from '../lib/api_v1.ts';
  import FieldRenderer from './FieldRenderer.svelte';

  interface Props { jobId: string; }
  let { jobId }: Props = $props();

  let data: PlacesPage | null = $state(null);
  let loading = $state(true);
  let error = $state('');
  let page = $state(1);
  let perPage = $state(20);
  let expanded = $state<Set<string>>(new Set());
  let reviewsMap = $state<Record<string, Review[] | 'loading' | 'error'>>({});

  async function load(): Promise<void> {
    loading = true;
    error = '';
    try {
      data = await v1.jobPlaces(jobId, page, perPage);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    page; // track
    void load();
  });

  async function toggleExpand(id: string, placeId: string): Promise<void> {
    const next = new Set(expanded.has(id)
      ? [...expanded].filter((x) => x !== id)
      : [...expanded, id]);
    expanded = next;

    if (next.has(id) && reviewsMap[id] === undefined) {
      reviewsMap = { ...reviewsMap, [id]: 'loading' };
      try {
        const reviews = await v1.placeReviews(placeId);
        reviewsMap = { ...reviewsMap, [id]: reviews };
      } catch {
        reviewsMap = { ...reviewsMap, [id]: 'error' };
      }
    }
  }

  const scalarFields: Array<keyof Place> = [
    'title', 'category', 'address', 'phone', 'website',
    'price', 'status', 'review_rating', 'review_count',
    'latitude', 'longitude', 'description',
    'plus_code', 'timezone', 'thumbnail', 'reviews_link',
    'place_id', 'cid', 'keyword',
  ];

  const complexFields: Array<keyof Place> = [
    'categories', 'emails', 'reviews_per_rating',
    'complete_address', 'open_hours', 'popular_times',
    'images', 'menu', 'about', 'owner', 'reservations', 'order_online',
  ];
</script>

{#if loading}
  <p class="muted">Loading places…</p>
{:else if error}
  <p class="error">⚠ {error}</p>
{:else if data}
  <div class="toolbar">
    <span class="muted">{data.total} places total · halaman {data.page}</span>
    <div class="actions">
      <button class="ghost icon-btn" onclick={() => { page = Math.max(1, page - 1); }} disabled={page <= 1}>← Prev</button>
      <span class="muted small">hal {page} / {Math.ceil(data.total / perPage)}</span>
      <button class="ghost icon-btn" onclick={() => { page++; }} disabled={data.items.length < perPage}>Next →</button>
    </div>
  </div>

  {#if data.items.length === 0}
    <p class="muted">Belum ada place ter-scrape untuk job ini.</p>
  {:else}
    <div class="place-list">
      {#each data.items as place (place.id)}
        <div class="place-card">
          <div class="place-header" role="button" tabindex="0"
            onclick={() => toggleExpand(place.id, place.place_id)}
            onkeydown={(e) => e.key === 'Enter' && toggleExpand(place.id, place.place_id)}
          >
            <div class="place-title">
              <strong>{place.title}</strong>
              {#if place.category}<span class="muted small"> · {place.category}</span>{/if}
            </div>
            <div class="place-meta">
              {#if place.review_rating}
                <span class="rating">★ {place.review_rating.toFixed(1)}</span>
                <span class="muted small">({place.review_count ?? 0} ulasan)</span>
              {/if}
              <span class="muted small">{fmtTimeAgo(place.scraped_at)}</span>
              <span class="expand-toggle">{expanded.has(place.id) ? '▲' : '▼'}</span>
            </div>
          </div>

          {#if expanded.has(place.id)}
            <div class="place-fields">
              <!-- Scalar + complex fields -->
              <table class="fields-table">
                <tbody>
                  {#each scalarFields as f}
                    {#if place[f] != null && place[f] !== ''}
                      <tr>
                        <th>{f.replace(/_/g, ' ')}</th>
                        <td><FieldRenderer value={place[f]} /></td>
                      </tr>
                    {/if}
                  {/each}
                  {#each complexFields as f}
                    {#if place[f] != null}
                      <tr>
                        <th>{f.replace(/_/g, ' ')}</th>
                        <td><FieldRenderer value={place[f]} /></td>
                      </tr>
                    {/if}
                  {/each}
                </tbody>
              </table>

              <!-- Reviews section -->
              <div class="reviews-section">
                <h4 class="reviews-heading">Reviews</h4>
                {#if reviewsMap[place.id] === 'loading'}
                  <p class="muted small">Loading reviews…</p>
                {:else if reviewsMap[place.id] === 'error'}
                  <p class="error small">Gagal memuat reviews.</p>
                {:else if Array.isArray(reviewsMap[place.id])}
                  {@const reviews = reviewsMap[place.id] as Review[]}
                  {#if reviews.length === 0}
                    <p class="muted small">Tidak ada review.</p>
                  {:else}
                    <div class="review-list">
                      {#each reviews as r (r.id)}
                        <div class="review-card">
                          <div class="review-header">
                            <span class="review-name">{r.name || '(anonim)'}</span>
                            {#if r.rating != null}
                              <span class="rating small">{'★'.repeat(r.rating)}{'☆'.repeat(5 - r.rating)}</span>
                            {/if}
                            {#if r.when}
                              <span class="muted small">· {r.when}</span>
                            {/if}
                            {#if r.age_days != null}
                              <span class="muted small">({r.age_days}d ago)</span>
                            {/if}
                          </div>
                          {#if r.description}
                            <p class="review-text">{r.description}</p>
                          {/if}
                          {#if r.owner_response}
                            <div class="owner-response">
                              <span class="muted small">Respons pemilik: </span>
                              <FieldRenderer value={r.owner_response} />
                            </div>
                          {/if}
                          {#if r.images && r.images.length > 0}
                            <div class="review-images">
                              {#each r.images as img}
                                <a href={img} target="_blank" rel="noopener noreferrer" class="img-link">📷</a>
                              {/each}
                            </div>
                          {/if}
                        </div>
                      {/each}
                    </div>
                  {/if}
                {/if}
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>

    <div class="pagination">
      <button class="ghost icon-btn" onclick={() => { page = Math.max(1, page - 1); }} disabled={page <= 1}>← Prev</button>
      <span class="muted small">halaman {page} / {Math.ceil(data.total / perPage)}</span>
      <button class="ghost icon-btn" onclick={() => { page++; }} disabled={data.items.length < perPage}>Next →</button>
    </div>
  {/if}
{/if}

<style>
  .toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; }
  .actions { display: flex; align-items: center; gap: 0.5rem; }
  .place-list { display: grid; gap: 0.5rem; }
  .place-card { border: 1px solid var(--pico-muted-border-color, #2a2c33); border-radius: 8px; overflow: hidden; }
  .place-header { display: flex; justify-content: space-between; align-items: center; padding: 0.65rem 0.85rem; cursor: pointer; gap: 1rem; }
  .place-header:hover { background: var(--pico-secondary-background, rgba(255,255,255,.03)); }
  .place-title { flex: 1; min-width: 0; }
  .place-meta { display: flex; align-items: center; gap: 0.5rem; flex-shrink: 0; font-size: 0.85em; }
  .rating { color: #eab308; }
  .expand-toggle { color: var(--pico-muted-color, #888); font-size: 0.75em; }
  .place-fields { padding: 0.5rem 0.85rem 0.85rem; border-top: 1px solid var(--pico-muted-border-color, #2a2c33); }
  .fields-table { width: 100%; border-collapse: collapse; font-size: 0.85em; }
  .fields-table th { text-align: left; padding: 0.25rem 0.6rem 0.25rem 0; color: var(--pico-muted-color, #888); font-weight: 500; width: 160px; white-space: nowrap; text-transform: capitalize; vertical-align: top; }
  .fields-table td { padding: 0.25rem 0; vertical-align: top; word-break: break-word; }

  .reviews-section { margin-top: 1rem; border-top: 1px solid var(--pico-muted-border-color, #2a2c33); padding-top: 0.75rem; }
  .reviews-heading { margin: 0 0 0.5rem; font-size: 0.9em; color: var(--pico-muted-color, #888); font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em; }
  .review-list { display: grid; gap: 0.5rem; }
  .review-card { background: var(--pico-code-background-color, #0d1117); border-radius: 6px; padding: 0.6rem 0.75rem; font-size: 0.85em; }
  .review-header { display: flex; align-items: center; gap: 0.4rem; flex-wrap: wrap; margin-bottom: 0.25rem; }
  .review-name { font-weight: 600; }
  .review-text { margin: 0.25rem 0 0; line-height: 1.5; color: var(--pico-color, #e2e8f0); white-space: pre-wrap; }
  .owner-response { margin-top: 0.35rem; padding-left: 0.75rem; border-left: 2px solid var(--pico-muted-border-color, #2a2c33); font-size: 0.9em; }
  .review-images { display: flex; gap: 0.35rem; margin-top: 0.35rem; }
  .img-link { font-size: 1em; }

  .small { font-size: 0.85em; }
  .muted { color: var(--pico-muted-color, #888); }
  .error { color: var(--pico-color-red-550, #c0392b); }
  .pagination { display: flex; align-items: center; justify-content: center; gap: 1rem; margin: 1rem 0; }
</style>
