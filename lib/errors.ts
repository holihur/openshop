import { ApiError } from "@lib/api";
import { t } from "@lib/i18n";
import type { MessageKey } from "@lib/i18n/messages";

// Maps the backend's stable error codes to localised messages. Unknown codes
// fall back to the server's message so nothing is ever hidden.
const byCode: Record<string, MessageKey> = {
  not_found: "errors.not_found",
  conflict: "errors.conflict",
  invalid_argument: "errors.invalid_argument",
  unauthorized: "errors.unauthorized",
  forbidden: "errors.forbidden",
  email_not_verified: "errors.email_not_verified",
  insufficient_stock: "errors.insufficient_stock",
  busy: "errors.busy",
  cart_empty: "errors.cart_empty",
  order_not_payable: "errors.order_not_payable",
  order_not_refundable: "errors.order_not_refundable",
  order_not_shippable: "errors.order_not_shippable",
  order_not_completable: "errors.order_not_completable",
  coupon_exhausted: "errors.coupon_exhausted",
  coupon_invalid: "errors.coupon_invalid",
  payment_failed: "errors.payment_failed",
  internal_error: "errors.internal_error",
  rate_limited: "errors.rate_limited",
  invoice_unavailable: "errors.invoice_unavailable",
  registration_disabled: "errors.registration_disabled",
  insufficient_funds: "errors.insufficient_funds",
  loyalty_disabled: "errors.loyalty_disabled",
  account_locked: "errors.account_locked",
  oidc_unavailable: "errors.oidc_unavailable",
};

/** Localised, user-facing message for any thrown value. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    const key = byCode[err.code];
    return key ? t(key) : err.message;
  }
  if (err instanceof Error && err.message) {
    return err.message;
  }
  return t("common.unexpectedError");
}
