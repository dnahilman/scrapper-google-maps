<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { api, type JobResponse } from '../lib/api.ts';
  import { notify } from '../lib/stores.ts';

  export let initialKeyword = 'cafe';

  const dispatch = createEventDispatcher<{ started: JobResponse }>();

  let open = false;
  let submitting = false;

  let keyword = initialKeyword;
  let shard = '';
  let kelurahan = '';
  let limit = '';
  let resume = true;
  let dryRun = false;

  $: if (initialKeyword) keyword = initialKeyword;

  async function submit(): Promise<void> {
    submitting = true;
    try {
      const body = {
        keyword: keyword.trim(),
        shard: shard.trim() || null,
        kelurahan: kelurahan.trim() || null,
        limit: limit ? Number(limit) : null,
        resume,
        dry_run: dryRun,
      };
      const job = await api.startJob(body);
      notify(`Job ${job.job_id} started (PID ${job.pid})`, 'success');
      open = false;
      dispatch('started', job);
    } catch (e) {
      notify(`Gagal start job: ${(e as Error).message}`, 'error');
    } finally {
      submitting = false;
    }
  }
</script>

<button class="icon-btn" on:click={() => (open = true)}>
  ▶ Start scrape
</button>

{#if open}
  <dialog open>
    <article style="max-width: 520px;">
      <header style="display:flex;justify-content:space-between;align-items:center">
        <strong>Start scrape job</strong>
        <button class="ghost icon-btn" type="button" on:click={() => (open = false)}>✕</button>
      </header>

      <form on:submit|preventDefault={submit}>
        <label>
          Keyword
          <input type="text" bind:value={keyword} placeholder="cafe" required />
        </label>

        <label>
          Shard <span class="muted">(opsional, format K/N — mis. 1/2)</span>
          <input type="text" bind:value={shard} placeholder="1/2" pattern="\d+/\d+" />
        </label>

        <label>
          Kelurahan filter <span class="muted">(opsional, substring)</span>
          <input type="text" bind:value={kelurahan} placeholder="Antapani" />
        </label>

        <label>
          Limit <span class="muted">(opsional, max places per kelurahan)</span>
          <input type="number" bind:value={limit} placeholder="50" min="1" />
        </label>

        <fieldset>
          <label><input type="checkbox" bind:checked={resume} /> Resume (skip kelurahan yang sudah done)</label>
          <label><input type="checkbox" bind:checked={dryRun} /> Dry run (list saja, tidak scrape)</label>
        </fieldset>

        <footer style="display:flex;justify-content:flex-end;gap:.5rem;margin-top:1rem">
          <button type="button" class="ghost" on:click={() => (open = false)}>Cancel</button>
          <button type="submit" disabled={submitting || !keyword.trim()}>
            {submitting ? 'Starting…' : 'Start job'}
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
    width: 92%;
    margin: 0;
  }
</style>
