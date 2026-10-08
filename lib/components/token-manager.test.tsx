import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { I18nProvider } from "@lib/i18n";
import type { ScopeInfo } from "@lib/types";

const scopes: ScopeInfo[] = [
  { scope: "*", group: "general", description: "Full access" },
  { scope: "orders:read", group: "orders", description: "Read orders" },
  { scope: "orders:write", group: "orders", description: "Create orders" },
  { scope: "products:write", group: "catalog", description: "Edit products" },
];

const createToken = vi.fn();
const revokeToken = vi.fn();
const createState = { isPending: false, mutate: createToken };
const revokeState = { isPending: false, mutate: revokeToken };
let authUser: { role: string; permissions: string[] } | null = {
  role: "admin",
  permissions: scopes.map((s) => s.scope),
};

vi.mock("@lib/hooks/useTokens", () => ({
  useTokenScopes: () => ({ data: scopes }),
  useTokens: () => ({ data: [] }),
  useCreateToken: () => createState,
  useRevokeToken: () => revokeState,
}));

vi.mock("@lib/auth", () => ({
  useAuth: () => ({ user: authUser }),
}));

const { TokenManager } = await import("@lib/components/token-manager");

function renderManager(realm: "front" | "ops" = "front") {
  return render(
    <I18nProvider>
      <TokenManager realm={realm} />
    </I18nProvider>,
  );
}

beforeEach(() => {
  createToken.mockReset();
  revokeToken.mockReset();
  authUser = { role: "admin", permissions: scopes.map((s) => s.scope) };
});

describe("TokenManager", () => {
  it("offers the grantable scopes", () => {
    renderManager();
    expect(screen.getByRole("checkbox", { name: /orders:read/ })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: /products:write/ })).toBeInTheDocument();
  });

  it("cannot create a token without a name and a scope", async () => {
    renderManager();
    const submit = screen.getByRole("button", { name: /create token/i });
    expect(submit).toBeDisabled();

    await userEvent.type(screen.getByLabelText(/name/i), "CI deploy");
    expect(submit).toBeDisabled(); // still no scope selected

    await userEvent.click(screen.getByRole("checkbox", { name: /orders:read/ }));
    expect(submit).toBeEnabled();
  });

  it("submits the selected scopes and parsed CIDRs", async () => {
    renderManager();
    await userEvent.type(screen.getByLabelText(/name/i), "CI deploy");
    await userEvent.click(screen.getByRole("checkbox", { name: /orders:read/ }));
    await userEvent.click(screen.getByRole("checkbox", { name: /orders:write/ }));
    await userEvent.type(screen.getByLabelText(/allowed ips/i), "10.0.0.0/8, 203.0.113.7");
    await userEvent.type(screen.getByLabelText(/expires in days/i), "30");
    await userEvent.click(screen.getByRole("button", { name: /create token/i }));

    expect(createToken).toHaveBeenCalledTimes(1);
    const [input] = createToken.mock.calls[0];
    expect(input.name).toBe("CI deploy");
    expect(input.scopes).toEqual(["orders:read", "orders:write"]);
    expect(input.cidrs).toEqual(["10.0.0.0/8", "203.0.113.7"]);
    expect(input.expiresInDays).toBe(30);
  });

  it("omits the expiry when none is given", async () => {
    renderManager();
    await userEvent.type(screen.getByLabelText(/name/i), "no expiry");
    await userEvent.click(screen.getByRole("checkbox", { name: /orders:read/ }));
    await userEvent.click(screen.getByRole("button", { name: /create token/i }));
    expect(createToken.mock.calls[0][0]).not.toHaveProperty("expiresInDays");
  });

  it("shows the secret once, then hides it on request", async () => {
    createToken.mockImplementation((_input, opts) => {
      opts?.onSuccess?.({ token: "osp_abc_secret", name: "CI deploy", scopes: [], cidrs: [] });
    });
    renderManager();
    await userEvent.type(screen.getByLabelText(/name/i), "CI deploy");
    await userEvent.click(screen.getByRole("checkbox", { name: /orders:read/ }));
    await userEvent.click(screen.getByRole("button", { name: /create token/i }));

    await waitFor(() => expect(screen.getByText("osp_abc_secret")).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: /close/i }));
    expect(screen.queryByText("osp_abc_secret")).not.toBeInTheDocument();
  });

  // An operations token must never offer more than the operator's own role, and
  // the wildcard scope is administrator-only.
  it("limits ops scopes to the caller's role", () => {
    authUser = { role: "support", permissions: ["orders:read", "orders:write"] };
    renderManager("ops");
    expect(screen.getByRole("checkbox", { name: /orders:read/ })).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: /products:write/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: /full access/i })).not.toBeInTheDocument();
  });

  it("offers the wildcard scope to an administrator", () => {
    authUser = { role: "admin", permissions: ["*"] };
    renderManager("ops");
    expect(screen.getByRole("checkbox", { name: /full access/i })).toBeInTheDocument();
  });

  it("does not restrict the storefront realm by role", () => {
    authUser = { role: "customer", permissions: [] };
    renderManager("front");
    expect(screen.getByRole("checkbox", { name: /orders:write/ })).toBeInTheDocument();
  });
});
