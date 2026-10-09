/**
 * Server-side access to the Go API.
 *
 * This app is a server-rendered storefront: pages call the API on the server,
 * so the browser never sees an access token and the HTML arrives complete. The
 * API base is therefore a server-only variable, and every helper here is
 * careful to keep request-scoped cookies out of any shared cache.
 */

export const API_BASE = process.env.OPENSHOP_API_BASE ?? "http://localhost:8080";

export const ACCESS_COOKIE = "ssr_access";
export const REFRESH_COOKIE = "ssr_refresh";
export const GUEST_COOKIE = "ssr_guest";

export interface ApiErrorBody {
  code: string;
  message: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

interface Envelope<T> {
  data?: T;
  meta?: { total: number; page: number; pageSize: number; nextCursor?: string; fuzzy?: boolean };
  error?: ApiErrorBody;
}

export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  fuzzy?: boolean;
}

export interface RequestOptions {
  method?: "GET" | "POST" | "PATCH" | "PUT" | "DELETE";
  body?: unknown;
  /** Bearer token for the signed-in shopper. */
  token?: string;
  /** Guest cart owner, sent as X-Guest-Id. */
  guestId?: string;
  /** Seconds to cache the response in the Next data cache. 0 disables caching. */
  revalidate?: number;
  tags?: string[];
  acceptLanguage?: string;
}

interface RawResponse<T> {
  data: T;
  meta: Envelope<T>["meta"];
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<RawResponse<T>> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (options.body !== undefined) headers["Content-Type"] = "application/json";
  if (options.token) headers.Authorization = `Bearer ${options.token}`;
  if (options.guestId) headers["X-Guest-Id"] = options.guestId;
  if (options.acceptLanguage) headers["Accept-Language"] = options.acceptLanguage;

  const init: RequestInit & { next?: { revalidate?: number; tags?: string[] } } = {
    method: options.method ?? "GET",
    headers,
    cache: "no-store",
  };
  if (options.body !== undefined) init.body = JSON.stringify(options.body);
  if (options.revalidate !== undefined && options.revalidate > 0) {
    init.cache = undefined;
    init.next = { revalidate: options.revalidate, tags: options.tags };
  }

  const res = await fetch(`${API_BASE}/api/v1${path}`, init);
  const text = await res.text();
  let payload: Envelope<T> = {};
  try {
    payload = text ? (JSON.parse(text) as Envelope<T>) : {};
  } catch {
    throw new ApiError(res.status, "invalid_response", text.slice(0, 200) || "empty response");
  }
  if (!res.ok || payload.error) {
    const body = payload.error ?? { code: "request_failed", message: res.statusText };
    throw new ApiError(res.status, body.code, body.message);
  }
  return { data: payload.data as T, meta: payload.meta };
}

/** Fetches a single object. */
export async function apiGet<T>(path: string, options: RequestOptions = {}): Promise<T> {
  return (await request<T>(path, options)).data;
}

/** Fetches a paginated list with its metadata. */
export async function apiList<T>(path: string, options: RequestOptions = {}): Promise<Page<T>> {
  const { data, meta } = await request<T[]>(path, options);
  return {
    items: data ?? [],
    total: meta?.total ?? 0,
    page: meta?.page ?? 1,
    pageSize: meta?.pageSize ?? 20,
    fuzzy: meta?.fuzzy,
  };
}

/** Issues a mutating call (used from client islands and route handlers). */
export async function apiSend<T>(
  path: string,
  options: RequestOptions & { method: NonNullable<RequestOptions["method"]> },
): Promise<T> {
  return (await request<T>(path, options)).data;
}
