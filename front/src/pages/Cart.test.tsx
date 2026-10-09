import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { I18nProvider } from "@lib/i18n";
import { t } from "@lib/i18n/translate";

// --- test doubles ---------------------------------------------------------

const post = vi.fn();
const navigate = vi.fn();
let auth: { user: unknown } = { user: null };

vi.mock("@lib/api", () => ({
  api: { post: (...args: unknown[]) => post(...args) },
  ApiError: class ApiError extends Error {},
}));

vi.mock("@lib/auth", () => ({ useAuth: () => auth }));

vi.mock("@lib/currency", () => ({
  useCurrency: () => ({ currency: "CNY", convert: (c: number) => c }),
  CurrencyProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return { ...actual, useNavigate: () => navigate };
});

const cartData = {
  items: [
    { productId: "p1", variantId: "", title: "Voyage Backpack", priceCents: 129900, quantity: 1, subtotal: 129900 },
  ],
  totalCents: 129900,
  totalCount: 1,
};

vi.mock("@lib/hooks/useCart", () => ({
  useCart: () => ({ data: cartData, isLoading: false }),
  useUpdateCartItem: () => ({ mutate: vi.fn(), isPending: false }),
  useRemoveCartItem: () => ({ mutate: vi.fn(), isPending: false }),
  useClearCart: () => ({ mutate: vi.fn() }),
}));

vi.mock("@lib/hooks/useAddresses", () => ({
  useAddresses: () => ({
    data: [{ id: "a1", recipient: "Ada", province: "GD", city: "SZ", district: "", line1: "1 Main St", default: true }],
  }),
}));

vi.mock("@lib/hooks/useShipping", () => ({
  useShippingMethods: () => ({
    data: [
      { id: "s1", name: "Standard", flatRateCents: 1000, freeThresholdCents: 0 },
      { id: "s2", name: "Express", flatRateCents: 3000, freeThresholdCents: 0 },
    ],
  }),
  useDeliveryEstimates: () => ({
    data: [
      {
        methodId: "s1",
        code: "standard",
        name: "Standard",
        priceCents: 1000,
        minDays: 3,
        maxDays: 5,
        earliest: "2026-03-10T00:00:00Z",
        latest: "2026-03-12T00:00:00Z",
        freeThresholdCents: 0,
        freeRemainingCents: 0,
      },
      {
        methodId: "s2",
        code: "express",
        name: "Express",
        priceCents: 3000,
        minDays: 1,
        maxDays: 2,
        earliest: "2026-03-09T00:00:00Z",
        latest: "2026-03-10T00:00:00Z",
        freeThresholdCents: 0,
        freeRemainingCents: 0,
      },
    ],
  }),
}));

vi.mock("@lib/hooks/usePayment", () => ({
  usePaymentMethods: () => ({ data: ["mock", "offline"] }),
}));

vi.mock("@lib/hooks/useLoyalty", () => ({
  useWallet: () => ({ data: { currency: "CNY", balanceCents: 5000 } }),
  usePoints: () => ({ data: { balance: 200, lifetimeEarned: 200 } }),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const { CartPage } = await import("@/pages/Cart");

function renderCart() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <I18nProvider>
          <CartPage />
        </I18nProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function orderPayload(): Record<string, unknown> {
  const call = post.mock.calls.find((c) => c[0] === "/orders");
  return (call?.[1] ?? {}) as Record<string, unknown>;
}

beforeEach(() => {
  post.mockReset();
  navigate.mockReset();
  auth = { user: null };
  post.mockImplementation(async (path: string) => {
    if (path === "/orders") return { id: "o1", orderNo: "OS-1", status: "pending_payment", totalCents: 130900 };
    if (path === "/payments") return { id: "pay1", status: "pending" };
    return {};
  });
});

describe("CartPage", () => {
  it("renders the items and the order summary", () => {
    renderCart();
    expect(screen.getByText("Voyage Backpack")).toBeInTheDocument();
    expect(screen.getByText(t("cart.orderSummary"))).toBeInTheDocument();
  });

  it("defaults the payment method to the first provider", () => {
    renderCart();
    // The label is the localised channel name, not the provider key.
    expect(screen.getByRole("radio", { name: t("payment.method.mock") })).toBeChecked();
    expect(screen.getByRole("radio", { name: t("payment.method.offline") })).not.toBeChecked();
  });

  it("uses the channel the shopper picks", async () => {
    auth = { user: { id: "u1", email: "ada@example.com" } };
    renderCart();
    await userEvent.click(screen.getByRole("radio", { name: t("payment.method.offline") }));
    await userEvent.click(screen.getByRole("button", { name: new RegExp(t("cart.checkout"), "i") }));

    await waitFor(() => expect(post).toHaveBeenCalledWith("/payments", expect.anything()));
    const payment = post.mock.calls.find((c) => c[0] === "/payments")?.[1] as Record<string, unknown>;
    expect(payment.provider).toBe("offline");
  });

  // The signed-in checkout must send the resolved address, shipping method and
  // currency, then open a payment session with the selected provider.
  it("checks out with the selected address, shipping and provider", async () => {
    auth = { user: { id: "u1", email: "ada@example.com" } };
    renderCart();
    await userEvent.click(screen.getByRole("button", { name: new RegExp(t("cart.checkout"), "i") }));

    await waitFor(() => expect(post).toHaveBeenCalledWith("/orders", expect.anything()));
    const payload = orderPayload();
    expect(payload.addressId).toBe("a1");
    expect(payload.shippingMethodId).toBe("s1");
    expect(payload.currency).toBe("CNY");
    // No stored value unless the shopper opts in.
    expect(payload.useWallet).toBeUndefined();
    expect(payload.points).toBeUndefined();

    await waitFor(() => expect(post).toHaveBeenCalledWith("/payments", expect.anything()));
    const payment = post.mock.calls.find((c) => c[0] === "/payments")?.[1] as Record<string, unknown>;
    expect(payment.orderId).toBe("o1");
    expect(payment.provider).toBe("mock");
  });

  it("includes the wallet and points when the shopper opts in", async () => {
    auth = { user: { id: "u1", email: "ada@example.com" } };
    renderCart();

    await userEvent.click(screen.getByRole("checkbox", { name: new RegExp(t("cart.useWallet"), "i") }));
    await userEvent.type(screen.getByRole("spinbutton"), "50");
    await userEvent.click(screen.getByRole("button", { name: new RegExp(t("cart.checkout"), "i") }));

    await waitFor(() => expect(post).toHaveBeenCalledWith("/orders", expect.anything()));
    const payload = orderPayload();
    expect(payload.useWallet).toBe(true);
    expect(payload.points).toBe(50);
  });

  // A guest must supply enough to deliver and to reach them before ordering.
  it("keeps a guest from checking out until the delivery details are complete", async () => {
    renderCart();
    const button = screen.getByRole("button", { name: new RegExp(t("cart.checkout"), "i") });
    expect(button).toBeDisabled();

    await userEvent.type(screen.getByPlaceholderText(t("cart.emailForReceipt")), "guest@example.com");
    expect(button).toBeDisabled();

    await userEvent.type(screen.getByPlaceholderText(t("cart.recipientName")), "Guest");
    expect(button).toBeDisabled();

    await userEvent.type(screen.getByPlaceholderText(t("cart.addressLine")), "1 Main St");
    expect(button).toBeEnabled();
  });

  it("checks a guest out and keeps the access token for later", async () => {
    post.mockImplementation(async (path: string) => {
      if (path === "/orders") return { id: "o2", accessToken: "tok-1", status: "pending_payment" };
      if (path === "/guest/orders/tok-1/pay") return { id: "pay2", status: "pending" };
      return {};
    });
    renderCart();
    await userEvent.type(screen.getByPlaceholderText(t("cart.emailForReceipt")), "guest@example.com");
    await userEvent.type(screen.getByPlaceholderText(t("cart.recipientName")), "Guest");
    await userEvent.type(screen.getByPlaceholderText(t("cart.addressLine")), "1 Main St");
    await userEvent.click(screen.getByRole("button", { name: new RegExp(t("cart.checkout"), "i") }));

    await waitFor(() => expect(post).toHaveBeenCalledWith("/orders", expect.anything()));
    const payload = orderPayload();
    expect(payload.email).toBe("guest@example.com");
    expect(payload.address).toMatchObject({ recipient: "Guest", line1: "1 Main St" });
    expect(localStorage.getItem("openshop.guestOrderToken")).toBe("tok-1");
  });
});
