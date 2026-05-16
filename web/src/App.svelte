<script lang="ts">
  import Sidebar from './components/Sidebar.svelte';
  import Header from './components/Header.svelte';
  import ProgressDashboard from './components/ProgressDashboard.svelte';
  import JobsPageV2 from './components/JobsPageV2.svelte';
  import WorkerBoard from './components/WorkerBoard.svelte';
  import PlacesBrowser from './components/PlacesBrowser.svelte';
  import LogsPage from './components/LogsPage.svelte';
  import { route, toasts } from './lib/stores.ts';
  import { ws, type WSStatus } from './lib/ws.ts';

  let wsStatus: WSStatus = $state('closed');

  $effect(() => {
    const unsub = ws.subscribe((s) => { wsStatus = s; });
    ws.connect();
    return () => {
      unsub();
      ws.disconnect();
    };
  });
</script>

<div class="app-shell">
  <Sidebar />
  <Header {wsStatus} />
  <main class="app-main">
    <div class="container">
      {#if $route.section === 'dashboard'}
        <ProgressDashboard />
      {:else if $route.section === 'jobs'}
        <JobsPageV2 />
      {:else if $route.section === 'workers'}
        <WorkerBoard />
      {:else if $route.section === 'places'}
        <PlacesBrowser />
      {:else if $route.section === 'logs'}
        <LogsPage />
      {/if}
    </div>
  </main>
</div>

<!-- Toast notifications -->
<div class="toast-container">
  {#each $toasts as t (t.id)}
    <div class="toast {t.kind}">{t.message}</div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: 1.25rem;
    right: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: .5rem;
    z-index: 100;
  }
  .toast {
    padding: .65rem 1rem;
    border-radius: 10px;
    background: var(--pico-card-background-color);
    color: var(--pico-color);
    border: 1px solid var(--pico-muted-border-color);
    box-shadow: 0 8px 28px rgba(0,0,0,.4);
    font-size: .88rem;
    min-width: 220px;
    animation: slidein .2s ease;
  }
  .toast.success { border-left: 3px solid #22c55e; }
  .toast.error   { border-left: 3px solid #ef4444; }
  .toast.info    { border-left: 3px solid #22c55e; }
  .toast.warn    { border-left: 3px solid #eab308; }
  @keyframes slidein {
    from { opacity: 0; transform: translateX(20px); }
    to   { opacity: 1; transform: translateX(0); }
  }
</style>
