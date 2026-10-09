package service

import (
	"strings"
	"testing"
)

func TestParseProductCSVLocatesColumnsByName(t *testing.T) {
	// Reordered columns and an extra one must still parse.
	file := strings.Join([]string{
		"title,handle,unknown,price,stock,name_zh,variant_sku,variant_price",
		"Backpack,pk-1,ignored,129.90,4,背包,PK-1-RED,99.50",
	}, "\n")

	rows, err := parseProductCSV(strings.NewReader(file))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	got := rows[0]
	if got.Handle != "pk-1" || got.Title != "Backpack" {
		t.Fatalf("unexpected row: %+v", got)
	}
	if got.Names["zh"] != "背包" {
		t.Errorf("localized name = %q", got.Names["zh"])
	}
	if got.VarSKU != "PK-1-RED" || got.VarP != "99.50" {
		t.Errorf("variant columns = %q / %q", got.VarSKU, got.VarP)
	}
	if got.Line != 2 {
		t.Errorf("line = %d, want 2 (the header is line 1)", got.Line)
	}
}

func TestParseProductCSVSkipsBlankLines(t *testing.T) {
	file := "handle,title\n\npk-1,Backpack\n,,\n"
	rows, err := parseProductCSV(strings.NewReader(file))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 1 || rows[0].Handle != "pk-1" {
		t.Fatalf("blank rows must be skipped, got %+v", rows)
	}
}

func TestParseProductCSVRequiresHandleAndTitle(t *testing.T) {
	for name, file := range map[string]string{
		"no handle": "title\nBackpack\n",
		"no title":  "handle\npk-1\n",
	} {
		if _, err := parseProductCSV(strings.NewReader(file)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestParseMajor(t *testing.T) {
	ok := map[string]int64{
		"":          0,
		"12":        1200,
		"12.34":     1234,
		"1,234.50":  123450,
		"$19.99":    1999,
		"¥1,299.00": 129900,
		"  0.05  ":  5,
		"99.999":    10000, // rounds to the nearest cent
	}
	for in, want := range ok {
		got, err := parseMajor(in)
		if err != nil {
			t.Errorf("parseMajor(%q) errored: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseMajor(%q) = %d, want %d", in, got, want)
		}
	}
	for _, bad := range []string{"abc", "-5"} {
		if _, err := parseMajor(bad); err == nil {
			t.Errorf("parseMajor(%q) should fail", bad)
		}
	}
}

func TestParseCount(t *testing.T) {
	ok := map[string]int{"": 0, "5": 5, "1,000": 1000, " 7 ": 7}
	for in, want := range ok {
		got, err := parseCount(in)
		if err != nil {
			t.Errorf("parseCount(%q) errored: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseCount(%q) = %d, want %d", in, got, want)
		}
	}
	for _, bad := range []string{"1.5", "-2", "x"} {
		if _, err := parseCount(bad); err == nil {
			t.Errorf("parseCount(%q) should fail", bad)
		}
	}
}

func TestSplitImages(t *testing.T) {
	got := splitImages("a.jpg|b.jpg\nc.jpg")
	want := []string{"a.jpg", "b.jpg", "c.jpg"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("image %d = %q, want %q", i, got[i], want[i])
		}
	}
	if splitImages("   ") != nil {
		t.Error("blank input must produce no images")
	}
}

// The header written by export must be readable by import, so a file that was
// downloaded and re-uploaded round-trips.
func TestProductCSVHeaderIsImportable(t *testing.T) {
	header := productCSVColumns()
	var b strings.Builder
	for i, col := range header {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(col)
	}
	b.WriteString("\npk-1,Backpack,Desc,published,,129.90,3,500,,,Backpack EN,背包,PK-1-RED,Red,99.50,2,480,true\n")

	rows, err := parseProductCSV(strings.NewReader(b.String()))
	if err != nil {
		t.Fatalf("the exported header must be parseable: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	row := rows[0]
	if row.Handle != "pk-1" || row.Status != "published" || row.Price != "129.90" {
		t.Errorf("unexpected row: %+v", row)
	}
	if row.Names["en"] != "Backpack EN" || row.Names["zh"] != "背包" {
		t.Errorf("names = %v", row.Names)
	}
	if row.VarSKU != "PK-1-RED" || row.VarA != "true" {
		t.Errorf("variant = %+v", row)
	}
}

func majorRoundTrip(t *testing.T) {
	t.Helper()
	for _, cents := range []int64{0, 1, 999, 1234, 129900} {
		got, err := parseMajor(major(cents))
		if err != nil {
			t.Fatalf("round trip %d: %v", cents, err)
		}
		if got != cents {
			t.Errorf("round trip %d -> %d", cents, got)
		}
	}
}

func TestMajorRoundTrip(t *testing.T) { majorRoundTrip(t) }
