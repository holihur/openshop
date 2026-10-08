import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Pagination } from "@lib/components/pagination";
import { I18nProvider } from "@lib/i18n";

function renderPagination(props: { page: number; pageSize: number; total: number; onChange?: () => void }) {
  return render(
    <I18nProvider>
      <Pagination onChange={props.onChange ?? (() => {})} {...props} />
    </I18nProvider>,
  );
}

describe("Pagination", () => {
  it("renders nothing when there is a single page", () => {
    const { container } = renderPagination({ page: 1, pageSize: 20, total: 20 });
    expect(container).toBeEmptyDOMElement();
  });

  it("renders nothing for an empty result set", () => {
    const { container } = renderPagination({ page: 1, pageSize: 20, total: 0 });
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the current page out of the total", () => {
    renderPagination({ page: 2, pageSize: 10, total: 45 });
    // 45 items at 10 per page = 5 pages.
    expect(screen.getByText(/2/)).toBeInTheDocument();
    expect(screen.getByText(/5/)).toBeInTheDocument();
  });

  it("disables previous on the first page and next on the last", () => {
    const { unmount } = renderPagination({ page: 1, pageSize: 10, total: 30 });
    expect(screen.getByRole("button", { name: /previous/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /next/i })).toBeEnabled();
    unmount();

    renderPagination({ page: 3, pageSize: 10, total: 30 });
    expect(screen.getByRole("button", { name: /previous/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /next/i })).toBeDisabled();
  });

  it("reports the requested page", async () => {
    const onChange = vi.fn();
    renderPagination({ page: 2, pageSize: 10, total: 30, onChange });
    await userEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(onChange).toHaveBeenCalledWith(3);
    await userEvent.click(screen.getByRole("button", { name: /previous/i }));
    expect(onChange).toHaveBeenCalledWith(1);
  });

  // A page beyond the end (e.g. after deleting the last row) must not offer a
  // next page.
  it("clamps a page beyond the end", () => {
    renderPagination({ page: 9, pageSize: 10, total: 30 });
    expect(screen.getByRole("button", { name: /next/i })).toBeDisabled();
  });
});
