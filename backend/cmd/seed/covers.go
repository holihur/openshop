package main

import (
	"bytes"
	"context"
	"fmt"
	"html"

	"github.com/holihur/openshop/internal/port"
)

// productCover renders a small, self-contained SVG cover for a demo product.
// Generating the artwork locally means the seed ships no third-party imagery,
// so there are no licensing, attribution or hotlinking concerns, and it works
// offline. Replace the returned URL with real photos in a real catalog.
func productCover(title, emoji, from, to string) []byte {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600" viewBox="0 0 800 600" role="img" aria-label="%s">
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="%s"/>
      <stop offset="1" stop-color="%s"/>
    </linearGradient>
  </defs>
  <rect width="800" height="600" fill="url(#bg)"/>
  <circle cx="648" cy="118" r="190" fill="#ffffff" opacity="0.10"/>
  <circle cx="128" cy="512" r="150" fill="#000000" opacity="0.08"/>
  <text x="400" y="320" font-size="210" text-anchor="middle" dominant-baseline="central">%s</text>
  <text x="400" y="500" font-size="42" font-weight="600" fill="#ffffff"
        font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" text-anchor="middle">%s</text>
</svg>`, html.EscapeString(title), from, to, emoji, html.EscapeString(title))
	return []byte(svg)
}

// uploadCover stores the generated SVG through the ObjectStorage port and
// returns its public URL, so the same code works for the local and S3 drivers.
func uploadCover(ctx context.Context, store port.ObjectStorage, slug, title, emoji, from, to string) (string, error) {
	svg := productCover(title, emoji, from, to)
	key := "seed/" + slug + ".svg"
	return store.Put(ctx, key, bytes.NewReader(svg), int64(len(svg)), "image/svg+xml")
}
