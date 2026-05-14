<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fmtUptime } from '../lib/api.ts';
  import { v1, type HealthV1 } from '../lib/api_v1.ts';
  import { section } from '../lib/stores.ts';

  let healthState: 'healthy' | 'down' | 'checking' = 'checking';
  let lastLatency: number | null = null;
  let lastCheck: Date | null = null;
  let info: HealthV1 | null = null;
  let timer: ReturnType<typeof setInterval>;

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

  onMount(() => {
    check();
    timer = setInterval(check, 10000);
  });

  onDestroy(() => clearInterval(timer));

  $: tooltip =
    info != null
      ? `uptime ${fmtUptime(info.uptime_seconds)} · db ${info.db} · workers ${info.workers_online}/${info.workers_total} · ${lastLatency} ms · checked ${
          lastCheck?.toLocaleTimeString('id-ID', { hour12: false }) ?? '-'
        }`
      : 'API tidak merespons';

  $: label =
    healthState === 'healthy'
      ? `${info?.workers_online ?? 0}/${info?.workers_total ?? 0} workers`
      : healthState === 'down'
        ? 'down'
        : 'checking…';
</script>

<header class="app-header">
  <div>
    <p class="breadcrumb">
      <span class="muted">Web UI</span>
      <span class="muted">/</span>
      <span style="text-transform:capitalize">{$section}</span>
    </p>
  </div>
  <div class="right">
    <span class="health-indicator {healthState}" title={tooltip}>
      <span class="dot"></span>
      <span>{label}</span>
    </span>
  </div>
</header>
