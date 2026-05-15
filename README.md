# expense

Combine receipt images (JPG/JPEG/PNG) and PDFs in a directory into a single optimised PDF. Images are downscaled, converted to grayscale, and re-encoded as JPEG before being placed on A4 pages; the merged PDF is then run through a structural optimisation pass.

## Install

```sh
go install github.com/stefan/expense@latest
```

Or build locally:

```sh
make build
```

## Usage

```sh
expense -in ./receipts -out ./2026-q1.pdf
```

Files are processed in lexicographic order by filename — prefix with dates (e.g. `2026-01-15_coffee.jpg`) if order matters. Run `expense -h` for the full flag list.

## Notes

- Existing PDF inputs are passed through untouched; the optimisation pass deduplicates objects but does **not** recompress embedded images.
- The working directory (`-work`) is removed on success unless `-keep-work` is set or an explicit `-work` path is given.
