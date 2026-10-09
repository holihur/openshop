package qr

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var pathRe = regexp.MustCompile(`M(\d+) (\d+)h1v1h-1z`)

func TestSVGEncodesThePayload(t *testing.T) {
	svg, err := SVG("weixin://wxpay/bizpayurl?pr=abc")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
		t.Fatalf("not an SVG document: %.40s", svg)
	}
	// The quiet zone plus a real matrix: a QR code of this payload is at least
	// 21 modules across, so the viewBox must be larger than that.
	if !strings.Contains(svg, "shape-rendering=\"crispEdges\"") {
		t.Error("modules must be drawn crisply, otherwise the SVG blurs when scaled")
	}
	if strings.Count(svg, "M") < 100 {
		t.Errorf("too few modules drawn: %d", strings.Count(svg, "M"))
	}

	// The same payload must always produce the same image, and a different one
	// a different image.
	again, _ := SVG("weixin://wxpay/bizpayurl?pr=abc")
	if again != svg {
		t.Error("encoding is not deterministic")
	}
	other, _ := SVG("weixin://wxpay/bizpayurl?pr=xyz")
	if other == svg {
		t.Error("different payloads produced the same code")
	}
}

// decode reads the path data back into a grid, so the drawing itself can be
// checked rather than trusting that the encoder and the renderer agree.
func decode(t *testing.T, svg string, size int) [][]bool {
	t.Helper()
	grid := make([][]bool, size)
	for i := range grid {
		grid[i] = make([]bool, size)
	}
	// Every module is emitted as "M<x> <y>h1v1h-1z".
	for _, match := range pathRe.FindAllStringSubmatch(svg, -1) {
		var x, y int
		if _, err := fmt.Sscanf(match[1]+" "+match[2], "%d %d", &x, &y); err != nil {
			t.Fatalf("module %v: %v", match, err)
		}
		if x < 0 || y < 0 || x >= size || y >= size {
			t.Fatalf("module (%d,%d) is outside the %dx%d grid", x, y, size, size)
		}
		grid[y][x] = true
	}
	return grid
}

func TestSVGDrawsFindersAndQuietZone(t *testing.T) {
	const payload = "weixin://wxpay/bizpayurl?pr=abc"
	svg, err := SVG(payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var size int
	if _, err := fmt.Sscanf(svg[strings.Index(svg, "viewBox=")+len(`viewBox="0 0 `):], "%d", &size); err != nil {
		t.Fatalf("viewBox: %v", err)
	}
	grid := decode(t, svg, size)

	// The quiet zone must be clear, otherwise scanners cannot find the code.
	for i := 0; i < size; i++ {
		for _, point := range [][2]int{{i, 0}, {i, 1}, {i, size - 1}, {0, i}, {1, i}, {size - 1, i}} {
			if grid[point[1]][point[0]] {
				t.Fatalf("quiet zone is not clear at (%d,%d)", point[0], point[1])
			}
		}
	}

	// A QR code always has three 7x7 finder patterns, in the top-left,
	// top-right and bottom-left corners of the data area.
	const quiet = 2
	for _, corner := range [][2]int{{quiet, quiet}, {size - quiet - 7, quiet}, {quiet, size - quiet - 7}} {
		expected := []string{
			"1111111",
			"1000001",
			"1011101",
			"1011101",
			"1011101",
			"1000001",
			"1111111",
		}
		for dy, row := range expected {
			for dx, want := range row {
				got := grid[corner[1]+dy][corner[0]+dx]
				if got != (want == '1') {
					t.Fatalf("finder pattern mismatch at (%d,%d): got %v", corner[0]+dx, corner[1]+dy, got)
				}
			}
		}
	}
}

func TestSVGRejectsEmptyPayload(t *testing.T) {
	if _, err := SVG("   "); err == nil {
		t.Fatal("an empty payload must be rejected")
	}
}
