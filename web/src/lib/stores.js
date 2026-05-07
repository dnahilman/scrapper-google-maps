import { writable, derived } from 'svelte/store';

// Active section di sidebar. Sync ke hash supaya bookmarkable.
function readHash() {
  const h = (window.location.hash || '#/dashboard').replace(/^#\//, '');
  const seg = h.split('/')[0];
  if (['dashboard', 'jobs', 'files', 'logs'].includes(seg)) return seg;
  return 'dashboard';
}

export const section = writable(readHash());

section.subscribe((s) => {
  const target = `#/${s}`;
  if (window.location.hash !== target) {
    window.history.replaceState(null, '', target);
  }
});

window.addEventListener('hashchange', () => {
  section.set(readHash());
});

// Toast / notif sederhana
export const toasts = writable([]);

export function notify(message, kind = 'info', timeout = 3500) {
  const id = Math.random().toString(36).slice(2);
  toasts.update((t) => [...t, { id, message, kind }]);
  setTimeout(() => {
    toasts.update((t) => t.filter((x) => x.id !== id));
  }, timeout);
}

// Active jobs counter — di-update oleh JobList polling
export const activeJobsCount = writable(0);

// Theme
function readTheme() {
  return localStorage.getItem('theme') || 'dark';
}

export const theme = writable(readTheme());

theme.subscribe((t) => {
  document.documentElement.setAttribute('data-theme', t);
  localStorage.setItem('theme', t);
});

export const isDark = derived(theme, ($t) => $t === 'dark');
