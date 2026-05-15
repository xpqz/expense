# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository. The user's name is Stefan.

@docs/claude/claude-go.md

## Code Intelligence

Prefer LSP over Grep/Glob/Read for code navigation:

- `goToDefinition` / `goToImplementation` to jump to source
- `findReferences` to see all usages across the codebase
- `workspaceSymbol` to find where something is defined
- `documentSymbol` to list all symbols in a file
- `hover` for type info without reading the file
- `incomingCalls` / `outgoingCalls` for call hierarchy

Before renaming or changing a function signature, use `findReferences` to find all call sites first.

Use Grep/Glob only for text/pattern searches (comments, strings, config values) where LSP doesn't help.

After writing or editing code, check LSP diagnostics before moving on. Fix any type errors or missing imports immediately.

## Project

`expense` — a single-file Go CLI (`main.go`, module `github.com/stefan/expense`) that scans a directory of receipts (PDFs + JPG/JPEG/PNG), converts images to A4 PDF pages, merges everything into one PDF, and runs a structural optimisation pass. Dependencies: `go-pdf/fpdf`, `pdfcpu`, `golang.org/x/image/draw`.

## Commands

Use the Makefile for common tasks:

```
make build           # go build -o expense .
make run ARGS="-in ./receipts -out ./out.pdf"
make fmt vet test tidy clean
```

Direct invocation also works: `go run . -in ./receipts -out ./out.pdf`. There are no tests yet — run a single test with `go test -run TestName ./...` once you add one.

## Architecture notes

The whole pipeline lives in `main.go`. Two things are worth knowing before editing:

- **Order is filename-lexicographic**, not mtime. Inputs are sorted by `entries[i].Name()` (main.go:146) before processing, and the merge preserves that order. Users are expected to prefix filenames with dates if order matters — don't "fix" this by switching to mtime without checking.
- **External tools.** Two optional shell-outs: `gs` (Ghostscript, recommended — recompresses embedded images in the final PDF; falls back to pdfcpu-only with a stderr warning if missing) and `sips` (always present on macOS — transcodes HEIC/HEIF iPhone photos to PNG before the image pipeline runs). HEIC support is fail-fast: if any `.heic`/`.heif` input is detected and `sips` is missing from `$PATH`, the run aborts immediately with exit 1 before any other work happens, telling the user to convert by other means.
- **Image pipeline is in-process; PDF recompression is delegated to Ghostscript.** `imageToPDF` downscales to `maxLongEdge=2000px`, converts to 8-bit grayscale, and re-encodes JPEG at quality 80 before placing on A4. After merge + `api.OptimizeFile` (which only does structural dedup), the file is fed through `gs -sDEVICE=pdfwrite -dPDFSETTINGS=/ebook` at 150 DPI to actually shrink embedded images inside input PDFs. If `gs` is missing from `$PATH`, the tool falls back to the pdfcpu-optimised file (with a stderr warning) and large embedded PDF images stay large.

The working directory (`-work` or a temp dir) holds per-image intermediate PDFs plus `_merged.pdf`. It's removed on success unless `-keep-work` or an explicit `-work` path is given — useful when debugging a bad page.

Exit codes are load-bearing for scripting: `0` success, `1` arg/runtime error, `2` no supported inputs found.
