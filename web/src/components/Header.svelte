<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { api, fmtUptime, type HealthResponse } from '../lib/api.ts';
  import { section } from '../lib/stores.ts';

  let healthState: 'healthy' | 'down' | 'checking' = 'checking';
  let lastLatency: number | null = null;
  let lastCheck: Date | null = null;
  let info: HealthResponse | null = null;
  let timer: ReturnType<typeof setInterval>;

  async function check(): Promise<void> {
    const t0 = performance.now();
    try {
      info = await api.health();
      healthState = info.ok ? 'healthy' : 'down';
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
      ? `version ${info.version} · uptime ${fmtUptime(info.uptime_sec)} · ${lastLatency} ms · checked ${
          lastCheck?.toLocaleTimeString('id-ID', { hour12: false }) ?? '-'
        }`
      : 'API tidak merespons';

  $: label =
    healthState === 'healthy'
      ? 'healthy'
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
