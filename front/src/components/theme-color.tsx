import { useEffect } from "react";

import { useSite } from "@lib/hooks/useSite";

// contrastForeground picks black or white text for a hex background.
function contrastForeground(hex: string): string {
  const raw = hex.replace("#", "");
  const full = raw.length === 3 ? raw.split("").map((c) => c + c).join("") : raw;
  const channel = (i: number) => parseInt(full.slice(i, i + 2), 16) / 255;
  const lin = (c: number) => (c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4));
  const luminance = 0.2126 * lin(channel(0)) + 0.7152 * lin(channel(2)) + 0.0722 * lin(channel(4));
  return luminance > 0.5 ? "#0a0a0a" : "#ffffff";
}

/** Applies the operator-configured brand color as the theme's primary color. */
export function ThemeColor() {
  const { data: site } = useSite();
  const color = site?.themeColor?.trim();

  useEffect(() => {
    const root = document.documentElement;
    if (!color) {
      root.style.removeProperty("--primary");
      root.style.removeProperty("--primary-foreground");
      root.style.removeProperty("--ring");
      return;
    }
    root.style.setProperty("--primary", color);
    root.style.setProperty("--primary-foreground", contrastForeground(color));
    root.style.setProperty("--ring", color);
  }, [color]);

  return null;
}
