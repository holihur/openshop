import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

/** A fresh module per test, so the refresh de-duplication state is isolated. */
async function loadApi() {
  vi.resetModules();
  return import("@lib/api");
}

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("tokenStore", () => {
  it("stores and clears both tokens", async () => {
    const { tokenStore } = await loadApi();
    tokenStore.set("access-1", "refresh-1");
    expect(tokenStore.access()).toBe("access-1");
    expect(tokenStore.refresh()).toBe("refresh-1");
    tokenStore.clear();
    expect(tokenStore.access()).toBeNull();
    expect(tokenStore.refresh()).toBeNull();
  });
});

describe("guestId", () => {
  it("is stable across calls and persisted", async () => {
    const { guestId } = await loadApi();
    const first = guestId();
    expect(first).toBeTruthy();
    expect(guestId()).toBe(first);
    expect(localStorage.getItem("openshop.guestId")).toBe(first);
  });
});

describe("api requests", () => {
  it("attaches the bearer token, guest id and locale", async () => {
    const { api, tokenStore, guestId } = await loadApi();
    tokenStore.set("access-1", "refresh-1");
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => json({ data: { id: "o1" } }));
    vi.stubGlobal("fetch", fetchMock);

    await api.get("/orders");

    const [, init] = fetchMock.mock.calls[0];
    const headers = init?.headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer access-1");
    expect(headers["X-Guest-Id"]).toBe(guestId());
    expect(headers["Accept-Language"]).toBeTruthy();
  });

  it("omits the bearer token for a public request", async () => {
    const { api, tokenStore } = await loadApi();
    tokenStore.set("access-1", "refresh-1");
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => json({ data: [] }));
    vi.stubGlobal("fetch", fetchMock);

    await api.publicGet("/products");

    const [, init] = fetchMock.mock.calls[0];
    const headers = init?.headers as Record<string, string>;
    expect(headers.Authorization).toBeUndefined();
  });

  it("unwraps the envelope and the page metadata", async () => {
    const { api } = await loadApi();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => json({ data: [{ id: "a" }], meta: { total: 42, page: 2, pageSize: 10 } })),
    );
    const page = await api.getPage<{ id: string }[]>("/orders");
    expect(page.items).toEqual([{ id: "a" }]);
    expect(page.total).toBe(42);
    expect(page.page).toBe(2);
    expect(page.pageSize).toBe(10);
  });

  it("defaults missing page metadata", async () => {
    const { api } = await loadApi();
    vi.stubGlobal("fetch", vi.fn(async () => json({ data: [] })));
    const page = await api.getPage<unknown[]>("/orders");
    expect(page.total).toBe(0);
    expect(page.page).toBe(1);
    expect(page.pageSize).toBe(20);
  });

  it("throws an ApiError carrying the server code", async () => {
    const { api, ApiError } = await loadApi();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => json({ error: { code: "insufficient_stock", message: "out of stock" } }, 409)),
    );
    await expect(api.get("/orders")).rejects.toBeInstanceOf(ApiError);
    await api.get("/orders").catch((err: InstanceType<typeof ApiError>) => {
      expect(err.code).toBe("insufficient_stock");
      expect(err.status).toBe(409);
    });
  });

  it("falls back to a status-based message when the body is not an envelope", async () => {
    const { api } = await loadApi();
    vi.stubGlobal("fetch", vi.fn(async () => new Response("<html>oops</html>", { status: 500 })));
    await api.get("/orders").catch((err: { code: string; message: string }) => {
      expect(err.code).toBe("error");
      expect(err.message).toContain("500");
    });
  });
});

// A 401 must be retried once with a fresh token, and concurrent 401s must share
// a single refresh so a burst of requests cannot stampede the token endpoint.
describe("token refresh on 401", () => {
  it("refreshes once and retries with the new token", async () => {
    const { api, tokenStore } = await loadApi();
    tokenStore.set("stale-access", "refresh-1");

    let resourceCalls = 0;
    let refreshCalls = 0;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("/auth/refresh")) {
        refreshCalls++;
        return json({ data: { accessToken: "fresh-access", refreshToken: "refresh-2" } });
      }
      resourceCalls++;
      if (resourceCalls === 1) return new Response("", { status: 401 });
      const headers = init?.headers as Record<string, string>;
      return json({ data: { authorization: headers.Authorization } });
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await api.get<{ authorization: string }>("/orders");
    expect(refreshCalls).toBe(1);
    expect(resourceCalls).toBe(2);
    expect(result.authorization).toBe("Bearer fresh-access");
    expect(tokenStore.access()).toBe("fresh-access");
    expect(tokenStore.refresh()).toBe("refresh-2");
  });

  it("de-duplicates concurrent refreshes", async () => {
    const { api, tokenStore } = await loadApi();
    tokenStore.set("stale-access", "refresh-1");

    let refreshCalls = 0;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("/auth/refresh")) {
        refreshCalls++;
        return json({ data: { accessToken: "fresh-access", refreshToken: "refresh-2" } });
      }
      const headers = init?.headers as Record<string, string>;
      // Every first attempt is rejected; the retry (with the fresh token) passes.
      if (headers.Authorization === "Bearer stale-access") return new Response("", { status: 401 });
      return json({ data: { ok: true } });
    });
    vi.stubGlobal("fetch", fetchMock);

    await Promise.all([api.get("/orders"), api.get("/orders"), api.get("/orders")]);
    expect(refreshCalls).toBe(1);
  });

  it("clears the session when the refresh is rejected", async () => {
    const { api, tokenStore } = await loadApi();
    tokenStore.set("stale-access", "refresh-1");
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) =>
        String(input).endsWith("/auth/refresh") ? new Response("", { status: 401 }) : new Response("", { status: 401 }),
      ),
    );

    await expect(api.get("/orders")).rejects.toBeTruthy();
    expect(tokenStore.access()).toBeNull();
    expect(tokenStore.refresh()).toBeNull();
  });

  it("does not attempt a refresh without a refresh token", async () => {
    const { api } = await loadApi();
    const fetchMock = vi.fn(async () => new Response("", { status: 401 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(api.get("/orders")).rejects.toBeTruthy();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
