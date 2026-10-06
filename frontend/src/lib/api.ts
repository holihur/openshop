/**
 * Typed API client for the OpenShop backend.
 *
 * Access tokens are short-lived JWTs; refresh tokens are opaque and rotated by
 * the server. On a 401 the client transparently refreshes once and retries, so
 * components never deal with token expiry. All requests are stateless, which is
 * what allows the backend to run behind any load balancer.
 */

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api/v1";

const ACCESS_KEY = "openshop.accessToken";
const REFRESH_KEY = "openshop.refreshToken";
const GUEST_KEY = "openshop.guestId";

// guestId returns a stable per-browser id so anonymous shoppers get a cart
// without signing in. It is sent as X-Guest-Id on every request.
export function guestId(): string {
  let id = localStorage.getItem(GUEST_KEY);
  if (!id) {
    id = typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID()
      : Math.random().toString(36).slice(2) + Date.now().toString(36);
    localStorage.setItem(GUEST_KEY, id);
  }
  return id;
}

export interface ApiErrorShape {
  code: string;
  message: string;
  status: number;
}

export class ApiError extends Error implements ApiErrorShape {
  code: string;
  status: number;

  constructor({ code, message, status }: ApiErrorShape) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

export const tokenStore = {
  access: () => localStorage.getItem(ACCESS_KEY),
  refresh: () => localStorage.getItem(REFRESH_KEY),
  set(access: string, refresh: string) {
    localStorage.setItem(ACCESS_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

interface Envelope<T> {
  data?: T;
  meta?: { total: number; page: number; pageSize: number };
  error?: { code: string; message: string };
}

export interface Page<T> {
  items: T;
  total: number;
  page: number;
  pageSize: number;
}

let refreshPromise: Promise<boolean> | null = null;

/** Refresh the access token, de-duplicating concurrent refresh attempts. */
async function refreshAccessToken(): Promise<boolean> {
  const refresh = tokenStore.refresh();
  if (!refresh) return false;

  if (!refreshPromise) {
    refreshPromise = (async () => {
      try {
        const res = await fetch(`${API_BASE}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refreshToken: refresh }),
        });
        if (!res.ok) {
          tokenStore.clear();
          return false;
        }
        const body = (await res.json()) as Envelope<{
          accessToken: string;
          refreshToken: string;
        }>;
        if (!body.data) return false;
        tokenStore.set(body.data.accessToken, body.data.refreshToken);
        return true;
      } catch {
        return false;
      } finally {
        refreshPromise = null;
      }
    })();
  }
  return refreshPromise;
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  auth?: boolean;
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

async function parse<T>(res: Response): Promise<Envelope<T>> {
  const text = await res.text();
  if (!text) return {};
  try {
    return JSON.parse(text) as Envelope<T>;
  } catch {
    return {};
  }
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<Envelope<T>> {
  const { method = "GET", body, auth = true, headers = {}, signal } = options;

  const send = async (): Promise<Response> => {
    const finalHeaders: Record<string, string> = { ...headers };
    if (body !== undefined) finalHeaders["Content-Type"] = "application/json";
    const token = tokenStore.access();
    if (auth && token) finalHeaders["Authorization"] = `Bearer ${token}`;
    finalHeaders["X-Guest-Id"] = guestId();

    return fetch(`${API_BASE}${path}`, {
      method,
      headers: finalHeaders,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal,
    });
  };

  let res = await send();
  if (res.status === 401 && auth && tokenStore.refresh()) {
    const ok = await refreshAccessToken();
    if (ok) res = await send();
  }

  const envelope = await parse<T>(res);
  if (!res.ok) {
    throw new ApiError({
      code: envelope.error?.code ?? "error",
      message: envelope.error?.message ?? `Request failed with status ${res.status}`,
      status: res.status,
    });
  }
  return envelope;
}

export const api = {
  async get<T>(path: string, signal?: AbortSignal): Promise<T> {
    const { data } = await request<T>(path, { signal });
    return data as T;
  },
  async getPage<T>(path: string, signal?: AbortSignal): Promise<Page<T>> {
    const { data, meta } = await request<T>(path, { signal });
    return {
      items: data as T,
      total: meta?.total ?? 0,
      page: meta?.page ?? 1,
      pageSize: meta?.pageSize ?? 20,
    };
  },
  async post<T>(path: string, body?: unknown): Promise<T> {
    const { data } = await request<T>(path, { method: "POST", body });
    return data as T;
  },
  async patch<T>(path: string, body?: unknown): Promise<T> {
    const { data } = await request<T>(path, { method: "PATCH", body });
    return data as T;
  },
  async put<T>(path: string, body?: unknown): Promise<T> {
    const { data } = await request<T>(path, { method: "PUT", body });
    return data as T;
  },
  async del<T>(path: string, body?: unknown): Promise<T> {
    const { data } = await request<T>(path, { method: "DELETE", body });
    return data as T;
  },
  /** Public (unauthenticated) request, used by the catalog. */
  async publicGet<T>(path: string, signal?: AbortSignal): Promise<T> {
    const { data } = await request<T>(path, { auth: false, signal });
    return data as T;
  },
  /** Multipart upload (e.g. product images). */
  async upload<T>(path: string, file: File): Promise<T> {
    const form = new FormData();
    form.append("file", file);

    const send = () =>
      fetch(`${API_BASE}${path}`, {
        method: "POST",
        headers: tokenStore.access()
          ? { Authorization: `Bearer ${tokenStore.access()}` }
          : {},
        body: form,
      });

    let res = await send();
    if (res.status === 401 && tokenStore.refresh()) {
      const ok = await refreshAccessToken();
      if (ok) res = await send();
    }
    const envelope = await parse<T>(res);
    if (!res.ok) {
      throw new ApiError({
        code: envelope.error?.code ?? "error",
        message: envelope.error?.message ?? `Upload failed with status ${res.status}`,
        status: res.status,
      });
    }
    return envelope.data as T;
  },
};
