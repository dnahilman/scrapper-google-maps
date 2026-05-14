<script lang="ts">
  import { onMount } from 'svelte';
  import { v1, fmtTimeAgo } from '../lib/api_v1.ts';

  export let placeId: string;
  export let onClose: () => void;

  // Place + reviews loaded by ID. Full payload pulled from /api/v1/places/:id.
  let place: any = null;
  let reviews: any[] = [];
  let loading = true;
  let error = '';

  async function load(): Promise<void> {
    try {
      const [p, r] = await Promise.all([
        v1.list ? null : null, // placeholder
        null,
      ]);
      // Use raw fetch since v1 doesn't have a typed wrapper for these yet.
      const pRes = await fetch(`/api/v1/places/${encodeURIComponent(placeId)}`);
      if (!pRes.ok) throw new Error(`${pRes.status} ${pRes.statusText}`);
      place = await pRes.json();
      const rRes = await fetch(`/api/v1/places/${encodeURIComponent(placeId)}/reviews?limit=100`);
      reviews = rRes.ok ? await rRes.json() : [];
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function fmtLatLng(p: any): string {
    if (!p?.latitude || !p?.longitude) return '–';
    return `${p.latitude.toFixed(6)}, ${p.longitude.toFixed(6)}`;
  }
</script>

<dialog open>
  <article class="detail">
    <header>
      <div>
        <strong>{place?.title ?? '…'}</strong>
        {#if place?.category}<div class="muted small">{place.category}</div>{/if}
      </div>
      <button class="ghost icon-btn" type="button" on:click={onClose}>✕</button>
    </header>

    {#if loading}
      <p class="muted">Loading…</p>
    {:else if error}
      <p class="error">⚠ {error}</p>
    {:else if place}
      <div class="tabs">
        <a href={place.link ?? '#'} target="_blank" rel="noopener" class="muted small">Open in Google Maps ↗</a>
      </div>

      <!-- INFO -->
      <section>
        <h4>Info</h4>
        <dl>
          <dt>Rating</dt><dd>{place.review_rating?.toFixed?.(1) ?? '–'} · {place.review_count ?? 0} reviews</dd>
          <dt>Address</dt><dd>{place.address ?? '–'}</dd>
          <dt>Phone</dt><dd>{place.phone || '–'}</dd>
          <dt>Website</dt>
          <dd>{#if place.website}<a href={place.website} target="_blank" rel="noopener">{place.website}</a>{:else}–{/if}</dd>
          <dt>Plus code</dt><dd><code>{place.plus_code ?? '–'}</code></dd>
          <dt>Coordinates</dt><dd><code>{fmtLatLng(place)}</code></dd>
          <dt>Status</dt><dd>{place.status}</dd>
          <dt>Price</dt><dd>{place.price || '–'}</dd>
          <dt>Timezone</dt><dd>{place.timezone || '–'}</dd>
          <dt>Place ID</dt><dd><code class="small">{place.place_id}</code></dd>
          <dt>CID</dt><dd><code class="small">{place.cid || '–'}</code></dd>
          <dt>Scraped</dt><dd>{fmtTimeAgo(place.scraped_at)}</dd>
        </dl>
      </section>

      <!-- COMPLETE ADDRESS -->
      {#if place.complete_address}
        <section>
          <h4>Complete address</h4>
          <dl class="ca">
            {#each ['street','borough','city','state','postal_code','country'] as k}
              {#if place.complete_address[k]}
                <dt>{k.replace('_',' ')}</dt><dd>{place.complete_address[k]}</dd>
              {/if}
            {/each}
          </dl>
        </section>
      {/if}

      <!-- HOURS -->
      {#if place.open_hours && Object.keys(place.open_hours).length > 0}
        <section>
          <h4>Opening hours</h4>
          <table class="hours">
            <tbody>
              {#each Object.keys(place.open_hours) as day}
                <tr>
                  <td><strong>{day}</strong></td>
                  <td>{(place.open_hours[day] || []).join(', ')}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </section>
      {/if}

      <!-- ABOUT -->
      {#if place.about && place.about.length > 0}
        <section>
          <h4>About</h4>
          {#each place.about as sec}
            <div class="about-sec">
              <div class="about-title">{sec.name}</div>
              <ul class="about-options">
                {#each sec.options as opt}
                  <li class:disabled={!opt.enabled}>
                    <span class="check">{opt.enabled ? '✓' : '✕'}</span>
                    {opt.name}
                  </li>
                {/each}
              </ul>
            </div>
          {/each}
        </section>
      {/if}

      <!-- MENU -->
      {#if place.menu && (place.menu.items?.length > 0 || place.menu.photos?.length > 0)}
        <section>
          <h4>Menu</h4>
          {#if place.menu.items?.length > 0}
            <ul class="menu-items">
              {#each place.menu.items as it}
                <li><strong>{it.name}</strong> {#if it.price}<span class="muted">— {it.price}</span>{/if}</li>
              {/each}
            </ul>
          {/if}
          {#if place.menu.photos?.length > 0}
            <div class="muted small">{place.menu.photos.length} menu photos</div>
          {/if}
        </section>
      {/if}

      <!-- IMAGES -->
      {#if place.images?.length > 0}
        <section>
          <h4>Photos ({place.images.length})</h4>
          <div class="gallery">
            {#each place.images.slice(0, 12) as img}
              <img src={img.image} alt={img.title || place.title} loading="lazy" />
            {/each}
          </div>
        </section>
      {/if}

      <!-- REVIEWS -->
      {#if reviews.length > 0}
        <section>
          <h4>Reviews ({reviews.length})</h4>
          <ul class="reviews">
            {#each reviews as r (r.id)}
              <li>
                <div class="r-head">
                  <strong>{r.name || '(anonymous)'}</strong>
                  <span class="muted small">{r.when || fmtTimeAgo(r.created_at)} · {r.rating ?? '–'}/5</span>
                </div>
                {#if r.description}<p>{r.description}</p>{/if}
                {#if r.owner_response?.text}
                  <div class="owner-resp">
                    <span class="muted small">↳ Owner response:</span>
                    <p>{r.owner_response.text}</p>
                  </div>
                {/if}
              </li>
            {/each}
          </ul>
        </section>
      {/if}
    {/if}
  </article>
</dialog>

<style>
  dialog {
    position: fixed;
    inset: 0;
    z-index: 60;
    background: rgba(0, 0, 0, 0.6);
    border: none;
    margin: 0;
    width: 100%;
    height: 100%;
    display: grid;
    place-items: center;
    padding: 0;
  }
  .detail {
    width: 92%;
    max-width: 820px;
    max-height: 92vh;
    overflow-y: auto;
    margin: 0;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
    padding-bottom: 0.5rem;
    border-bottom: 1px solid var(--pico-muted-border-color, #2a2c33);
  }
  section { margin-top: 1rem; }
  h4 {
    margin: 0 0 0.4rem;
    font-size: 0.95em;
    color: var(--pico-muted-color, #888);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  dl {
    margin: 0;
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.25rem 0.75rem;
    font-size: 0.9em;
  }
  dl.ca { grid-template-columns: max-content 1fr; }
  dt { color: var(--pico-muted-color, #888); text-transform: capitalize; }
  dd { margin: 0; word-break: break-word; }
  .small { font-size: 0.8em; }
  .error { color: var(--pico-color-red-550, #c0392b); }

  table.hours { width: auto; font-size: 0.9em; }
  table.hours td { padding: 0.2rem 0.6rem; border: 0; }

  .about-sec { margin-bottom: 0.6rem; }
  .about-title { font-weight: 500; margin-bottom: 0.2rem; }
  .about-options {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 0.15rem 0.5rem;
    font-size: 0.85em;
  }
  .about-options .check { display: inline-block; width: 1em; color: #22c55e; }
  .about-options li.disabled .check { color: #ef4444; }
  .about-options li.disabled { color: var(--pico-muted-color, #888); }

  .menu-items {
    list-style: none;
    padding: 0;
    margin: 0;
    columns: 2;
    column-gap: 1rem;
    font-size: 0.9em;
  }
  .menu-items li { break-inside: avoid; margin-bottom: 0.25rem; }

  .gallery {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 0.4rem;
  }
  .gallery img {
    width: 100%;
    aspect-ratio: 4/3;
    object-fit: cover;
    border-radius: 6px;
    background: var(--pico-muted-border-color, #2a2c33);
  }

  .reviews {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.75rem;
  }
  .reviews > li {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--pico-muted-border-color, #2a2c33);
    border-radius: 6px;
  }
  .r-head { display: flex; justify-content: space-between; align-items: baseline; gap: 0.5rem; }
  .reviews p { margin: 0.25rem 0; font-size: 0.9em; }
  .owner-resp {
    margin-top: 0.4rem;
    padding-left: 0.6rem;
    border-left: 2px solid var(--pico-muted-border-color, #444);
  }
</style>
