import { useEffect, useState } from "react";
import { Moon, Sun } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { useI18n } from "@lib/i18n";

const THEME_KEY = "openshop.theme";

// useTheme reads/writes the theme preference and keeps the <html> class in sync.
export function useTheme() {
  const [dark, setDark] = useState(() => localStorage.getItem(THEME_KEY) === "dark");
  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark);
    localStorage.setItem(THEME_KEY, dark ? "dark" : "light");
  }, [dark]);
  return { dark, toggle: () => setDark((d) => !d) };
}

/** Light/dark toggle, rendered in the footer. */
export function ThemeToggle() {
  const { dark, toggle } = useTheme();
  const { t } = useI18n();
  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={toggle}
      aria-label={t("common.toggleTheme")}
      data-testid="theme-toggle"
    >
      {dark ? <Sun className="size-4" /> : <Moon className="size-4" />}
    </Button>
  );
}
