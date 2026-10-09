// Package qr renders a QR code as an SVG.
//
// It exists for payment flows whose payload is not a URL the browser can open:
// WeChat Pay returns a "weixin://" code_url that the shopper scans with the
// WeChat app, so the storefront needs an image to show.
package qr

import (
	"fmt"
	"strings"

	"rsc.io/qr"
)

// SVG encodes text as a scalable QR code. A quiet zone is included because
// scanners need the margin to lock on.
func SVG(text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("qr: empty payload")
	}
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", fmt.Errorf("qr: %w", err)
	}

	const quiet = 2
	size := code.Size + quiet*2
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" `+
		`shape-rendering="crispEdges" role="img" aria-label="Payment QR code">`, size, size)
	// White background so the code scans on a dark page too.
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#ffffff"/>`, size, size)
	b.WriteString(`<path fill="#000000" d="`)
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; x++ {
			if !code.Black(x, y) {
				continue
			}
			// Each module is one unit, offset by the quiet zone.
			fmt.Fprintf(&b, "M%d %dh1v1h-1z", x+quiet, y+quiet)
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String(), nil
}
