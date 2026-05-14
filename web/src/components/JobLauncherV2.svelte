<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { v1, type JobV1, type City } from '../lib/api_v1.ts';
  import { notify } from '../lib/stores.ts';
  import CitySelector from './CitySelector.svelte';
  import KelurahanPicker from './KelurahanPicker.svelte';

  const dispatch = createEventDispatcher<{ started: JobV1 }>();

  let open = false;
  let submitting = false;

  let cityId = '';
  let cityName = '';
  let keyword = 'cafe';
  let kelurahanNames: string[] = [];
  let maxAttempts = 3;
  let maxReviews = 200;
  let minReviewAgeDays: number | '' = '';
  let maxReviewAgeDays: number | '' = '';
  let limitPerKelurahan: number | '' = '';
  let enableEmailCrawl = false;

  function onCitySelect(ev: CustomEvent<City>): void {
    cityId = ev.detail.id;
    cityName = ev.detail.name;
  }

  function onKelurahanChange(ev: CustomEvent<string[]>): void {
    kelurahanNames = ev.detail;
  }

  async function submit(): Promise<void> {
    if (!cityId) {
      notify('Pilih kota dulu', 'warn');
      return;
    }
    if (!keyword.trim()) {
      notify('Keyword wajib diisi', 'warn');
      return;
    }
    submitting = true;
    try {
      const options: Record<string, unknown> = {
        max_reviews_per_place: maxReviews,
      };
      if (limitPerKelurahan !== '' && Number(limitPerKelurahan) > 0) {
        options.limit_per_kelurahan = Number(limitPerKelurahan);
      }
      if (minReviewAgeDays !== '' && Number(minReviewAgeDays) >= 0) {
        options.min_review_age_days = Number(minReviewAgeDays);
      }
      if (maxReviewAgeDays !== '' && Number(maxReviewAgeDays) > 0) {
        options.max_review_age_days = Number(maxReviewAgeDays);
      }
      if (enableEmailCrawl) {
        options.enable_email_crawl = true;
      }
      const body = {
        city_id: cityId,
        keyword: keyword.trim(),
        kelurahan_names: kelurahanNames.length > 0 ? kelurahanNames : undefined,
        max_attempts: maxAttempts,
        options,
      };
      const job = await v1.createJob(body);
      notify(`Job ${job.id.slice(0, 8)}… created (${job.total_tasks} tasks)`, 'success');
      open = false;
      dispatch('started', job);
    } catch (e) {
      notify(`Gagal start job: ${(e as Error).message}`, 'error');
    } finally {
      submitting = false;
    }
  }
</script>

<button class="icon-btn primary" on:click={() => (open = true)}>
  ▶ Start scrape job
</button>

{#if open}
  <dialog open>
    <article style="max-width: 720px; width: 92%;">
      <header style="display:flex;justify-content:space-between;align-items:center">
        <strong>Start scrape job</strong>
        <button class="ghost icon-btn" type="button" on:click={() => (open = false)}>✕</button>
      </header>

      <form on:submit|preventDefault={submit}>
        <CitySelector on:select={onCitySelect} />

        <label>
          Keyword
          <input type="text" bind:value={keyword} placeholder="cafe / barbershop / kuliner" required />
        </label>

        {#if cityId}
          <KelurahanPicker {cityId} on:change={onKelurahanChange} />
        {/if}

        <details>
          <summary>Opsi lanjutan</summary>

          <div class="grid-2">
            <label>
              Max attempts per task
              <input type="number" bind:value={maxAttempts} min="1" max="10" />
            </label>
            <label>
              Limit places per kelurahan <span class="muted">(0 / kosong = no limit)</span>
              <input type="number" bind:value={limitPerKelurahan} min="0" placeholder="no limit" />
            </label>
          </div>

          <fieldset class="reviews-fieldset">
            <legend>Reviews filter</legend>

            <label>
              Max reviews per place
              <input type="number" bind:value={maxReviews} min="0" max="2000" />
            </label>

            <div class="grid-2">
              <label>
                Min age (hari) <span class="muted">drop yang lebih baru</span>
                <input type="number" bind:value={minReviewAgeDays} min="0" placeholder="0" />
              </label>
              <label>
                Max age (hari) <span class="muted">drop yang lebih tua</span>
                <input type="number" bind:value={maxReviewAgeDays} min="0" placeholder="no limit" />
              </label>
            </div>
            <p class="muted small">
              Contoh: min=90, max=300 → hanya review berumur 3–10 bulan. Kosong = pakai default worker
              (umur ≤ MAX_REVIEW_AGE_DAYS env, default 730).
            </p>
          </fieldset>

          <label class="checkbox-row">
            <input type="checkbox" bind:checked={enableEmailCrawl} />
            <span>Crawl website untuk extract emails <span class="muted">(menambah HTTP fetch per place)</span></span>
          </label>
        </details>

        <footer style="display:flex;justify-content:flex-end;gap:.5rem;margin-top:1rem">
          <button type="button" class="ghost" on:click={() => (open = false)}>Cancel</button>
          <button type="submit" disabled={submitting || !cityId || !keyword.trim() || kelurahanNames.length === 0}>
            {submitting ? 'Starting…' : `Start ${kelurahanNames.length} task${kelurahanNames.length === 1 ? '' : 's'}`}
          </button>
        </footer>
      </form>
    </article>
  </dialog>
{/if}

<style>
  dialog {
    position: fixed;
    inset: 0;
    z-index: 50;
    background: rgba(0, 0, 0, 0.6);
    border: none;
    margin: 0;
    width: 100%;
    height: 100%;
    display: grid;
    place-items: center;
  }
  dialog article {
    margin: 0;
    max-height: 92vh;
    overflow-y: auto;
  }
  details summary {
    cursor: pointer;
    margin: 0.5rem 0;
  }
  .grid-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem 0.75rem;
  }
  .reviews-fieldset {
    border: 1px solid var(--pico-muted-border-color, #2a2c33);
    border-radius: 6px;
    padding: 0.5rem 0.75rem 0.25rem;
    margin: 0.5rem 0;
  }
  .reviews-fieldset legend {
    padding: 0 0.4rem;
    font-size: 0.85em;
    color: var(--pico-muted-color, #888);
  }
  .checkbox-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0.5rem 0;
  }
  .checkbox-row input[type="checkbox"] { width: auto; margin: 0; }
  .small { font-size: 0.85em; }
</style>
