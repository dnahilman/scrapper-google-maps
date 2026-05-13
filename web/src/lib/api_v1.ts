// Typed client for the new Go backend (/api/v1).
// Kept separate from api.ts so legacy Python endpoints stay untouched
// while components migrate one by one.

const BASE = '/api/v1';

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
      const body = (await res.json()) as { error?: string; detail?: string };
      detail = body.error ?? body.detail ?? res.statusText;
    } catch {
      detail = res.statusText;
    }
    throw new Error(`${res.status}: ${detail}`);
  }
  if (res.status === 204) return null as T;
  return res.json() as Promise<T>;
}

// ---------- Domain types ----------

export interface City {
  id: string;
  emsifa_regency_id: string;
  emsifa_province_id: string;
  name: string;
  slug: string;
  province_name: string;
  created_at: string;
  updated_at: string;
}

export interface Kelurahan {
  id: string;
  city_id: string;
  emsifa_village_id: string;
  emsifa_district_id: string;
  name: string;
  kecamatan_name: string;
  code?: string;
  created_at: string;
}

export interface CityDetail {
  city: City;
  kelurahan_count: number;
}

export type WorkerStatus = 'online' | 'offline' | 'draining';

export interface Worker {
  id: string;
  name: string;
  hostname?: string;
  ip_addr?: string;
  max_concurrency: number;
  status: WorkerStatus;
  last_heartbeat?: string;
  registered_at: string;
}

export type JobStatusV1 =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled';

export interface JobV1 {
  id: string;
  city_id: string;
  city?: City;
  keyword: string;
  status: JobStatusV1;
  total_tasks: number;
  done_count: number;
  failed_count: number;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface JobDetail {
  job: JobV1;
  task_counts: Record<string, number>;
}

export type TaskStatusV1 =
  | 'queued'
  | 'in_progress'
  | 'done'
  | 'failed'
  | 'cancelled';

export interface TaskV1 {
  id: string;
  job_id: string;
  kelurahan_id: string;
  kelurahan?: Kelurahan;
  worker_id?: string;
  worker?: Worker;
  status: TaskStatusV1;
  attempt: number;
  max_attempts: number;
  last_error?: string;
  places_count?: number;
  enqueued_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface HealthV1 {
  status: string;
  db: string;
  uptime_seconds: number;
  workers_total: number;
  workers_online: number;
}

export interface CreateJobBody {
  city_id: string;
  keyword: string;
  kelurahan_names?: string[];
  kelurahan_ids?: string[];
  options?: Record<string, unknown>;
  max_attempts?: number;
}

// ---------- API surface ----------

export const v1 = {
  // Health
  health: (): Promise<HealthV1> => req('/health'),

  // Cities & kelurahan
  cities: (): Promise<City[]> => req('/cities'),
  city: (idOrSlug: string): Promise<CityDetail> =>
    req(`/cities/${encodeURIComponent(idOrSlug)}`),
  syncCities: (): Promise<{ synced: number }> =>
    req('/cities/sync', { method: 'POST' }),
  kelurahan: (cityIdOrSlug: string, search?: string): Promise<Kelurahan[]> => {
    const q = search ? `?search=${encodeURIComponent(search)}` : '';
    return req(`/cities/${encodeURIComponent(cityIdOrSlug)}/kelurahan${q}`);
  },
  syncKelurahan: (cityIdOrSlug: string): Promise<{ city: string; kelurahan: number }> =>
    req(`/cities/${encodeURIComponent(cityIdOrSlug)}/kelurahan/sync`, { method: 'POST' }),

  // Workers
  workers: (): Promise<Worker[]> => req('/workers'),
  drainWorker: (id: string): Promise<null> =>
    req(`/workers/${encodeURIComponent(id)}/drain`, { method: 'POST' }),
  deleteWorker: (id: string): Promise<null> =>
    req(`/workers/${encodeURIComponent(id)}`, { method: 'DELETE' }),

  // Jobs
  jobs: (status?: string): Promise<JobV1[]> => {
    const q = status ? `?status=${encodeURIComponent(status)}` : '';
    return req(`/jobs${q}`);
  },
  job: (id: string): Promise<JobDetail> => req(`/jobs/${encodeURIComponent(id)}`),
  createJob: (body: CreateJobBody): Promise<JobV1> =>
    req('/jobs', { method: 'POST', body: JSON.stringify(body) }),
  cancelJob: (id: string): Promise<null> =>
    req(`/jobs/${encodeURIComponent(id)}/cancel`, { method: 'POST' }),
  retryFailed: (id: string): Promise<{ requeued: number }> =>
    req(`/jobs/${encodeURIComponent(id)}/retry-failed`, { method: 'POST' }),

  // Tasks
  tasks: (params: { jobId?: string; status?: string; limit?: number } = {}): Promise<TaskV1[]> => {
    const q = new URLSearchParams();
    if (params.jobId) q.set('job_id', params.jobId);
    if (params.status) q.set('status', params.status);
    if (params.limit) q.set('limit', String(params.limit));
    const qs = q.toString();
    return req(`/tasks${qs ? `?${qs}` : ''}`);
  },
  resetTask: (id: string): Promise<null> =>
    req(`/tasks/${encodeURIComponent(id)}/reset`, { method: 'POST' }),
};

export function fmtTimeAgo(iso?: string | null): string {
  if (!iso) return '–';
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  const diff = Math.max(0, (Date.now() - t) / 1000);
  if (diff < 60) return `${Math.round(diff)}s ago`;
  if (diff < 3600) return `${Math.round(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.round(diff / 3600)}h ago`;
  return `${Math.round(diff / 86400)}d ago`;
}
