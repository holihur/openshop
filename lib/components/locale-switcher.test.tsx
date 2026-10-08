import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { LocaleSwitcher } from "@lib/components/locale-switcher";
import { I18nProvider } from "@lib/i18n";
import { catalogs } from "@lib/i18n/messages";

function renderSwitcher() {
  return render(
    <I18nProvider>
      <LocaleSwitcher />
    </I18nProvider>,
  );
}

describe("LocaleSwitcher", () => {
  it("lists the supported locales", () => {
    renderSwitcher();
    const select = screen.getByTestId("locale-switcher");
    expect(select).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /english/i })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /中文/ })).toBeInTheDocument();
  });

  it("has an accessible label", () => {
    renderSwitcher();
    expect(screen.getByLabelText(/language/i)).toBe(screen.getByTestId("locale-switcher"));
  });

  it("changes the active locale", async () => {
    renderSwitcher();
    const select = screen.getByTestId("locale-switcher") as HTMLSelectElement;
    await userEvent.selectOptions(select, "zh");
    expect(select.value).toBe("zh");
    // The accessible label now comes from the Chinese catalogue.
    expect(screen.getByLabelText(catalogs.zh["common.language"])).toBe(select);
  });
});
