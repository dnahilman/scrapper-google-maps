<script lang="ts">
  import { v1, type JobV1 } from '../lib/api_v1.ts';
  import { notify } from '../lib/stores.ts';
  import CitySelector from './CitySelector.svelte';
  import KelurahanPicker from './KelurahanPicker.svelte';

  interface Props { onStarted?: (job: JobV1) => void; }
  let { onStarted }: Props = $props();

  let submitting = $state(false);
  let cityId = $state('');
  let keyword = $state('cafe');
  let kelurahanNames: string[] = $state([]);
  let maxAttempts = $state(3);
  let maxReviews = $state(200);
  let minReviewAgeDays: number | '' = $state('');
  let maxReviewAgeDays: number | '' = $state('');
  let limitPerKelurahan: number | '' = $state('');
  let enableEmailCrawl = $state(false);

  async function submit(): Promise<void> {
    if (!cityId) { notify('Pilih kota dulu', 'warn'); return; }
    if (!keyword.trim()) { notify('Keyword wajib diisi', 'warn'); return; }
    submitting = true;
    try {
      const options: Record<string, unknown> = { max_reviews_per_place: maxReviews };
      if (limitPerKelurahan !== '' && Number(limitPerKelurahan) > 0) options.limit_per_kelurahan = Number(limitPerKelurahan);
      if (minReviewAgeDays !== '' && Number(minReviewAgeDays) >= 0) options.min_review_age_days = Number(minReviewAgeDays);
      if (maxReviewAgeDays !== '' && Number(maxReviewAgeDays) > 0) options.max_review_age_days = Number(maxReviewAgeDays);
      if (enableEmailCrawl) options.enable_email_crawl = true;

      const job = await v1.createJob({
        city_id: cityId,
        keyword: keyword.trim(),
        kelurahan_names: kelurahanNames.length > 0 ? kelurahanNames : undefined,
        max_attempts: maxAttempts,
        options,
      });
      notify(`Job ${job.id.slice(0, 8)}… dibuat (${job.total_tasks} tasks)`, 'success');
      onStarted?.(job);
    } catch (e) {
      notify(`Gagal start job: ${(e as Error).message}`, 'error');
    } finally {
      submitting = false;
    }
  }
</script>

<div class="launcher">
  <form onsubmit={(e) => { e.preventDefault(); void submit(); }}>
    <CitySelector onSelect={(c) => { cityId = c.id; }} />

    <label>
      Keyword
      <input type="text" bind:value={keyword} placeholder="cafe / barbershop / kuliner" required />
    </label>

    {#if cityId}
      <KelurahanPicker {cityId} onChange={(names) => { kelurahanNames = names; }} />
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
      </fieldset>

      <label class="checkbox-row">
        <input type="checkbox" bind:checked={enableEmailCrawl} />
        <span>Crawl website untuk extract emails <span class="muted">(menambah HTTP fetch per place)</span></span>
      </label>
    </details>

    <footer>
      <button type="submit" disabled={submitting || !cityId || !keyword.trim() || kelurahanNames.length === 0}>
        {submitting ? 'Starting…' : `Start ${kelurahanNames.length} task${kelurahanNames.length === 1 ? '' : 's'}`}
      </button>
    </footer>
  </form>
</div>

<style>
  .launcher { max-width: 720px; }
  details summary { cursor: pointer; margin: 0.5rem 0; }
  .grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem 0.75rem; }
  .reviews-fieldset { border: 1px solid var(--pico-muted-border-color, #2a2c33); border-radius: 6px; padding: 0.5rem 0.75rem 0.25rem; margin: 0.5rem 0; }
  .reviews-fieldset legend { padding: 0 0.4rem; font-size: 0.85em; color: var(--pico-muted-color, #888); }
  .checkbox-row { display: flex; align-items: center; gap: 0.5rem; margin: 0.5rem 0; }
  .checkbox-row input[type="checkbox"] { width: auto; margin: 0; }
  footer { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1rem; }
</style>
