import { describe, expect, it } from "vitest";

import { catalogs } from "@lib/i18n/messages";
import { getCurrentLocale, setCurrentLocale, t } from "@lib/i18n/translate";

describe("translate", () => {
  it("interpolates named variables", () => {
    setCurrentLocale("en");
    expect(t("common.pageOf", { page: 2, total: 5 })).toContain("2");
    expect(t("common.pageOf", { page: 2, total: 5 })).toContain("5");
  });

  it("leaves an unknown placeholder untouched instead of dropping it", () => {
    setCurrentLocale("en");
    // A missing variable is visible, which makes the bug obvious rather than
    // silently rendering "undefined".
    expect(t("common.pageOf", { page: 1 })).toContain("{total}");
  });

  it("switches locale", () => {
    setCurrentLocale("zh");
    expect(getCurrentLocale()).toBe("zh");
    expect(t("common.cancel")).toBe(catalogs.zh["common.cancel"]);
    setCurrentLocale("en");
    expect(t("common.cancel")).toBe(catalogs.en["common.cancel"]);
  });

  it("returns the key itself for an unknown key", () => {
    setCurrentLocale("en");
    // @ts-expect-error deliberately bypassing the MessageKey union
    expect(t("definitely.not.a.key")).toBe("definitely.not.a.key");
  });
});

// Both catalogues must stay in step: a key added to one language and forgotten
// in the other shows English text to a Chinese reader (or vice versa).
describe("catalogues", () => {
  it("define the same keys in both locales", () => {
    const en = Object.keys(catalogs.en).sort();
    const zh = Object.keys(catalogs.zh).sort();
    const missingInZh = en.filter((k) => !(k in catalogs.zh));
    const missingInEn = zh.filter((k) => !(k in catalogs.en));
    expect(missingInZh, `missing in zh: ${missingInZh.slice(0, 10).join(", ")}`).toEqual([]);
    expect(missingInEn, `missing in en: ${missingInEn.slice(0, 10).join(", ")}`).toEqual([]);
  });

  it("has no empty values", () => {
    for (const locale of ["en", "zh"] as const) {
      const empty = Object.entries(catalogs[locale])
        .filter(([, v]) => typeof v === "string" && v.trim() === "")
        .map(([k]) => k);
      // notification.system.body is intentionally empty (a title-only notice).
      const unexpected = empty.filter((k) => k !== "notification.system.body");
      expect(unexpected, `empty ${locale} values: ${unexpected.join(", ")}`).toEqual([]);
    }
  });
});
