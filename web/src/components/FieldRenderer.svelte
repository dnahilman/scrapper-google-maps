<script lang="ts">
  interface Props {
    value: unknown;
    label?: string;
  }
  let { value, label }: Props = $props();

  function isPlainObject(v: unknown): v is Record<string, unknown> {
    return typeof v === 'object' && v !== null && !Array.isArray(v);
  }

  function isArrayOfStrings(v: unknown): v is string[] {
    return Array.isArray(v) && v.every((x) => typeof x === 'string');
  }

  function isEmpty(v: unknown): boolean {
    if (v == null) return true;
    if (typeof v === 'string') return v === '';
    if (Array.isArray(v)) return v.length === 0;
    if (isPlainObject(v)) return Object.keys(v).length === 0;
    return false;
  }

  let expanded = $state(false);
</script>

{#if isEmpty(value)}
  <span class="muted">—</span>
{:else if typeof value === 'boolean'}
  <span class={value ? 'yes' : 'no'}>{value ? '✓' : '✕'}</span>
{:else if typeof value === 'number'}
  {value}
{:else if typeof value === 'string'}
  {value}
{:else if isArrayOfStrings(value)}
  <ul class="tag-list">
    {#each value as item}
      <li><span class="tag">{item}</span></li>
    {/each}
  </ul>
{:else if Array.isArray(value)}
  <div class="json-block">
    <button class="toggle" onclick={() => (expanded = !expanded)}>
      [{value.length} items] {expanded ? '▲' : '▼'}
    </button>
    {#if expanded}
      <pre class="json">{JSON.stringify(value, null, 2)}</pre>
    {/if}
  </div>
{:else if isPlainObject(value)}
  <div class="json-block">
    <button class="toggle" onclick={() => (expanded = !expanded)}>
      &#123;{Object.keys(value).length} keys&#125; {expanded ? '▲' : '▼'}
    </button>
    {#if expanded}
      <pre class="json">{JSON.stringify(value, null, 2)}</pre>
    {/if}
  </div>
{:else}
  <span class="muted">{String(value)}</span>
{/if}

<style>
  .muted { color: var(--pico-muted-color, #888); }
  .yes { color: #22c55e; }
  .no  { color: #ef4444; }
  .tag-list { list-style: none; padding: 0; margin: 0; display: flex; flex-wrap: wrap; gap: 0.25rem; }
  .tag-list li { margin: 0; }
  .tag { font-size: 0.78em; padding: 0.1rem 0.45rem; border-radius: 999px; background: var(--pico-secondary-background, rgba(255,255,255,.06)); border: 1px solid var(--pico-muted-border-color, #333); }
  .json-block { display: inline-block; }
  .toggle { background: none; border: 1px solid var(--pico-muted-border-color, #333); border-radius: 4px; padding: 0.1rem 0.4rem; font-size: 0.78em; cursor: pointer; color: var(--pico-muted-color, #888); }
  .json { margin: 0.25rem 0 0; font-size: 0.75em; background: var(--pico-code-background-color, #0d1117); border-radius: 6px; padding: 0.5rem 0.75rem; overflow-x: auto; max-height: 300px; overflow-y: auto; white-space: pre; }
</style>
