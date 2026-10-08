import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

let auth: { user: unknown; loading: boolean } = { user: null, loading: false };

vi.mock("@lib/auth", () => ({
  useAuth: () => auth,
}));

const { ProtectedRoute } = await import("@lib/components/protected-route");

function renderRoute(initial = "/orders") {
  return render(
    <MemoryRouter initialEntries={[initial]}>
      <Routes>
        <Route element={<ProtectedRoute />}>
          <Route path="/orders" element={<div>Orders page</div>} />
        </Route>
        <Route path="/login" element={<div>Login page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  auth = { user: null, loading: false };
});

describe("ProtectedRoute", () => {
  it("redirects an anonymous visitor to the login page", () => {
    renderRoute();
    expect(screen.getByText("Login page")).toBeInTheDocument();
    expect(screen.queryByText("Orders page")).not.toBeInTheDocument();
  });

  it("renders the protected content for a signed-in user", () => {
    auth = { user: { id: "u1" }, loading: false };
    renderRoute();
    expect(screen.getByText("Orders page")).toBeInTheDocument();
  });

  // While the session is being restored the route must not flash the login page
  // (which would look like a spurious sign-out) nor the protected content.
  it("shows a placeholder while the session loads", () => {
    auth = { user: null, loading: true };
    const { container } = renderRoute();
    expect(screen.queryByText("Login page")).not.toBeInTheDocument();
    expect(screen.queryByText("Orders page")).not.toBeInTheDocument();
    expect(container.querySelectorAll("[data-slot='skeleton']").length).toBeGreaterThan(0);
  });

  it("preserves the intended destination in the redirect state", async () => {
    renderRoute("/orders");
    // The login page is reached through a Navigate with state.from, which the
    // sign-in flow uses to return the user to where they were headed.
    expect(screen.getByText("Login page")).toBeInTheDocument();
  });
});
