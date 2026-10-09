import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { I18nProvider } from "@lib/i18n";
import type { DeliveryEstimate } from "@lib/types";

import { DeliveryEstimateLine, FreeShippingProgress } from "./delivery-estimate";

function renderUi(node: React.ReactNode) {
  return render(<I18nProvider>{node}</I18nProvider>);
}

function estimate(over: Partial<DeliveryEstimate> = {}): DeliveryEstimate {
  return {
    methodId: "m1",
    code: "standard",
    name: "Standard",
    priceCents: 800,
    minDays: 3,
    maxDays: 5,
    earliest: "2026-03-10T09:00:00Z",
    latest: "2026-03-12T09:00:00Z",
    freeThresholdCents: 9900,
    freeRemainingCents: 4900,
    ...over,
  };
}

const format = (cents: number) => `¥${(cents / 100).toFixed(2)}`;

describe("FreeShippingProgress", () => {
  it("shows how much more is needed and how far along the shopper is", () => {
    renderUi(
      <FreeShippingProgress
        subtotalCents={5000}
        thresholdCents={9900}
        remainingCents={4900}
        format={format}
      />,
    );
    expect(screen.getByRole("status")).toHaveTextContent("¥49.00");
    // 5000 of 9900 is 51%, so the bar is filled correctly and progressively.
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "51");
  });

  it("celebrates when the threshold is reached", () => {
    renderUi(
      <FreeShippingProgress
        subtotalCents={12000}
        thresholdCents={9900}
        remainingCents={0}
        format={format}
      />,
    );
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });

  it("renders nothing when no threshold is configured", () => {
    const { container } = renderUi(
      <FreeShippingProgress
        subtotalCents={5000}
        thresholdCents={0}
        remainingCents={0}
        format={format}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});

describe("DeliveryEstimateLine", () => {
  it("states the arrival window", () => {
    renderUi(<DeliveryEstimateLine estimate={estimate()} />);
    expect(screen.getByText(/Mar 10/)).toBeInTheDocument();
    expect(screen.getByText(/Mar 12/)).toBeInTheDocument();
  });

  it("falls back to business days when the dates are unusable", () => {
    renderUi(<DeliveryEstimateLine estimate={estimate({ earliest: "", latest: "" })} />);
    expect(screen.getByText("Arrives in 3–5 business days")).toBeInTheDocument();
  });

  it("renders nothing without an estimate", () => {
    const { container } = renderUi(<DeliveryEstimateLine />);
    expect(container).toBeEmptyDOMElement();
  });
});
