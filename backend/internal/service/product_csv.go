package service

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/holihur/openshop/internal/domain"
)

// ProductCSVLocales are the locales exported as name_<locale> columns. Import
// ignores unknown name_ columns.
var ProductCSVLocales = []string{"en", "zh"}

// productCSVColumns is the file header, in order. There is one row per variant;
// rows sharing a handle belong to the same product, which is what makes bulk
// editing in a spreadsheet practical (the same shape Shopify and WooCommerce
// use). A row with an empty variant_sku carries product fields only.
func productCSVColumns() []string {
	cols := []string{
		"handle", "title", "description", "status", "category",
		"price", "stock", "weight_grams", "cover_image", "images",
	}
	for _, loc := range ProductCSVLocales {
		cols = append(cols, "name_"+loc)
	}
	return append(cols,
		"variant_sku", "variant_name", "variant_price", "variant_stock", "variant_weight", "variant_active")
}

// csvProductRow is one parsed line. Empty strings mean "not provided", which is
// what lets a partial file update only the columns it fills in.
type csvProductRow struct {
	Line    int
	Handle  string
	Title   string
	Desc    string
	Status  string
	CatSlug string
	Price   string
	Stock   string
	Weight  string
	Cover   string
	Images  string
	Names   map[string]string
	VarSKU  string
	VarName string
	VarP    string
	VarS    string
	VarW    string
	VarA    string
}

// ImportOptions controls an import run.
type ImportOptions struct {
	// DryRun validates and reports without writing anything.
	DryRun bool
	// Create allows handles that do not exist yet to be created.
	Create bool
}

// ImportReport summarises what an import did (or would do).
type ImportReport struct {
	RowsRead        int           `json:"rowsRead"`
	ProductsCreated int           `json:"productsCreated"`
	ProductsUpdated int           `json:"productsUpdated"`
	VariantsCreated int           `json:"variantsCreated"`
	VariantsUpdated int           `json:"variantsUpdated"`
	Errors          []ImportError `json:"errors,omitempty"`
	DryRun          bool          `json:"dryRun"`
}

// ImportError points at the offending line so the file can be fixed.
type ImportError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func (r *ImportReport) addError(line int, format string, args ...any) {
	if len(r.Errors) < 100 {
		r.Errors = append(r.Errors, ImportError{Line: line, Message: fmt.Sprintf(format, args...)})
	}
}

