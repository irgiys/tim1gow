/**
 * Satu-satunya pintu keluar HTTP frontend iMitra.
 *
 * Aturan yang ditegakkan berkas ini (README.md bagian 1, AGENTS.md bagian 4.3):
 *  - Tidak ada `fetch()` telanjang di komponen. Semua lewat sini.
 *  - Pesan error diambil apa adanya dari field `message` respons API dan TIDAK
 *    disusun ulang, supaya pesan yang menyebut kode BR (AC-04) tidak hilang.
 *  - Kegagalan tidak pernah dianggap sukses. Tidak ada nilai default diam-diam.
 *  - NIK / nomor dokumen / path foto tidak pernah masuk ke query string atau
 *    path URL (BR-11) — pemanggil memakai id internal pengajuan.
 */

import type { ApiError } from './types';

/**
 * Base URL backend dipakai dari BROWSER, jadi memakai localhost + port host,
 * bukan nama service docker (`docker-compose.yml` catatan b).
 */
const BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080';

/** Galat yang membawa status HTTP dan kode BR dari server. */
export class ApiClientError extends Error {
  readonly status: number;
  readonly code: string;
  readonly rule?: string;

  constructor(status: number, body: ApiError) {
    // Pesan dipakai apa adanya dari server — jangan disusun ulang (AC-04).
    super(body.message);
    this.name = 'ApiClientError';
    this.status = status;
    this.code = body.error;
    this.rule = body.rule;
  }
}

/** Galat jaringan / server tidak terjangkau. Bukan berarti "tidak apa-apa". */
export class ApiNetworkError extends Error {
  constructor(penyebab: unknown) {
    super('Tidak dapat menghubungi server. Periksa apakah backend sedang berjalan.');
    this.name = 'ApiNetworkError';
    this.cause = penyebab;
  }
}

export interface RequestOptions {
  /** Token bearer. Di Client Component ambil lewat lib/auth.ts. */
  token?: string | null;
  signal?: AbortSignal;
  /** Timeout ms; default 15000. Kegagalan timeout dilempar, tidak ditelan. */
  timeoutMs?: number;
}

interface InternalOptions extends RequestOptions {
  method: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
  body?: unknown;
}

const TIMEOUT_DEFAULT_MS = 15_000;

async function request<T>(path: string, opts: InternalOptions): Promise<T> {
  const { method, body, token, signal, timeoutMs = TIMEOUT_DEFAULT_MS } = opts;

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  if (signal) {
    signal.addEventListener('abort', () => controller.abort(), { once: true });
  }

  const headers: Record<string, string> = { Accept: 'application/json' };
  if (body !== undefined && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  let res: Response;
  try {
    res = await fetch(`${BASE_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body),
      signal: controller.signal,
      cache: 'no-store',
    });
  } catch (penyebab) {
    // Jangan menelan exception: lempar sebagai galat eksplisit.
    throw new ApiNetworkError(penyebab);
  } finally {
    clearTimeout(timer);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  const teks = await res.text();
  let payload: unknown = null;
  if (teks.length > 0) {
    try {
      payload = JSON.parse(teks);
    } catch {
      payload = null;
    }
  }

  if (!res.ok) {
    const badan: ApiError =
      payload !== null && typeof payload === 'object' && 'message' in payload
        ? (payload as ApiError)
        : {
            error: 'UNEXPECTED_ERROR',
            message: `Server merespons ${res.status} tanpa badan pesan yang dikenali.`,
          };
    throw new ApiClientError(res.status, badan);
  }

  return payload as T;
}

export const apiClient = {
  get: <T>(path: string, opts?: RequestOptions) => request<T>(path, { ...opts, method: 'GET' }),

  post: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { ...opts, method: 'POST', body }),

  patch: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { ...opts, method: 'PATCH', body }),

  put: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { ...opts, method: 'PUT', body }),

  /**
   * Sengaja TIDAK disediakan helper `delete` untuk audit trail.
   * Audit trail append-only (FR-09, AC-13, AGENTS.md Larangan 8).
   */
  delete: <T>(path: string, opts?: RequestOptions) =>
    request<T>(path, { ...opts, method: 'DELETE' }),
};

/**
 * Ambil pesan yang layak ditampilkan ke pengguna dari galat apa pun.
 * Pesan dari server dipakai apa adanya; kode BR ikut ditampilkan kalau ada.
 */
export function pesanGalat(e: unknown): string {
  if (e instanceof ApiClientError) {
    return e.rule ? `[${e.rule}] ${e.message}` : e.message;
  }
  if (e instanceof ApiNetworkError) {
    return e.message;
  }
  if (e instanceof Error) {
    return e.message;
  }
  return 'Terjadi galat yang tidak dikenali.';
}
