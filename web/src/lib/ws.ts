import { writable, type Readable } from 'svelte/store';

export type WSStatus = 'connecting' | 'open' | 'closed' | 'error';

export interface WSEvent {
  type: string;
  payload: unknown;
  at: string;
}

export interface LogEntry {
  level: string;
  time: string;
  message: string;
  fields?: Record<string, unknown>;
}

interface PlaceScrapedPayload {
  task_id?: string;
  place_id?: string;
  title?: string;
  kelurahan_name?: string;
  kecamatan_name?: string;
  city_name?: string;
  index?: number;
  total?: number;
}

interface WSStore extends Readable<WSStatus> {
  events: Readable<WSEvent | null>;
  logEvents: Readable<LogEntry[]>;
  connect(): void;
  disconnect(): void;
  send(payload: unknown): void;
  clearLogs(): void;
}

function buildWSURL(): string {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
  return `${proto}://${window.location.host}/ws`;
}

export function createWS(): WSStore {
  const status = writable<WSStatus>('closed');
  const events = writable<WSEvent | null>(null);
  const logEvents = writable<LogEntry[]>([]);

  let socket: WebSocket | null = null;
  let reconnectAttempts = 0;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let manualClose = false;

  function scheduleReconnect() {
    if (manualClose) return;
    const delay = Math.min(30_000, 1000 * Math.pow(2, reconnectAttempts++));
    reconnectTimer = setTimeout(connect, delay);
  }

  function connect() {
    manualClose = false;
    status.set('connecting');
    try {
      socket = new WebSocket(buildWSURL());
    } catch {
      status.set('error');
      scheduleReconnect();
      return;
    }

    socket.addEventListener('open', () => {
      reconnectAttempts = 0;
      status.set('open');
    });

    socket.addEventListener('message', (ev) => {
      try {
        const data = JSON.parse(ev.data) as WSEvent;
        events.set(data);
        if (data.type === 'log') {
          const entry = data.payload as LogEntry;
          logEvents.update((prev) => [...prev.slice(-2000), entry]);
        } else if (data.type === 'place.scraped') {
          const p = data.payload as PlaceScrapedPayload;
          const parts = [p.kelurahan_name, p.kecamatan_name].filter(Boolean).join(', ');
          logEvents.update((prev) => [
            ...prev.slice(-2000),
            {
              level: 'info',
              time: data.at,
              message: `✓ ${p.title ?? p.place_id ?? '?'}`,
              fields: {
                ...(parts ? { lokasi: parts } : {}),
                ...(p.index != null && p.total != null ? { progress: `${p.index}/${p.total}` } : {}),
              },
            },
          ]);
        }
      } catch {
        // Ignore malformed payloads.
      }
    });

    socket.addEventListener('close', () => {
      status.set('closed');
      scheduleReconnect();
    });

    socket.addEventListener('error', () => {
      status.set('error');
    });
  }

  function disconnect() {
    manualClose = true;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    if (socket) {
      socket.close();
      socket = null;
    }
    status.set('closed');
  }

  function send(payload: unknown) {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(payload));
    }
  }

  function clearLogs() {
    logEvents.set([]);
  }

  return {
    subscribe: status.subscribe,
    events: { subscribe: events.subscribe },
    logEvents: { subscribe: logEvents.subscribe },
    connect,
    disconnect,
    send,
    clearLogs,
  };
}

export const ws = createWS();
