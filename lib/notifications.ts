import { formatMoney } from "@lib/format";
import type { MessageKey } from "@lib/i18n/messages";
import type { Notification } from "@lib/types";

type Translate = (key: MessageKey, vars?: Record<string, string | number>) => string;

/**
 * Renders a notification in the reader's language. The server stores a stable
 * `code` (e.g. "order.paid") plus template variables; when no translation is
 * registered the stored English title/body is used as a fallback.
 */
export function notificationText(n: Notification, t: Translate): { title: string; body: string } {
  const data = n.data ?? {};
  const code = typeof data.code === "string" ? data.code : n.type;

  const vars: Record<string, string | number> = {};
  for (const [key, value] of Object.entries(data)) {
    if (typeof value === "string" || typeof value === "number") vars[key] = value;
  }
  if (typeof data.amountCents === "number") {
    vars.amount = formatMoney(data.amountCents, typeof data.currency === "string" ? data.currency : "CNY");
  }
  if (typeof data.reference === "string" && data.reference) vars.reference = data.reference;
  if (typeof data.reason === "string" && data.reason) vars.reason = data.reason;

  const titleKey = `notification.${code}.title` as MessageKey;
  const bodyKey = `notification.${code}.body` as MessageKey;
  const title = t(titleKey, vars);
  if (title === titleKey) {
    return { title: n.title, body: n.body ?? "" };
  }
  const body = t(bodyKey, vars);
  return { title, body: body === bodyKey ? (n.body ?? "") : body };
}
