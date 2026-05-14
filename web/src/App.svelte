<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Sidebar from './components/Sidebar.svelte';
  import Header from './components/Header.svelte';
  import ProgressDashboard from './components/ProgressDashboard.svelte';
  import JobsPageV2 from './components/JobsPageV2.svelte';
  import WorkerBoard from './components/WorkerBoard.svelte';
  import FileBrowser from './components/FileBrowser.svelte';
  import LogsPage from './components/LogsPage.svelte';
  import { section, toasts } from './lib/stores.ts';
  import { ws } from './lib/ws.ts';

  let wsStatus: 'connecting' | 'open' | 'closed' | 'error' = 'closed';
  let unsubWS: (() => void) | null = null;

  onMount(() => {
    unsubWS = ws.subscribe((s) => {
      wsStatus = s;
    });
    ws.connect();
  });

  onDestroy(() => {
    if (unsubWS) unsubWS();
    ws.disconnect();
  });
</script>

<div class="app-shell">
  <Sidebar />
  <Header />
  <main class="app-main">
    <div class="container">
      {#if $section === 'dashboard'}
        <ProgressDashboard />
      {:else if $section === 'jobs'}
        <JobsPageV2 />
      {:else if $section === 'workers'}
        <WorkerBoard />
      {:else if $section === 'files'}
        <FileBrowser />
      {:else if $section === 'logs'}
        <LogsPage />
      {/if}
    </div>
  </main>
</div>

<!-- WebSocket status indicator (bottom-left corner) -->
<div class="ws-indicator ws-{wsStatus}" title="WebSocket {wsStatus}">
  <span class="dot"></span>
  <span class="label">live</span>
</div>

<!-- Toast notifications -->
<div class="toast-container">
  {#each $toasts as t (t.id)}
    <div class="toast {t.kind}">{t.message}</div>
  {/each}
</div>

<style>
  .ws-indicator {
    position: fixed;
    bottom: 1rem;
    left: 1rem;
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.3rem 0.6rem;
    border-radius: 999px;
    background: var(--pico-card-background-color, rgba(0, 0, 0, 0.5));
    border: 1px solid var(--pico-muted-border-color, #2a2c33);
    font-size: 0.75em;
    color: var(--pico-muted-color, #888);
    z-index: 99;
    user-select: none;
  }
  .ws-indicator .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #888;
  }
  .ws-open .dot   { background: #22c55e; box-shadow: 0 0 6px #22c55e; }
  .ws-connecting .dot { background: #eab308; animation: pulse 1.2s ease-in-out infinite; }
  .ws-closed .dot { background: #888; }
  .ws-error .dot  { background: #ef4444; }
  .ws-open .label { color: #22c55e; }
  .ws-error .label { color: #ef4444; }
  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50%      { opacity: 0.3; }
  }

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
