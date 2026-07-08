# 中聯油脂案產品發票比對工具

This repository contains a Go utility for matching a cleaned product list against local cloud invoice detail CSV exports.

The repository intentionally does not track private working data under `.personal/`, `.presonal/`, or `invoice_details/`.

## Repository Contents

| Path | Purpose |
| --- | --- |
| `compare_products.go` | Main Go program. |
| `232_products_clean.csv` | Clean product list. Must include `product_id` and `product_name`. |
| `232項產品清單.pdf` | Source/reference product PDF. |
| `.personal/.gitkeep` | Keeps the private output directory present in Git. Actual generated reports are ignored. |
| `invoice_details/.gitkeep` | Keeps the invoice input directory present in Git. Actual invoice CSV files are ignored. |

## Ignored Local Data

These paths are for local/private files and should not be uploaded:

| Path | Notes |
| --- | --- |
| `.personal/*` | Local personal data and generated reports. Only `.personal/.gitkeep` is tracked. |
| `.presonal/` | Existing local personal data folder with the original misspelling. |
| `invoice_details/*` | Local invoice detail CSV exports. Only `invoice_details/.gitkeep` is tracked. |

Because Git cannot store a truly empty directory, the remote `.personal/` and `invoice_details/` folders each contain only `.gitkeep`. Put real invoice CSV files in this folder locally before running the tool.

## Input Requirements

The product file defaults to:

```text
232_products_clean.csv
```

It must contain:

| Column | Purpose |
| --- | --- |
| `product_id` | Product ID used for sorting and report output. |
| `product_name` | Product name to match against invoice detail rows. |

Local invoice detail CSV files go under:

```text
invoice_details/
```

Each invoice CSV must include:

| Column | Purpose |
| --- | --- |
| `消費明細_品名` | Product name from the invoice detail row. |
| `發票日期` | Written to the report when a product matches. |
| `發票號碼` | Written to the report when a product matches. |
| `賣方名稱` | Written to the report for context. |
| `消費明細_數量` | Written to the report for context. |
| `消費明細_單價` | Written to the report for context. |
| `消費明細_金額` | Written to the report for context. |

## Usage

Run with the default paths:

```bash
go run compare_products.go
```

This is equivalent to:

```bash
go run compare_products.go \
  -products 232_products_clean.csv \
  -details-dir invoice_details \
  -out .personal/invoice_product_hit_report.csv
```

The program scans every `.csv` file in `invoice_details/`. If the folder has no invoice CSV files, it exits with an error.

## Output

The default output file is private local data:

```text
.personal/invoice_product_hit_report.csv
```

The program creates `.personal/` automatically when the output path uses that directory. The output path can be changed with `-out`, but generated reports should stay out of Git.

Report columns:

| Column | Description |
| --- | --- |
| `source_file` | Invoice CSV file that contained the match. |
| `product_id` | Product ID from `232_products_clean.csv`. |
| `product_name` | Product name from `232_products_clean.csv`. |
| `match_type` | `exact` or `contains`. |
| `detail_line` | Line number in the source invoice CSV. |
| `消費明細_品名` | Matched invoice product name. |
| `發票日期` | Invoice date for the hit. |
| `發票號碼` | Invoice number for the hit. |
| `賣方名稱` | Seller name from the invoice row. |
| `消費明細_數量` | Quantity from the invoice row. |
| `消費明細_單價` | Unit price from the invoice row. |
| `消費明細_金額` | Amount from the invoice row. |

## Matching Rules

For each product, the program normalizes product names by removing simple receipt decorations and whitespace. It first checks for exact normalized matches. If no exact match exists, it checks whether the normalized invoice detail name contains the normalized product name.

Each matching invoice row is written as a separate output row.
