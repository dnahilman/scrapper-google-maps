import { writable } from 'svelte/store';

export type Section = 'dashboard' | 'jobs' | 'workers' | 'places' | 'logs';
export type ToastKind = 'info' | 'success' | 'error' | 'warn';

export interface Toast {
  id: string;
  message: string;
  kind: ToastKind;
}

export interface Route {
  section: Section;
  sub?: string;
  params: Record<string, string>;
}

const VALID_SECTIONS: Section[] = ['dashboard', 'jobs', 'workers', 'places', 'logs'];

function parseHash(raw: string): Route {
  // Format: #section/sub?key=val&key2=val2  or  #section?key=val
  const withoutHash = (raw || '#dashboard').replace(/^#/, '');
  const [pathPart, queryPart] = withoutHash.split('?');
  const segments = pathPart.split('/');
  const sectionRaw = segments[0] as Section;
  const section: Section = VALID_SECTIONS.includes(sectionRaw) ? sectionRaw : 'dashboard';
  const sub = segments[1] || undefined;

  const params: Record<string, string> = {};
  if (queryPart) {
    for (const kv of queryPart.split('&')) {
      const [k, v] = kv.split('=');
      if (k) params[decodeURIComponent(k)] = decodeURIComponent(v ?? '');
    }
  }
  return { section, sub, params };
}

function routeToHash(r: Route): string {
  let h = `#${r.section}`;
  if (r.sub) h += `/${r.sub}`;
  const keys = Object.keys(r.params ?? {});
  if (keys.length) {
    h += '?' + keys.map((k) => `${encodeURIComponent(k)}=${encodeURIComponent(r.params[k])}`).join('&');
  }
  return h;
}

function readRoute(): Route {
  return parseHash(window.location.hash);
}

export const route = writable<Route>(readRoute());

// Keep the hash in sync with the store
route.subscribe((r) => {
  const target = routeToHash(r);
  if (window.location.hash !== target) {
    window.history.replaceState(null, '', target);
  }
});

window.addEventListener('hashchange', () => {
  route.set(readRoute());
});

// Convenience: set the active section (clears sub/params)
export function navigate(hash: string): void {
  window.location.hash = hash;
  route.set(parseHash(hash));
}

// Legacy alias so Sidebar and other components don't break during refactor
export const section = {
  subscribe: (run: (s: Section) => void) => route.subscribe((r) => run(r.section)),
  set: (s: Section) => route.update((r) => ({ ...r, section: s, sub: undefined, params: {} })),
};

// Toast / notif sederhana
export const toasts = writable<Toast[]>([]);

export function notify(message: string, kind: ToastKind = 'info', timeout = 3500): void {
  const id = Math.random().toString(36).slice(2);
  toasts.update((t) => [...t, { id, message, kind }]);
  setTimeout(() => {
    toasts.update((t) => t.filter((x) => x.id !== id));
  }, timeout);
}

// Active jobs counter — di-update oleh JobList polling
export const activeJobsCount = writable<number>(0);
