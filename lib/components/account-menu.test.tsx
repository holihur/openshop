import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { I18nProvider, t } from "@lib/i18n";

const logout = vi.fn(async () => {});
const navigate = vi.fn();
let auth: { user: unknown; logout: typeof logout } = {
  user: { name: "Ada Lovelace", email: "ada@example.com" },
  logout,
};

vi.mock("@lib/auth", () => ({
  useAuth: () => auth,
}));

vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return { ...actual, useNavigate: () => navigate };
});

const { AccountMenu } = await import("@lib/components/account-menu");

function renderMenu() {
  return render(
    <MemoryRouter>
      <I18nProvider>
        <AccountMenu />
      </I18nProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  logout.mockClear();
  navigate.mockClear();
  auth = { user: { name: "Ada Lovelace", email: "ada@example.com" }, logout };
});

describe("AccountMenu", () => {
  it("offers a sign-in button when signed out", () => {
    auth = { user: null, logout };
    renderMenu();
    expect(screen.getByRole("link", { name: new RegExp(t("nav.signIn"), "i") })).toBeInTheDocument();
    expect(screen.queryByTestId("account-menu")).not.toBeInTheDocument();
  });

  it("shows the user's initial and name", () => {
    renderMenu();
    expect(screen.getByTestId("account-menu")).toHaveTextContent("A");
    expect(screen.getByTestId("account-menu")).toHaveTextContent("Ada Lovelace");
  });

  it("falls back to the email when there is no name", () => {
    auth = { user: { name: "", email: "noname@example.com" }, logout };
    renderMenu();
    expect(screen.getByTestId("account-menu")).toHaveTextContent("N");
    expect(screen.getByTestId("account-menu")).toHaveTextContent("noname@example.com");
  });

  it("is closed until opened, and exposes the menu role", async () => {
    renderMenu();
    const trigger = screen.getByTestId("account-menu");
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();

    await userEvent.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(screen.getByText("ada@example.com")).toBeInTheDocument();
  });

  // The menu is the only route to these areas, so a missing entry would make a
  // feature unreachable from the storefront.
  it("links to every customer area", async () => {
    renderMenu();
    await userEvent.click(screen.getByTestId("account-menu"));
    const expected = [
      "/orders",
      "/support",
      "/account/notifications",
      "/account/wallet",
      "/account/rewards",
      "/account/tokens",
      "/account/addresses",
      "/account/wishlist",
      "/account/settings",
    ];
    const hrefs = screen.getAllByRole("menuitem").map((el) => el.getAttribute("href"));
    for (const href of expected) {
      expect(hrefs, `missing link to ${href}`).toContain(href);
    }
  });

  it("closes when a link is chosen", async () => {
    renderMenu();
    await userEvent.click(screen.getByTestId("account-menu"));
    await userEvent.click(screen.getByRole("menuitem", { name: new RegExp(t("nav.orders"), "i") }));
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("signs out and returns home", async () => {
    renderMenu();
    await userEvent.click(screen.getByTestId("account-menu"));
    await userEvent.click(
      screen.getByRole("menuitem", { name: new RegExp(t("common.signOut"), "i") }),
    );
    expect(logout).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith("/");
  });

  it("closes on Escape", async () => {
    renderMenu();
    await userEvent.click(screen.getByTestId("account-menu"));
    expect(screen.getByRole("menu")).toBeInTheDocument();
    await userEvent.keyboard("{Escape}");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });
});
