import { writable } from 'svelte/store';

export type Section = 'dashboard' | 'jobs' | 'workers' | 'files' | 'logs';
export type ToastKind = 'info' | 'success' | 'error' | 'warn';

export interface Toast {
  id: string;
  message: string;
  kind: ToastKind;
}

const VALID_SECTIONS: Section[] = ['dashboard', 'jobs', 'workers', 'files', 'logs'];

// Active section di sidebar. Sync ke hash supaya bookmarkable.
// Format hash: #dashboard, #jobs, #files, #logs (tanpa slash)
function readHash(): Section {
  const h = (window.location.hash || '#dashboard').replace(/^#/, '');
  const seg = h.split('/')[0] as Section;
  if (VALID_SECTIONS.includes(seg)) return seg;
  return 'dashboard';
}

export const section = writable<Section>(readHash());

section.subscribe((s) => {
  const target = `#${s}`;
  if (window.location.hash !== target) {
    window.history.replaceState(null, '', target);
  }
});

window.addEventListener('hashchange', () => {
  section.set(readHash());
});

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
