<script lang="ts">
  import { v1, type HealthV1, fmtUptime } from '../lib/api_v1.ts';
  import { route } from '../lib/stores.ts';
  import type { WSStatus } from '../lib/ws.ts';

  interface Props { wsStatus: WSStatus; }
  let { wsStatus }: Props = $props();

  let healthState = $state<'healthy' | 'down' | 'checking'>('checking');
  let lastLatency = $state<number | null>(null);
  let lastCheck = $state<Date | null>(null);
  let info = $state<HealthV1 | null>(null);

  async function check(): Promise<void> {
    const t0 = performance.now();
    try {
      info = await v1.health();
      healthState = info.status === 'ok' && info.db === 'ok' ? 'healthy' : 'down';
    } catch {
      healthState = 'down';
      info = null;
    }
    lastLatency = Math.round(performance.now() - t0);
    lastCheck = new Date();
  }

  $effect(() => {
    check();
    const timer = setInterval(check, 30_000);
    return () => clearInterval(timer);
  });

  let tooltip = $derived(
    info
      ? `uptime ${fmtUptime(info?.uptime_seconds ?? 0)} · db ${info?.db} · workers ${info?.workers_online}/${info?.workers_total} · ${lastLatency} ms · checked ${lastCheck?.toLocaleTimeString('id-ID', { hour12: false }) ?? '-'}`
      : 'API tidak merespons'
  );

  let label = $derived(
    healthState === 'healthy'
      ? `${info?.workers_online ?? 0}/${info?.workers_total ?? 0} workers`
      : healthState === 'down'
        ? 'down'
        : 'checking…'
  );
</script>

<header class="app-header">
  <div>
    <p class="breadcrumb">
      <span class="muted">Web UI</span>
      <span class="muted">/</span>
      <span style="text-transform:capitalize">{$route.section}</span>
    </p>
  </div>
  <div class="right">
    <span class="health-indicator {healthState}" title={tooltip}>
      <span class="dot"></span>
      <span>{label}</span>
    </span>
    <span class="ws-indicator ws-{wsStatus}" title="WebSocket {wsStatus}">
      <span class="dot"></span>
      <span class="label">live</span>
    </span>
  </div>
</header>

<style>
  .right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }
  .ws-indicator {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.25rem 0.55rem;
    border-radius: 999px;
    background: var(--pico-card-background-color, rgba(0,0,0,0.5));
    border: 1px solid var(--pico-muted-border-color, #2a2c33);
    font-size: 0.75em;
    color: var(--pico-muted-color, #888);
    user-select: none;
  }
  .ws-indicator .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #888;
    flex-shrink: 0;
  }
  .ws-open .dot        { background: #22c55e; box-shadow: 0 0 6px #22c55e; }
  .ws-connecting .dot  { background: #eab308; animation: pulse 1.2s ease-in-out infinite; }
  .ws-closed .dot      { background: #888; }
  .ws-error .dot       { background: #ef4444; }
  .ws-open .label      { color: #22c55e; }
  .ws-error .label     { color: #ef4444; }
  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.3; }
  }
</style>
