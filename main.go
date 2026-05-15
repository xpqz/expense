package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"golang.org/x/image/draw"
)

const (
	maxLongEdge = 2000 // px
	jpegQuality = 80
	a4WidthMM   = 210.0
	a4HeightMM  = 297.0
)

const helpText = `expense — combine receipt images and PDFs into a single optimised PDF

USAGE
    expense -in <input-dir> -out <output-file> [options]

DESCRIPTION
    Scans <input-dir> for receipt files, converts any images to PDF pages,
    merges everything (originals + converted) into a single PDF at <output-file>,
    and runs a structural optimisation pass to reduce file size.

    Files are processed in lexicographic order by filename. Sort your inputs
    accordingly (e.g. prefix with dates: "2026-01-15_coffee.jpg") if order matters.

    Supported input extensions (case-insensitive):
        .pdf   passed through as-is, included in the merge
        .jpg   converted to a single A4 PDF page
        .jpeg  same as .jpg
        .png   same as .jpg

    Files with other extensions, hidden files, and subdirectories are ignored.
    The directory is not scanned recursively.

IMAGE PROCESSING
    Each image is:
      1. Downscaled so its longest edge is at most 2000px (preserves aspect ratio).
      2. Converted to 8-bit grayscale.
      3. Re-encoded as JPEG at quality 80.
      4. Placed on an A4 portrait page, scaled to fit while preserving aspect
         ratio, and centred. Landscape images will be letterboxed top/bottom;
         portrait images fill nearly the full page.

    Existing PDF files are NOT recompressed or downscaled. If an input PDF
    contains large embedded images, they will remain large in the output.
    The optimisation pass only deduplicates objects and cleans structure.

OPTIONS
    -in <path>      (required) Directory containing input files.
    -out <path>     (required) Output PDF file path. Overwritten if it exists.
    -work <path>    Working directory for intermediate files. Defaults to a
                    temp directory which is removed on success. If specified,
                    the directory is kept (useful for debugging).
    -keep-work      Keep the working directory even if -work is not specified.
    -h, -help       Show this help text.

EXIT CODES
    0  success
    1  invalid arguments or runtime error (details on stderr)
    2  no supported input files found in <input-dir>

EXAMPLES
    Basic use:
        expense -in ./receipts -out ./2026-q1.pdf

    Keep intermediate files for inspection:
        expense -in ./receipts -out ./out.pdf -work ./debug

OUTPUT
    On success, prints a single line to stdout:
        wrote <output-path> (<N> source files, <M> bytes)

    Errors and per-file warnings (e.g. an unreadable image) are written to
    stderr. A warning does not abort the run; the file is skipped.
`

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	var (
		inDir    string
		outFile  string
		workDir  string
		keepWork bool
	)

	flag.StringVar(&inDir, "in", "", "input directory (required)")
	flag.StringVar(&outFile, "out", "", "output PDF path (required)")
	flag.StringVar(&workDir, "work", "", "working directory (default: temp dir, auto-removed)")
	flag.BoolVar(&keepWork, "keep-work", false, "keep working directory after success")
	flag.Usage = func() { fmt.Fprint(os.Stderr, helpText) }
	flag.Parse()

	if inDir == "" || outFile == "" {
		fmt.Fprint(os.Stderr, helpText)
		os.Exit(1)
	}

	info, err := os.Stat(inDir)
	if err != nil {
		log.Fatalf("error: input directory: %v", err)
	}
	if !info.IsDir() {
		log.Fatalf("error: -in must be a directory: %s", inDir)
	}

	cleanupWork := false
	if workDir == "" {
		workDir, err = os.MkdirTemp("", "expense-*")
		if err != nil {
			log.Fatalf("error: create temp dir: %v", err)
		}
		cleanupWork = !keepWork
	} else {
		if err := os.MkdirAll(workDir, 0o755); err != nil {
			log.Fatalf("error: create work dir: %v", err)
		}
	}
	defer func() {
		if cleanupWork {
			_ = os.RemoveAll(workDir)
		}
	}()

	entries, err := os.ReadDir(inDir)
	if err != nil {
		log.Fatalf("error: read input dir: %v", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var pdfPaths []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		src := filepath.Join(inDir, e.Name())
		ext := strings.ToLower(filepath.Ext(e.Name()))

		switch ext {
		case ".pdf":
			pdfPaths = append(pdfPaths, src)
		case ".jpg", ".jpeg", ".png":
			out := filepath.Join(workDir, strings.TrimSuffix(e.Name(), ext)+".pdf")
			if err := imageToPDF(src, out); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skip %s: %v\n", src, err)
				continue
			}
			pdfPaths = append(pdfPaths, out)
		}
	}

	if len(pdfPaths) == 0 {
		fmt.Fprintln(os.Stderr, "error: no supported input files found")
		os.Exit(2)
	}

	merged := filepath.Join(workDir, "_merged.pdf")
	if err := api.MergeCreateFile(pdfPaths, merged, false, nil); err != nil {
		log.Fatalf("error: merge: %v", err)
	}

	if err := api.OptimizeFile(merged, outFile, nil); err != nil {
		log.Fatalf("error: optimise: %v", err)
	}

	st, err := os.Stat(outFile)
	if err != nil {
		log.Fatalf("error: stat output: %v", err)
	}
	fmt.Printf("wrote %s (%d source files, %d bytes)\n", outFile, len(pdfPaths), st.Size())
}

func imageToPDF(srcPath, dstPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	img = downscale(img, maxLongEdge)
	img = toGray(img)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return err
	}

	b := img.Bounds()
	imgW, imgH := float64(b.Dx()), float64(b.Dy())
	scale := a4WidthMM / imgW
	if a4HeightMM/imgH < scale {
		scale = a4HeightMM / imgH
	}
	drawW := imgW * scale
	drawH := imgH * scale
	x := (a4WidthMM - drawW) / 2
	y := (a4HeightMM - drawH) / 2

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	opt := fpdf.ImageOptions{ImageType: "JPEG", ReadDpi: false}
	pdf.RegisterImageOptionsReader(srcPath, opt, &buf)
	pdf.ImageOptions(srcPath, x, y, drawW, drawH, false, opt, 0, "")

	return pdf.OutputFileAndClose(dstPath)
}

func downscale(src image.Image, maxEdge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	long := w
	if h > long {
		long = h
	}
	if long <= maxEdge {
		return src
	}
	s := float64(maxEdge) / float64(long)
	dst := image.NewRGBA(image.Rect(0, 0, int(float64(w)*s), int(float64(h)*s)))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

func toGray(src image.Image) image.Image {
	b := src.Bounds()
	gray := image.NewGray(b)
	draw.Draw(gray, b, src, b.Min, draw.Src)
	return gray
}
