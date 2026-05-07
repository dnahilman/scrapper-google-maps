// Thin fetch wrapper. Same-origin di production (FastAPI serve static),
// proxy ke localhost:8000 di vite dev mode (lihat vite.config.js).

const BASE = '/api';

interface ReqOptions extends RequestInit {
  headers?: Record<string, string>;
}

async function req<T = unknown>(path: string, options: ReqOptions = {}): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(options.headers ?? {}) },
    ...options,
  });
  if (!res.ok) {
    let detail: string;
    try {
      detail = ((await res.json()) as { detail: string }).detail;
    } catch {
      detail = res.statusText;
    }
    throw new Error(`${res.status}: ${detail}`);
  }
  if (res.status === 204) return null as T;
  return res.json() as Promise<T>;
}

// ---- API response types ----

export interface HealthResponse {
  ok: boolean;
  version: string;
  uptime_sec: number;
  active_jobs: number;
  keywords: string[];
  data_dir_size_mb: number;
}

export interface JobResponse {
  job_id: string;
  keyword: string;
  shard: string | null;
  kelurahan: string | null;
  limit: number | null;
  pid: number;
  status: 'running' | 'exited' | 'failed' | 'done';
  started_at: string;
  resume: boolean;
  dry_run: boolean;
}

export interface StartJobBody {
  keyword: string;
  shard: string | null;
  kelurahan: string | null;
  limit: number | null;
  resume: boolean;
  dry_run: boolean;
}

export interface FileInfo {
  name: string;
  size_bytes: number;
  modified: string | number;
  shop_count?: number | null;
}

export interface LogFile {
  name: string;
  kind: string;
  size_bytes: number;
  modified: string | number;
}

export interface ProgressCounts {
  done: number;
  in_progress: number;
  failed: number;
}

export interface ProgressResponse {
  counts: ProgressCounts;
  total: number;
}

// ---- API object ----

export const api = {
  // health
  health: (): Promise<HealthResponse> => req('/health'),

  // keywords + files
  keywords: (): Promise<string[]> => req('/keywords'),
  files: (kw: string): Promise<FileInfo[]> => req(`/keywords/${encodeURIComponent(kw)}/files`),
  file: (kw: string, name: string): Promise<unknown> =>
    req(`/keywords/${encodeURIComponent(kw)}/files/${encodeURIComponent(name)}`),
  fileDownloadUrl: (kw: string, name: string): string =>
    `${BASE}/keywords/${encodeURIComponent(kw)}/files/${encodeURIComponent(name)}/download`,

  // jobs
  listJobs: (kw?: string | null): Promise<JobResponse[]> =>
    req(`/jobs${kw ? `?keyword=${encodeURIComponent(kw)}` : ''}`),
  getJob: (id: string): Promise<JobResponse> => req(`/jobs/${encodeURIComponent(id)}`),
  startJob: (body: StartJobBody): Promise<JobResponse> =>
    req('/jobs', { method: 'POST', body: JSON.stringify(body) }),
  stopJob: (id: string): Promise<null> =>
    req(`/jobs/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  // logs
  listLogFiles: (): Promise<LogFile[]> => req('/logs/files'),
  jobLogStreamUrl: (id: string, seed = 200): string =>
    `${BASE}/jobs/${encodeURIComponent(id)}/logs/stream?seed=${seed}`,
  fileLogStreamUrl: (name: string, seed = 200): string =>
    `${BASE}/logs/${encodeURIComponent(name)}/stream?seed=${seed}`,

  // progress
  progress: (kw: string): Promise<ProgressResponse> =>
    req(`/keywords/${encodeURIComponent(kw)}/progress`),
  failed: (kw: string): Promise<unknown> =>
    req(`/keywords/${encodeURIComponent(kw)}/progress/failed`),
  resetKelurahan: (kw: string, kel: string): Promise<unknown> =>
    req(`/keywords/${encodeURIComponent(kw)}/progress/${encodeURIComponent(kel)}/reset`, {
      method: 'POST',
    }),
};

// Connect ke SSE endpoint, callback per-line. Return EventSource (caller .close()).
export function streamLogs(
  url: string,
  onLine: (line: string) => void,
  onError?: (e: Event) => void
): EventSource {
  const es = new EventSource(url);
  es.onmessage = (e: MessageEvent<string>) => onLine(e.data);
  es.onerror = (e: Event) => {
    if (onError) onError(e);
  };
  return es;
}

export function fmtBytes(n: number | null | undefined): string {
  if (n == null) return '-';
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

export function fmtDate(iso: string | number | null | undefined): string {
  if (!iso) return '-';
  try {
    const d = typeof iso === 'number' ? new Date(iso * 1000) : new Date(iso);
    return d.toLocaleString('id-ID', { hour12: false });
  } catch {
    return String(iso);
  }
}

export function fmtUptime(s: number | null | undefined): string {
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
