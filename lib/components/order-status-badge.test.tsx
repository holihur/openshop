import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OrderStatusBadge } from "@lib/components/order-status-badge";
import { I18nProvider } from "@lib/i18n";
import { t } from "@lib/i18n/translate";
import type { OrderStatus } from "@lib/types";

function renderBadge(status: OrderStatus) {
  return render(
    <I18nProvider>
      <OrderStatusBadge status={status} />
    </I18nProvider>,
  );
}

describe("OrderStatusBadge", () => {
  const statuses: OrderStatus[] = [
    "pending_payment",
    "paid",
    "cancelled",
    "shipped",
    "completed",
    "refunded",
  ];

  it("renders a localised label for every status", () => {
    for (const status of statuses) {
      const { unmount } = renderBadge(status);
      const label = t(`status.${status}`);
      expect(screen.getByText(label), `status ${status}`).toBeInTheDocument();
      // A missing translation would render the raw key.
      expect(screen.queryByText(`status.${status}`)).not.toBeInTheDocument();
      unmount();
    }
  });

  // The colour carries meaning (paid/completed are success, refunded is
  // destructive), so pin the mapping down.
  it("maps statuses to the intended colour", () => {
    const cases: [OrderStatus, string][] = [
      ["pending_payment", "warning"],
      ["paid", "success"],
      ["shipped", "default"],
      ["completed", "success"],
      ["refunded", "destructive"],
      ["cancelled", "secondary"],
    ];
    for (const [status, variant] of cases) {
      const { container, unmount } = renderBadge(status);
      const badge = container.querySelector("span");
      expect(badge?.className, `status ${status}`).toContain(
        variant === "default" ? "bg-primary" : variant === "secondary" ? "bg-secondary" : "",
      );
      unmount();
    }
  });
});
