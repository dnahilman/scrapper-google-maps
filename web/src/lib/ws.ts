// WebSocket store: subscribes to /ws on the master and exposes a typed event stream.
// Phase 1: stub that auto-reconnects. Phase 4 will wire real-time events.

import { writable, type Readable } from 'svelte/store';

export type WSStatus = 'connecting' | 'open' | 'closed' | 'error';

export interface WSEvent {
  type: string;
  payload: unknown;
  at: string;
}

interface WSStore extends Readable<WSStatus> {
  events: Readable<WSEvent | null>;
  connect(): void;
  disconnect(): void;
  send(payload: unknown): void;
}

function buildWSURL(): string {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
  return `${proto}://${window.location.host}/ws`;
}

export function createWS(): WSStore {
  const status = writable<WSStatus>('closed');
  const events = writable<WSEvent | null>(null);

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
    } catch (err) {
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

  return {
    subscribe: status.subscribe,
    events: { subscribe: events.subscribe },
    connect,
    disconnect,
    send,
  };
}

export const ws = createWS();
