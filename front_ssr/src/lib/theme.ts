/** Brand colour helpers shared by the header, buttons and hero. */

const FALLBACK = "#4f46e5";

function channels(hex: string): [number, number, number] | null {
  const raw = hex.trim().replace("#", "");
  const full =
    raw.length === 3
      ? raw
          .split("")
          .map((c) => c + c)
          .join("")
      : raw;
  if (!/^[0-9a-fA-F]{6}$/.test(full)) return null;
  return [
    parseInt(full.slice(0, 2), 16),
    parseInt(full.slice(2, 4), 16),
    parseInt(full.slice(4, 6), 16),
  ];
}

/**
 * contrastForeground picks black or white text for a background, so an operator
 * who chooses a pale brand colour still gets readable buttons.
 */
export function contrastForeground(hex: string): string {
  const rgb = channels(hex);
  if (!rgb) return "#ffffff";
  const lin = (c: number) => {
    const v = c / 255;
    return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
  };
  const luminance = 0.2126 * lin(rgb[0]) + 0.7152 * lin(rgb[1]) + 0.0722 * lin(rgb[2]);
  return luminance > 0.5 ? "#0a0a0a" : "#ffffff";
}

/** A translucent version of the brand colour, for hero washes. */
export function withAlpha(hex: string, alpha: number): string {
  const rgb = channels(hex);
  if (!rgb) return `rgba(79, 70, 229, ${alpha})`;
  return `rgba(${rgb[0]}, ${rgb[1]}, ${rgb[2]}, ${alpha})`;
}

/** The configured brand colour, or a sensible default when none is set. */
export function brandColor(configured?: string): string {
  return channels(configured ?? "") ? (configured as string) : FALLBACK;
}
