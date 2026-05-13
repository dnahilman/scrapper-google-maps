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
      const body = {
        city_id: cityId,
        keyword: keyword.trim(),
        kelurahan_names: kelurahanNames.length > 0 ? kelurahanNames : undefined,
        max_attempts: maxAttempts,
        options: {
          max_reviews_per_place: maxReviews,
        },
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
          <label>
            Max attempts per task
            <input type="number" bind:value={maxAttempts} min="1" max="10" />
          </label>
          <label>
            Max reviews per place
            <input type="number" bind:value={maxReviews} min="0" max="2000" />
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
</style>