// ExportProductCSV streams every product as CSV, one row per variant.
func (s *CatalogService) ExportProductCSV(ctx context.Context, w io.Writer) error {
	bw := bufio.NewWriter(w)
	cw := csv.NewWriter(bw)

	if err := cw.Write(productCSVColumns()); err != nil {
		return err
	}

	categories, err := s.categories.List(ctx)
	if err != nil {
		return err
	}
	slugByCategory := make(map[string]string, len(categories))
	for _, c := range categories {
		slugByCategory[c.ID] = c.Slug
	}

	for page := 1; ; page++ {
		result, err := s.products.List(ctx, domain.ProductFilter{Page: page, PageSize: 100})
		if err != nil {
			return err
		}
		for _, p := range result.Items {
			variants, err := s.variants.ListByProduct(ctx, p.ID)
			if err != nil {
				return err
			}
			product := []string{
				p.Slug, p.Title, p.Description, string(p.Status), slugByCategory[p.CategoryID],
				major(p.PriceCents), strconv.Itoa(p.Stock), strconv.Itoa(p.WeightGrams),
				p.CoverImage, strings.Join(p.Images, "|"),
			}
			for _, loc := range ProductCSVLocales {
				product = append(product, p.Names[loc])
			}

			if len(variants) == 0 {
				if err := cw.Write(append(product, "", "", "", "", "", "")); err != nil {
					return err
				}
				continue
			}
			for _, v := range variants {
				row := append(append([]string{}, product...),
					v.SKU, v.Name, major(v.PriceCents), strconv.Itoa(v.Stock),
					strconv.Itoa(v.WeightGrams), strconv.FormatBool(v.Active))
				if err := cw.Write(row); err != nil {
					return err
				}
			}
		}
		if page*100 >= int(result.Total) || len(result.Items) == 0 {
			break
		}
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	return bw.Flush()
}

// ImportProductCSV applies a CSV file. Rows are grouped by handle, so a product
// and its variants arrive together. A bad row is reported and skipped rather
// than aborting the whole file.
func (s *CatalogService) ImportProductCSV(ctx context.Context, r io.Reader, opts ImportOptions) (ImportReport, error) {
	report := ImportReport{DryRun: opts.DryRun}
	rows, err := parseProductCSV(r)
	if err != nil {
		return report, err
	}
	report.RowsRead = len(rows)

	// Group rows by handle, preserving file order.
	order := make([]string, 0)
	groups := map[string][]csvProductRow{}
	for _, row := range rows {
		if row.Handle == "" {
			report.addError(row.Line, "handle is required")
			continue
		}
		if _, seen := groups[row.Handle]; !seen {
			order = append(order, row.Handle)
		}
		groups[row.Handle] = append(groups[row.Handle], row)
	}

	categories, err := s.categories.List(ctx)
	if err != nil {
		return report, err
	}
	idByCategory := make(map[string]string, len(categories))
	for _, c := range categories {
		idByCategory[strings.ToLower(c.Slug)] = c.ID
	}

	for _, handle := range order {
		group := groups[handle]
		if err := s.importGroup(ctx, handle, group, idByCategory, opts, &report); err != nil {
			report.addError(group[0].Line, "%v", err)
		}
	}
	return report, nil
}

func (s *CatalogService) importGroup(
	ctx context.Context,
	handle string,
	group []csvProductRow,
	idByCategory map[string]string,
	opts ImportOptions,
	report *ImportReport,
) error {
	head := group[0]
	// The product fields come from the first row that supplies a title.
	for _, row := range group {
		if row.Title != "" {
			head = row
			break
		}
	}
	if head.Title == "" {
		return fmt.Errorf("title is required on at least one row for handle %q", handle)
	}

	var categoryID string
	if head.CatSlug != "" {
		id, ok := idByCategory[strings.ToLower(head.CatSlug)]
		if !ok {
			return fmt.Errorf("unknown category %q", head.CatSlug)
		}
		categoryID = id
	}

	status := domain.ProductDraft
	if head.Status != "" {
		status = domain.ProductStatus(head.Status)
		switch status {
		case domain.ProductDraft, domain.ProductPublished, domain.ProductArchived:
		default:
			return fmt.Errorf("unknown status %q", head.Status)
		}
	}
	price, err := parseMajor(head.Price)
	if err != nil {
		return fmt.Errorf("price: %w", err)
	}
	stock, err := parseCount(head.Stock)
	if err != nil {
		return fmt.Errorf("stock: %w", err)
	}
	weight, err := parseCount(head.Weight)
	if err != nil {
		return fmt.Errorf("weight_grams: %w", err)
	}
	names := map[string]string{}
	for _, loc := range ProductCSVLocales {
		if v := strings.TrimSpace(head.Names[loc]); v != "" {
			names[loc] = v
		}
	}

	existing, err := s.products.FindBySlug(ctx, handle)
	switch {
	case err == nil:
		input := UpdateProductInput{
			Title: &head.Title, Description: &head.Desc, PriceCents: &price,
			Stock: &stock, WeightGrams: &weight, Status: &status,
		}
		if len(names) > 0 {
			input.Names = names
		}
		if head.CatSlug != "" {
			input.CategoryID = &categoryID
		}
		if head.Cover != "" {
			input.CoverImage = &head.Cover
		}
		if head.Images != "" {
			input.Images = splitImages(head.Images)
		}
		if opts.DryRun {
			report.ProductsUpdated++
		} else if _, err := s.UpdateProduct(ctx, existing.ID, input); err != nil {
			return err
		} else {
			report.ProductsUpdated++
		}
		// Always read the current variants, even for a dry run, so the report
		// distinguishes an update from a create.
		variants, err := s.variants.ListByProduct(ctx, existing.ID)
		if err != nil {
			return err
		}
		return s.importVariants(ctx, existing.ID, group, variants, opts, report)

	case err == domain.ErrNotFound:
		if !opts.Create {
			return fmt.Errorf("product %q not found (create is disabled)", handle)
		}
		product := &domain.Product{
			ID: s.ids.NewID(), CategoryID: categoryID, Title: head.Title, Slug: handle,
			Names: names, Description: head.Desc, PriceCents: price, Currency: s.currency,
			CoverImage: head.Cover, Images: splitImages(head.Images), Status: status,
			Stock: stock, WeightGrams: weight, CreatedAt: s.clock.Now(), UpdatedAt: s.clock.Now(),
		}
		if opts.DryRun {
			report.ProductsCreated++
			report.VariantsCreated += countVariantRows(group)
			return nil
		}
		if err := s.products.Create(ctx, product); err != nil {
			return err
		}
		report.ProductsCreated++
		return s.importVariants(ctx, product.ID, group, nil, opts, report)

	default:
		return err
	}
}

func (s *CatalogService) importVariants(
	ctx context.Context,
	productID string,
	group []csvProductRow,
	existing []domain.Variant,
	opts ImportOptions,
	report *ImportReport,
) error {
	bySKU := make(map[string]domain.Variant, len(existing))
	for _, v := range existing {
		bySKU[v.SKU] = v
	}

	for _, row := range group {
		if row.VarSKU == "" {
			continue
		}
		price, err := parseMajor(row.VarP)
		if err != nil {
			report.addError(row.Line, "variant_price: %v", err)
			continue
		}
		stock, err := parseCount(row.VarS)
		if err != nil {
			report.addError(row.Line, "variant_stock: %v", err)
			continue
		}
		weight, err := parseCount(row.VarW)
		if err != nil {
			report.addError(row.Line, "variant_weight: %v", err)
			continue
		}
		active := true
		if row.VarA != "" {
			active = strings.EqualFold(row.VarA, "true") || row.VarA == "1"
		}
		name := row.VarName
		if name == "" {
			name = row.VarSKU
		}

		if current, ok := bySKU[row.VarSKU]; ok {
			if opts.DryRun {
				report.VariantsUpdated++
				continue
			}
			if _, err := s.UpdateVariant(ctx, current.ID, UpdateVariantInput{
				Name: &name, PriceCents: &price, Stock: &stock, WeightGrams: &weight, Active: &active,
			}); err != nil {
				report.addError(row.Line, "%v", err)
				continue
			}
			report.VariantsUpdated++
			continue
		}
		if opts.DryRun {
			report.VariantsCreated++
			continue
		}
		if _, err := s.CreateVariant(ctx, CreateVariantInput{
			ProductID: productID, SKU: row.VarSKU, Name: name, PriceCents: price,
			Stock: stock, WeightGrams: weight, Active: active,
		}); err != nil {
			report.addError(row.Line, "%v", err)
			continue
		}
		report.VariantsCreated++
	}
	return nil
}

func countVariantRows(group []csvProductRow) int {
	n := 0
	for _, row := range group {
		if row.VarSKU != "" {
			n++
		}
	}
	return n
}

// parseProductCSV reads the header and every row. Columns are located by name,
// so a file may carry extra columns or reorder them.
func parseProductCSV(r io.Reader) ([]csvProductRow, error) {
	reader := csv.NewReader(r)
	// Spreadsheets quote liberally and pad rows; be forgiving.
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read the header row", domain.ErrInvalidArgument)
	}
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[strings.ToLower(strings.TrimSpace(name))] = i
	}
	if _, ok := index["handle"]; !ok {
		return nil, fmt.Errorf("%w: the file must have a handle column", domain.ErrInvalidArgument)
	}
	if _, ok := index["title"]; !ok {
		return nil, fmt.Errorf("%w: the file must have a title column", domain.ErrInvalidArgument)
	}

	get := func(record []string, key string) string {
		i, ok := index[key]
		if !ok || i >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[i])
	}

	var rows []csvProductRow
	for line := 2; ; line++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: line %d: %v", domain.ErrInvalidArgument, line, err)
		}
		if len(record) == 0 || strings.TrimSpace(strings.Join(record, "")) == "" {
			continue
		}
		row := csvProductRow{
			Line:  line,
			Names: map[string]string{},
		}
		row.Handle = get(record, "handle")
		row.Title = get(record, "title")
		row.Desc = get(record, "description")
		row.Status = strings.ToLower(get(record, "status"))
		row.CatSlug = get(record, "category")
		row.Price = get(record, "price")
		row.Stock = get(record, "stock")
		row.Weight = get(record, "weight_grams")
		row.Cover = get(record, "cover_image")
		row.Images = get(record, "images")
		row.VarSKU = get(record, "variant_sku")
		row.VarName = get(record, "variant_name")
		row.VarP = get(record, "variant_price")
		row.VarS = get(record, "variant_stock")
		row.VarW = get(record, "variant_weight")
		row.VarA = strings.ToLower(get(record, "variant_active"))
		for _, loc := range ProductCSVLocales {
			row.Names[loc] = get(record, "name_"+loc)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// major renders cents as a plain decimal string for spreadsheets.
func major(cents int64) string {
	return strconv.FormatFloat(float64(cents)/100, 'f', 2, 64)
}

// parseMajor accepts "12.34", "12", "1,234.50" or a value with a currency symbol
// and returns cents.
func parseMajor(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r == '.', r == '-':
			return r
		}
		return -1
	}, raw)
	if cleaned == "" {
		return 0, fmt.Errorf("not a number")
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("not a number")
	}
	if value < 0 {
		return 0, fmt.Errorf("must not be negative")
	}
	return int64(value*100 + 0.5), nil
}

func parseCount(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(strings.ReplaceAll(raw, ",", ""))
	if err != nil {
		return 0, fmt.Errorf("not a whole number")
	}
	if n < 0 {
		return 0, fmt.Errorf("must not be negative")
	}
	return n, nil
}

func splitImages(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == '|' || r == '\n' })
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
