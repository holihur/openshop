import { describe, expect, it } from "vitest";

import { ApiError } from "@lib/api";
import { errorMessage } from "@lib/errors";
import { setCurrentLocale, t } from "@lib/i18n/translate";

describe("errorMessage", () => {
  it("maps a known API error code to a localised message", () => {
    setCurrentLocale("en");
    const err = new ApiError({ code: "insufficient_stock", message: "insufficient stock", status: 409 });
    expect(errorMessage(err)).toBe(t("errors.insufficient_stock"));
    // The localised text must not be the raw server string.
    expect(errorMessage(err)).not.toBe("insufficient stock");
  });

  it("localises in the active locale", () => {
    setCurrentLocale("zh");
    const err = new ApiError({ code: "forbidden", message: "permission denied", status: 403 });
    expect(errorMessage(err)).toBe(t("errors.forbidden"));
    setCurrentLocale("en");
  });

  // A code the client does not know must still show the server's message rather
  // than a generic one, so nothing is hidden from the user.
  it("falls back to the server message for an unmapped code", () => {
    const err = new ApiError({ code: "brand_new_code", message: "Something specific", status: 400 });
    expect(errorMessage(err)).toBe("Something specific");
  });

  it("uses a plain Error message", () => {
    expect(errorMessage(new Error("network down"))).toBe("network down");
  });

  it("never returns an empty string for unknown throwables", () => {
    expect(errorMessage(undefined)).toBe(t("common.unexpectedError"));
    expect(errorMessage({ weird: true })).toBe(t("common.unexpectedError"));
    expect(errorMessage(new Error(""))).toBe(t("common.unexpectedError"));
  });

  // Contract test: every stable code the API can return must have a
  // translation, otherwise a reader sees the server's English sentence in a
  // localised UI. Add new backend codes here when they are introduced.
  it("localises every backend error code", () => {
    setCurrentLocale("en");
    const backendCodes = [
      "not_found",
      "conflict",
      "invalid_argument",
      "unauthorized",
      "forbidden",
      "email_not_verified",
      "insufficient_stock",
      "busy",
      "cart_empty",
      "order_not_payable",
      "order_not_refundable",
      "order_not_shippable",
      "order_not_completable",
      "coupon_exhausted",
      "coupon_invalid",
      "payment_failed",
      "internal_error",
      "rate_limited",
      "invoice_unavailable",
      "registration_disabled",
      "insufficient_funds",
      "loyalty_disabled",
      "account_locked",
      "oidc_unavailable",
    ];
    const untranslated = backendCodes.filter((code) => {
      const message = errorMessage(new ApiError({ code, message: code, status: 400 }));
      return message === code; // it fell through to the raw server message
    });
    expect(untranslated, `codes with no translation: ${untranslated.join(", ")}`).toEqual([]);
  });
});
