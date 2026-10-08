import { describe, expect, it } from "vitest";

import { notificationText } from "@lib/notifications";
import { t } from "@lib/i18n/translate";
import type { Notification } from "@lib/types";

function notification(over: Partial<Notification> = {}): Notification {
  return {
    id: "n1",
    type: "order",
    title: "Order paid",
    body: "",
    read: false,
    createdAt: "2026-02-03T04:05:06Z",
    ...over,
  };
}

describe("notificationText", () => {
  it("localises from the stored code", () => {
    const { title, body } = notificationText(
      notification({ data: { code: "order.paid", orderNo: "OS-1", amountCents: 1234, currency: "USD" } }),
      t,
    );
    expect(title).toBe(t("notification.order.paid.title"));
    expect(body).toContain("OS-1");
    // The amount is rendered as currency, not as raw cents.
    expect(body).toContain("12.34");
  });

  it("falls back to the stored text when the code has no translation", () => {
    const { title, body } = notificationText(
      notification({ title: "A stored title", body: "A stored body", data: { code: "unknown.code" } }),
      t,
    );
    expect(title).toBe("A stored title");
    expect(body).toBe("A stored body");
  });

  it("uses the notification type when no code is present", () => {
    const { title } = notificationText(
      notification({ type: "system", title: "Broadcast", data: undefined }),
      t,
    );
    // "system" has a title translation, so it wins over the stored title.
    expect(title).toBe(t("notification.system.title"));
  });

  it("never throws on missing data", () => {
    expect(() => notificationText(notification({ data: undefined }), t)).not.toThrow();
    expect(() => notificationText(notification({ data: {} }), t)).not.toThrow();
  });

  it("does not leak the raw key when a template is missing", () => {
    const { title } = notificationText(notification({ title: "Fallback", data: { code: "nope" } }), t);
    expect(title).not.toContain("notification.");
  });
});
