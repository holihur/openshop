import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { I18nProvider } from "@lib/i18n";
import type { Product } from "@lib/types";

const addToCart = vi.fn();
const addState = { isPending: false, mutate: addToCart };

vi.mock("@lib/hooks/useCart", () => ({
  useAddToCart: () => addState,
}));
vi.mock("@lib/hooks/usePrice", () => ({
  usePrice: () => (cents: number) => `¥${(cents / 100).toFixed(2)}`,
}));
vi.mock("@lib/components/wishlist-button", () => ({
  WishlistButton: () => <button type="button">wishlist</button>,
}));

const { ProductCard } = await import("@lib/components/product-card");

function product(over: Partial<Product> = {}): Product {
  return {
    id: "p1",
    title: "Voyage Backpack",
    description: "A sturdy backpack",
    priceCents: 129900,
    stock: 10,
    status: "published",
    ...over,
  } as Product;
}

function renderCard(p: Product) {
  return render(
    <MemoryRouter>
      <I18nProvider>
        <ProductCard product={p} />
      </I18nProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  addToCart.mockReset();
});

describe("ProductCard", () => {
  it("shows the title, price and a link to the product", () => {
    renderCard(product());
    expect(screen.getByText("Voyage Backpack")).toBeInTheDocument();
    expect(screen.getByText("¥1299.00")).toBeInTheDocument();
    const links = screen.getAllByRole("link");
    expect(links.some((l) => l.getAttribute("href") === "/products/p1")).toBe(true);
  });

  it("adds the product to the cart", async () => {
    renderCard(product());
    await userEvent.click(screen.getByRole("button", { name: /add/i }));
    expect(addToCart).toHaveBeenCalledWith({ productId: "p1" });
  });

  it("disables the button and flags an out-of-stock product", async () => {
    renderCard(product({ stock: 0 }));
    const button = screen.getByRole("button", { name: /add/i });
    expect(button).toBeDisabled();
    await userEvent.click(button);
    expect(addToCart).not.toHaveBeenCalled();
    // A badge explains why it cannot be bought.
    expect(screen.getByText(/out of stock/i)).toBeInTheDocument();
  });

  it("warns when stock is low but still allows buying", () => {
    renderCard(product({ stock: 3 }));
    expect(screen.getByText(/3/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /add/i })).toBeEnabled();
  });

  it("renders a placeholder when there is no cover image", () => {
    const { container } = renderCard(product({ coverImage: undefined }));
    expect(container.querySelector("img")).toBeNull();
  });

  it("adds a responsive srcset for backend-served images", () => {
    const { container } = renderCard(product({ coverImage: "/uploads/a.jpg" }));
    const img = container.querySelector("img");
    expect(img?.getAttribute("srcset")).toContain("/uploads/a.jpg?w=320 320w");
    expect(img?.getAttribute("alt")).toBe("Voyage Backpack");
  });

  it("does not build a srcset for an external image", () => {
    const { container } = renderCard(product({ coverImage: "https://cdn.example.com/a.jpg" }));
    expect(container.querySelector("img")?.getAttribute("srcset")).toBeNull();
  });

  it("shows the rating when there are reviews, otherwise the description", () => {
    const { unmount } = renderCard(product({ reviewCount: 2, rating: 4.5 }));
    expect(screen.getByText("4.5")).toBeInTheDocument();
    expect(screen.queryByText("A sturdy backpack")).not.toBeInTheDocument();
    unmount();

    renderCard(product({ reviewCount: 0 }));
    expect(screen.getByText("A sturdy backpack")).toBeInTheDocument();
  });
});
