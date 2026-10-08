import { describe, expect, it } from "vitest";

import { formatDate, formatMoney } from "@lib/format";

describe("formatMoney", () => {
  it("formats minor units as currency", () => {
    // Locale-dependent separators, so assert on the digits and the code.
    const formatted = formatMoney(123456, "USD");
    expect(formatted).toContain("1,234.56");
    expect(formatted).toMatch(/USD|\$/);
  });

  it("upper-cases the currency code", () => {
    expect(formatMoney(1000, "cny")).toBe(formatMoney(1000, "CNY"));
  });

  // A bad currency code used to white-screen the page; it must degrade instead.
  it("never throws on an invalid currency code", () => {
    expect(() => formatMoney(1000, "NOTACODE")).not.toThrow();
    expect(formatMoney(1000, "NOTACODE")).toContain("10.00");
  });

  it("handles an empty or missing currency", () => {
    expect(() => formatMoney(1000, "")).not.toThrow();
    expect(() => formatMoney(1000)).not.toThrow();
  });

  it("handles zero and negative amounts", () => {
    expect(formatMoney(0, "USD")).toContain("0.00");
    expect(formatMoney(-500, "USD")).toContain("5.00");
  });
});

describe("formatDate", () => {
  it("renders a placeholder for a missing or invalid value", () => {
    expect(formatDate()).toBe("—");
    expect(formatDate("")).toBe("—");
    expect(formatDate("not-a-date")).toBe("—");
  });

  it("formats a valid ISO timestamp", () => {
    const out = formatDate("2026-02-03T04:05:06Z");
    expect(out).not.toBe("—");
    expect(out).toMatch(/2026/);
  });
});
