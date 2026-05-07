<script>
  import { section, activeJobsCount } from '../lib/stores.js';

  const items = [
    { id: 'dashboard', label: 'Dashboard', icon: '◐' },
    { id: 'jobs', label: 'Jobs', icon: '▶' },
    { id: 'files', label: 'Files', icon: '◫' },
    { id: 'logs', label: 'Logs', icon: '≡' },
  ];

  function go(id) {
    section.set(id);
  }
</script>

<aside class="app-sidebar">
  <div class="logo">
    <span class="logo-mark">G</span>
    <span>Scrapper</span>
  </div>

  <nav>
    {#each items as it}
      <div
        class="nav-item"
        class:active={$section === it.id}
        on:click={() => go(it.id)}
        on:keydown={(e) => e.key === 'Enter' && go(it.id)}
        tabindex="0"
        role="button"
      >
        <span><span style="margin-right:.5rem;opacity:.7">{it.icon}</span>{it.label}</span>
        {#if it.id === 'jobs' && $activeJobsCount > 0}
          <span class="badge">{$activeJobsCount}</span>
        {/if}
      </div>
    {/each}
  </nav>

  <div class="footer">
    <span class="muted">Google Maps Scraper</span>
    <span class="muted">v0.2.0</span>
  </div>
</aside>
