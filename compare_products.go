package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type product struct {
	ID   int
	Name string
	Row  map[string]string
}

type detail struct {
	SourceFile string
	Line       int
	Name       string
	Row        map[string]string
}

type hit struct {
	Product product
	Type    string
	Detail  detail
}

func main() {
	productsPath := flag.String("products", "232_products_clean.csv", "CSV containing product_id and product_name")
	detailsDir := flag.String("details-dir", "invoice_details", "folder containing invoice detail CSV files")
	outputPath := flag.String("out", ".personal/invoice_product_hit_report.csv", "output CSV report path")
	flag.Parse()

	products, err := readProducts(*productsPath)
	must(err)

	details, err := readDetailsDir(*detailsDir)
	must(err)

	hits := findHits(products, details)
	must(ensureParentDir(*outputPath))
	must(writeOutput(*outputPath, hits))

	matchedProducts := make(map[int]bool)
	for _, h := range hits {
		matchedProducts[h.Product.ID] = true
	}
	fmt.Printf("checked %d products against %d detail rows\n", len(products), len(details))
	fmt.Printf("matched %d products; total hits %d\n", len(matchedProducts), len(hits))
	fmt.Printf("wrote %s\n", *outputPath)
}

func readProducts(path string) ([]product, error) {
	rows, header, err := readCSV(path)
	if err != nil {
		return nil, err
	}

	idCol, ok := header["product_id"]
	if !ok {
		return nil, fmt.Errorf("%s: missing product_id column", path)
	}
	nameCol, ok := header["product_name"]
	if !ok {
		return nil, fmt.Errorf("%s: missing product_name column", path)
	}

	products := make([]product, 0, len(rows))
	for _, row := range rows {
		if idCol >= len(row) || nameCol >= len(row) {
			continue
		}
		idText := strings.TrimSpace(row[idCol])
		if idText == "" {
			continue
		}
		id, err := strconv.Atoi(idText)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid product_id %q: %w", path, idText, err)
		}
		products = append(products, product{
			ID:   id,
			Name: strings.TrimSpace(row[nameCol]),
			Row:  rowMap(row, header),
		})
	}

	sort.Slice(products, func(i, j int) bool {
		return products[i].ID < products[j].ID
	})

	return products, nil
}

func readDetailsDir(dir string) ([]detail, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	paths := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if strings.EqualFold(filepath.Ext(path), ".csv") {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	if len(paths) == 0 {
		return nil, fmt.Errorf("%s: no .csv files found", dir)
	}

	allDetails := []detail{}
	for _, path := range paths {
		details, err := readDetails(path)
		if err != nil {
			return nil, err
		}
		allDetails = append(allDetails, details...)
	}

	return allDetails, nil
}

func readDetails(path string) ([]detail, error) {
	rows, header, err := readCSV(path)
	if err != nil {
		return nil, err
	}

	nameCol, ok := header["消費明細_品名"]
	if !ok {
		return nil, fmt.Errorf("%s: missing 消費明細_品名 column", path)
	}

	details := make([]detail, 0, len(rows))
	for i, row := range rows {
		if nameCol >= len(row) {
			continue
		}
		name := strings.TrimSpace(row[nameCol])
		if name == "" {
			continue
		}
		details = append(details, detail{
			SourceFile: filepath.Base(path),
			Line:       i + 2,
			Name:       name,
			Row:        rowMap(row, header),
		})
	}

	return details, nil
}

func findHits(products []product, details []detail) []hit {
	detailNames := make(map[string][]detail, len(details))
	for _, d := range details {
		name := normalizeName(d.Name)
		if name == "" {
			continue
		}
		detailNames[name] = append(detailNames[name], d)
	}

	hits := []hit{}
	for _, p := range products {
		productName := normalizeName(p.Name)
		if productName == "" {
			continue
		}

		exactDetails := detailNames[productName]
		for _, d := range exactDetails {
			hits = append(hits, hit{Product: p, Type: "exact", Detail: d})
		}
		if len(exactDetails) > 0 {
			continue
		}

		for _, d := range details {
			detailName := normalizeName(d.Name)
			if detailName != "" && strings.Contains(detailName, productName) {
				hits = append(hits, hit{Product: p, Type: "contains", Detail: d})
			}
		}
	}

	return hits
}

func readCSV(path string) ([][]string, map[string]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	rawHeader, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("%s: read header: %w", path, err)
	}

	header := make(map[string]int, len(rawHeader))
	for i, col := range rawHeader {
		header[trimBOM(strings.TrimSpace(col))] = i
	}

	rows := [][]string{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("%s: read row: %w", path, err)
		}
		rows = append(rows, row)
	}

	return rows, header, nil
}

func rowMap(row []string, header map[string]int) map[string]string {
	out := make(map[string]string, len(header))
	for name, idx := range header {
		if idx < len(row) {
			out[name] = row[idx]
		}
	}
	return out
}

func normalizeName(name string) string {
	name = trimBOM(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		switch r {
		case '★', '*', '@', '＊':
			return -1
		case '（':
			return '('
		case '）':
			return ')'
		case '－', '–', '—':
			return '-'
		default:
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}
	}, name)

	for {
		trimmed := strings.TrimLeft(name, ".-_/ ")
		if trimmed == name {
			break
		}
		name = trimmed
	}

	return name
}

func trimBOM(s string) string {
	return strings.TrimPrefix(s, "\ufeff")
}

func writeOutput(path string, hits []hit) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{
		"source_file",
		"product_id",
		"product_name",
		"match_type",
		"detail_line",
		"消費明細_品名",
		"發票日期",
		"發票號碼",
		"賣方名稱",
		"消費明細_數量",
		"消費明細_單價",
		"消費明細_金額",
	}); err != nil {
		return err
	}

	for _, h := range hits {
		row := []string{
			h.Detail.SourceFile,
			strconv.Itoa(h.Product.ID),
			h.Product.Name,
			h.Type,
			strconv.Itoa(h.Detail.Line),
			h.Detail.Name,
			h.Detail.Row["發票日期"],
			h.Detail.Row["發票號碼"],
			h.Detail.Row["賣方名稱"],
			h.Detail.Row["消費明細_數量"],
			h.Detail.Row["消費明細_單價"],
			h.Detail.Row["消費明細_金額"],
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return writer.Error()
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
