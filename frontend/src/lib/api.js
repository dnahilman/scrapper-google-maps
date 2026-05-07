// Thin fetch wrapper. Same-origin di production (FastAPI serve static),
// proxy ke localhost:8000 di vite dev mode (lihat vite.config.js).

const BASE = '/api';

async function req(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    ...options,
  });
  if (!res.ok) {
    let detail;
    try { detail = (await res.json()).detail; } catch { detail = res.statusText; }
    throw new Error(`${res.status}: ${detail}`);
  }
  if (res.status === 204) return null;
  return res.json();
}

export const api = {
  // health
  health: () => req('/health'),

  // keywords + files
  keywords: () => req('/keywords'),
  files: (kw) => req(`/keywords/${encodeURIComponent(kw)}/files`),
  file: (kw, name) => req(`/keywords/${encodeURIComponent(kw)}/files/${encodeURIComponent(name)}`),
  fileDownloadUrl: (kw, name) =>
    `${BASE}/keywords/${encodeURIComponent(kw)}/files/${encodeURIComponent(name)}/download`,

  // jobs
  listJobs: (kw) => req(`/jobs${kw ? `?keyword=${encodeURIComponent(kw)}` : ''}`),
  getJob: (id) => req(`/jobs/${encodeURIComponent(id)}`),
  startJob: (body) => req('/jobs', { method: 'POST', body: JSON.stringify(body) }),
  stopJob: (id) => req(`/jobs/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  // logs
  listLogFiles: () => req('/logs/files'),
  jobLogStreamUrl: (id, seed = 200) =>
    `${BASE}/jobs/${encodeURIComponent(id)}/logs/stream?seed=${seed}`,
  fileLogStreamUrl: (name, seed = 200) =>
    `${BASE}/logs/${encodeURIComponent(name)}/stream?seed=${seed}`,

  // progress
  progress: (kw) => req(`/keywords/${encodeURIComponent(kw)}/progress`),
  failed: (kw) => req(`/keywords/${encodeURIComponent(kw)}/progress/failed`),
  resetKelurahan: (kw, kel) =>
    req(`/keywords/${encodeURIComponent(kw)}/progress/${encodeURIComponent(kel)}/reset`, {
      method: 'POST',
    }),
};

// Connect ke SSE endpoint, callback per-line. Return EventSource (caller .close()).
export function streamLogs(url, onLine, onError) {
  const es = new EventSource(url);
  es.onmessage = (e) => onLine(e.data);
  es.onerror = (e) => {
    if (onError) onError(e);
  };
  return es;
}

export function fmtBytes(n) {
  if (n == null) return '-';
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

export function fmtDate(iso) {
  if (!iso) return '-';
  try {
    const d = typeof iso === 'number' ? new Date(iso * 1000) : new Date(iso);
    return d.toLocaleString('id-ID', { hour12: false });
  } catch {
    return String(iso);
  }
}

export function fmtUptime(s) {
  if (s == null) return '-';
  const sec = Math.floor(s % 60);
  const min = Math.floor((s / 60) % 60);
  const hr = Math.floor((s / 3600) % 24);
  const d = Math.floor(s / 86400);
  if (d) return `${d}d ${hr}h`;
  if (hr) return `${hr}h ${min}m`;
  if (min) return `${min}m ${sec}s`;
  return `${sec}s`;
}
