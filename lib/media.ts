// Builds a responsive `srcset` for images served by the backend's /uploads
// endpoint, which resizes raster images on demand (?w=NNN). Returns undefined
// for external/CDN URLs that the backend does not resize.
export function responsiveSrcSet(url?: string | null): string | undefined {
  if (!url || !url.includes("/uploads/")) return undefined;
  const sep = url.includes("?") ? "&" : "?";
  return [320, 640, 1280].map((w) => `${url}${sep}w=${w} ${w}w`).join(", ");
}
