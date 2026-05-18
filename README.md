# expense

A pair of Go command-line tools for preparing UK business-expense submissions from a folder of receipts:

- `expense` merges receipt images and PDF receipts into a single, size-optimised PDF suitable for attaching to a claim.
- `expense-sheet` reads the same folder, extracts the structured fields from each receipt via the Claude API, and fills the line items on a company-issued .xlsx claim template.

Both tools accept the same input formats: `.pdf`, `.jpg`, `.jpeg`, `.png`, `.heic`, `.heif`. `expense` processes files in lexicographic order by filename, so prefix receipts with a date (e.g. `2026-05-11_taxi.heic`) if the order of pages in the merged PDF matters. `expense-sheet` ignores filenames for ordering and instead sorts the rows of the claim by the transaction date the model extracts from each receipt.

## Install

Build both binaries locally with the supplied Makefile:

```sh
make build
```

This produces `./expense` and `./expense-sheet` in the working directory. Alternatively, install directly with Go:

```sh
go install github.com/stefan/expense@latest
go install github.com/stefan/expense/cmd/expense-sheet@latest
```

### Runtime dependencies

| Dependency       | Required for                                          | macOS install            |
| ---------------- | ----------------------------------------------------- | ------------------------ |
| `gs` (Ghostscript) | Recommended for `expense`: recompresses embedded images in input PDFs | `brew install ghostscript` |
| `sips`           | Required when HEIC/HEIF inputs are present            | Built in on macOS        |
| `ANTHROPIC_API_KEY` | Required for `expense-sheet`: extraction via Claude | Set in shell environment |

If `gs` is missing, `expense` falls back to a structural-only optimisation pass and prints a warning to stderr. If `sips` is missing and HEIC inputs are present, both tools abort up front with an instruction to convert the files by other means.

## expense: merge receipts into a single PDF

```sh
expense -in ./receipts -out ./2026-q2-claim.pdf
```

For each input, `expense` performs the following:

1. PDF inputs are passed through unchanged at the merge step.
2. Image inputs (including HEIC, which is transcoded to PNG via `sips`) are downscaled so the long edge is at most 2000 pixels, converted to 8-bit greyscale, re-encoded as JPEG at quality 80, and placed on an A4 portrait page. Landscape-pixel photos are rotated 90 degrees clockwise so receipts shot in portrait read upright.
3. All PDFs are merged with `pdfcpu`, run through its structural optimisation pass, and then piped through Ghostscript with `-dPDFSETTINGS=/ebook` (images downsampled to 150 DPI) to actually shrink large embedded images in airline-receipt PDFs.

The output PDF carries an internal `expense-merge` marker keyword in its PDF Info dictionary. `expense-sheet` recognises this marker and silently skips the file when scanning for receipts, so you may safely leave the merged PDF in the same directory as the source receipts.

### Flags

| Flag          | Description                                                                  |
| ------------- | ---------------------------------------------------------------------------- |
| `-in <path>`  | Directory containing input files (required)                                  |
| `-out <path>` | Output PDF path (required). Overwritten if it exists.                        |
| `-work <path>`| Working directory for intermediate files. Defaults to a temporary directory removed on success. |
| `-keep-work`  | Keep the working directory even when no `-work` is specified. Useful for inspecting per-page intermediates. |
| `-h`, `-help` | Print full help text                                                         |

### Exit codes

| Code | Meaning                                          |
| ---- | ------------------------------------------------ |
| 0    | Success                                          |
| 1    | Invalid arguments or runtime error               |
| 2    | No supported input files found in `-in`          |

## expense-sheet: fill the .xlsx claim form

```sh
export ANTHROPIC_API_KEY=sk-ant-...

expense-sheet \
  -in ./receipts \
  -template ./assets/Expenses_Claim_Form_Template_2026.xlsx \
  -out ./2026-q2-claim.xlsx \
  -name "Stefan Kruger" \
  -claim-date 2026-05-15
```

For each receipt, `expense-sheet` sends the file to the Claude API (model `claude-sonnet-4-6`) with a forced `record_expense` tool call. The tool's schema constrains the response to a structured JSON object with five fields: `date`, `vendor`, `currency`, `amount`, and `summary`. PDFs are sent as document blocks; images are sent as image blocks. HEIC inputs are transcoded to JPEG with `sips -Z 2000` first, both for HEIC compatibility and to stay below the Anthropic 5 MB per-image limit.

Extracted rows are written to the .xlsx template:

- The claimant name is placed in B2 and the claim date in D2.
- Each receipt becomes a line item starting at row 7. The date goes in column A, a description (`<vendor> — <summary>`) in column B, and the amount in column D.
- GBP amounts are written to the Total column (D) as numeric values, picked up by the existing `=SUM(D7:D23)` total at row 24.
- Non-GBP amounts are noted in column B as `<vendor> — <summary> [<amount> <CCY>]` and column D is left blank for those rows, per company policy. Conversion to GBP and entry of the converted total is left to the accounts team.

The template provides 17 line-item slots (rows 7 to 23). If more receipts are found, `expense-sheet` exits with an error before contacting the API; split the run into multiple claims.

### Flags

| Flag                 | Description                                                            |
| -------------------- | ---------------------------------------------------------------------- |
| `-in <path>`         | Directory containing receipt files (required)                          |
| `-template <path>`   | Path to the .xlsx claim template (required)                            |
| `-out <path>`        | Output .xlsx path (required). Overwritten if it exists.                |
| `-name "<string>"`   | Claimant name written to cell B2 (required)                            |
| `-claim-date <date>` | Claim date in YYYY-MM-DD format, written to cell D2 (required)         |
| `-h`, `-help`        | Print full help text                                                   |

### Exit codes

| Code | Meaning                                          |
| ---- | ------------------------------------------------ |
| 0    | Success                                          |
| 1    | Invalid arguments or runtime error (including extraction failure or missing `ANTHROPIC_API_KEY`) |
| 2    | No supported input files found in `-in`          |

### Cost and accuracy

Each receipt extraction is a single API call with roughly 2,000 input tokens and 50 output tokens. At Claude Sonnet 4.6 pricing this works out to under a penny per receipt; a full claim of 10 to 15 receipts typically costs less than ten pence. The model is generally reliable on clear airline e-tickets, hotel folios, and photographed till receipts, but the output is not reviewed before being written to the .xlsx. Open the generated file and check it before submitting.

## Typical workflow

```sh
# 1. Drop receipts into a folder, prefixed with the date so they sort in order.
ls ./receipts
#   2026-05-11_arrowcars-out.pdf
#   2026-05-11_brs-tortilla.heic
#   2026-05-11_klm-brs-lin.pdf
#   2026-05-11_paris-extime.heic
#   2026-05-11_milan-hotel.heic
#   2026-05-13_milan-trenord.pdf
#   2026-05-13_arrowcars-home.pdf

# 2. Produce a single PDF of all receipts to attach to the claim.
expense -in ./receipts -out ./receipts/all-receipts.pdf

# 3. Fill the claim form.
expense-sheet \
  -in ./receipts \
  -template ./assets/Expenses_Claim_Form_Template_2026.xlsx \
  -out ./2026-05-15-claim.xlsx \
  -name "Stefan Kruger" \
  -claim-date 2026-05-15

# 4. Open ./2026-05-15-claim.xlsx, sanity-check the rows, and submit
#    alongside ./receipts/all-receipts.pdf.
```

The merged PDF can safely live in the same folder as the source receipts; `expense-sheet` recognises the `expense-merge` marker keyword in its Info dictionary and skips it automatically.

## Licence

MIT. See [LICENSE](LICENSE).
