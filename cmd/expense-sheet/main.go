package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

const (
	maxLineItems       = 17
	expenseMergeMarker = "expense-merge"
)

const helpText = `expense-sheet — extract receipt data via Claude and fill an expense claim .xlsx

USAGE
    expense-sheet -in <input-dir> -template <template.xlsx> -out <out.xlsx>
                  -name "<full name>" -claim-date YYYY-MM-DD

DESCRIPTION
    Scans <input-dir> for receipts (.jpg/.jpeg/.png/.heic/.heif/.pdf),
    sends each one to the Claude API to extract {date, vendor, currency,
    amount, summary}, and fills the rows of a company-issued .xlsx
    expense-claim template.

    GBP amounts are written to the Total column. Non-GBP amounts are
    noted in the Details column (with the currency code) and the Total
    column is left blank for those rows, per company policy.

    Requires ANTHROPIC_API_KEY in the environment. HEIC inputs require
    'sips' on $PATH (built in on macOS).

OPTIONS
    -in          Directory of receipt files (required)
    -template    Path to the .xlsx template (required)
    -out         Output .xlsx path (required)
    -name        Claimant name written to B2 (required)
    -claim-date  Claim date YYYY-MM-DD written to D2 (required)
    -h, -help    Show this help text

EXIT CODES
    0  success
    1  invalid arguments or runtime error
    2  no supported input files found
`

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	var inDir, templatePath, outFile, name, claimDateStr string
	for i := 1; i < len(os.Args); i++ {
		switch a := os.Args[i]; a {
		case "-h", "-help", "--help":
			fmt.Print(helpText)
			return
		case "-in":
			inDir = nextArg(i)
			i++
		case "-template":
			templatePath = nextArg(i)
			i++
		case "-out":
			outFile = nextArg(i)
			i++
		case "-name":
			name = nextArg(i)
			i++
		case "-claim-date":
			claimDateStr = nextArg(i)
			i++
		default:
			fmt.Fprintf(os.Stderr, "unknown argument: %s\n\n", a)
			fmt.Fprint(os.Stderr, helpText)
			os.Exit(1)
		}
	}

	if inDir == "" || templatePath == "" || outFile == "" || name == "" || claimDateStr == "" {
		fmt.Fprint(os.Stderr, helpText)
		os.Exit(1)
	}

	claimDate, err := time.Parse("2006-01-02", claimDateStr)
	if err != nil {
		log.Fatalf("error: -claim-date must be YYYY-MM-DD: %v", err)
	}

	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		log.Fatalf("error: ANTHROPIC_API_KEY is not set in the environment")
	}

	receipts, err := scanInputs(inDir, outFile)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if len(receipts) == 0 {
		fmt.Fprintln(os.Stderr, "error: no supported input files found")
		os.Exit(2)
	}
	if len(receipts) > maxLineItems {
		log.Fatalf("error: %d receipts found, but the template has only %d line-item slots — split into multiple claims", len(receipts), maxLineItems)
	}

	if needsSips(receipts) {
		if _, err := exec.LookPath("sips"); err != nil {
			log.Fatalf("error: HEIC/HEIF input detected but 'sips' was not found on $PATH.\n" +
				"Convert these files to JPEG or PNG by other means, then re-run.")
		}
	}

	client := anthropic.NewClient()
	ctx := context.Background()

	expenses := make([]Expense, 0, len(receipts))
	for _, r := range receipts {
		fmt.Fprintf(os.Stderr, "extracting %s ... ", filepath.Base(r))
		exp, err := extractExpense(ctx, &client, r)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed")
			log.Fatalf("error: extract %s: %v", r, err)
		}
		fmt.Fprintf(os.Stderr, "%s %s %.2f %s\n", exp.Date, exp.Vendor, exp.Amount, exp.Currency)
		expenses = append(expenses, exp)
	}

	if err := fillTemplate(templatePath, outFile, name, claimDate, expenses); err != nil {
		log.Fatalf("error: fill template: %v", err)
	}

	fmt.Printf("wrote %s (%d expenses)\n", outFile, len(expenses))
}

func nextArg(i int) string {
	if i+1 >= len(os.Args) {
		fmt.Fprintf(os.Stderr, "error: flag %s requires a value\n", os.Args[i])
		os.Exit(1)
	}
	return os.Args[i+1]
}

func scanInputs(inDir, outFile string) ([]string, error) {
	entries, err := os.ReadDir(inDir)
	if err != nil {
		return nil, fmt.Errorf("read input dir: %w", err)
	}
	outAbs, err := filepath.Abs(outFile)
	if err != nil {
		return nil, fmt.Errorf("resolve output path: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	var receipts []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		switch ext {
		case ".pdf", ".jpg", ".jpeg", ".png", ".heic", ".heif":
		default:
			continue
		}
		src := filepath.Join(inDir, e.Name())
		if srcAbs, err := filepath.Abs(src); err == nil && srcAbs == outAbs {
			continue
		}
		if ext == ".pdf" && isExpenseMergeOutput(src) {
			fmt.Fprintf(os.Stderr, "skipping %s (produced by expense-merge)\n", e.Name())
			continue
		}
		receipts = append(receipts, src)
	}
	return receipts, nil
}

func isExpenseMergeOutput(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	kws, err := api.Keywords(f, nil)
	if err != nil {
		return false
	}
	for _, k := range kws {
		if strings.Contains(k, expenseMergeMarker) {
			return true
		}
	}
	return false
}

func needsSips(paths []string) bool {
	for _, p := range paths {
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".heic" || ext == ".heif" {
			return true
		}
	}
	return false
}
