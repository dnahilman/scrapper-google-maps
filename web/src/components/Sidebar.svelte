<script lang="ts">
  import { section, activeJobsCount, type Section } from '../lib/stores.ts';

  interface NavItem {
    id: Section;
    label: string;
    icon: string;
  }

  const items: NavItem[] = [
    { id: 'dashboard', label: 'Dashboard', icon: '◐' },
    { id: 'jobs',      label: 'Jobs',      icon: '▶' },
    { id: 'files',     label: 'Files',     icon: '◫' },
    { id: 'logs',      label: 'Logs',      icon: '≡' },
  ];

  function go(id: Section): void {
    section.set(id);
  }
</script>

<aside class="app-sidebar">
  <div class="logo">
    <span class="logo-mark">S</span>
    <span>Scrapper</span>
  </div>

  <nav>
    {#each items as it}
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div
        class="nav-item"
        class:active={$section === it.id}
        on:click={() => go(it.id)}
        on:keydown={(e) => e.key === 'Enter' && go(it.id)}
        tabindex="0"
        role="button"
        aria-current={$section === it.id ? 'page' : undefined}
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
    <span class="muted">v0.2.0</span>
  </div>
</aside>
