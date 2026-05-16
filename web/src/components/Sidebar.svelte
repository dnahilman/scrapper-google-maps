<script lang="ts">
  import { route, activeJobsCount, navigate, type Section } from '../lib/stores.ts';

  const items = [
    { id: 'dashboard' as Section, label: 'Dashboard', icon: '◐' },
    { id: 'jobs'      as Section, label: 'Jobs',      icon: '▶' },
    { id: 'workers'   as Section, label: 'Workers',   icon: '◉' },
    { id: 'places'    as Section, label: 'Places',    icon: '◆' },
    { id: 'logs'      as Section, label: 'Logs',      icon: '≡' },
  ];
</script>

<aside class="app-sidebar">
  <div class="logo">
    <span class="logo-mark">S</span>
    <span>Scrapper</span>
  </div>

  <nav>
    {#each items as it}
      <div
        class="nav-item"
        class:active={$route.section === it.id}
        onclick={() => navigate(`#${it.id}`)}
        onkeydown={(e) => e.key === 'Enter' && navigate(`#${it.id}`)}
        tabindex="0"
        role="button"
        aria-current={$route.section === it.id ? 'page' : undefined}
      >
        <span><span class="nav-icon">{it.icon}</span>{it.label}</span>
        {#if it.id === 'jobs' && $activeJobsCount > 0}
          <span class="badge">{$activeJobsCount}</span>
        {/if}
      </div>
    {/each}
  </nav>

  <div class="footer">
    <span class="muted">Google Maps Scraper</span>
    <span class="muted">v0.1.4</span>
  </div>
</aside>
