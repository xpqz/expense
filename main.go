package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"golang.org/x/image/draw"
)

const (
	maxLongEdge        = 2000 // px
	jpegQuality        = 80
	a4WidthMM          = 210.0
	a4HeightMM         = 297.0
	expenseMergeMarker = "expense-merge"
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
        .pdf         passed through as-is, included in the merge
        .jpg/.jpeg   converted to a single A4 PDF page
        .png         same as .jpg
        .heic/.heif  iPhone format; requires 'sips' (macOS built-in)
                     to transcode to PNG first, then same as .jpg

    Files with other extensions, hidden files, and subdirectories are ignored.
    The directory is not scanned recursively.

IMAGE PROCESSING
    Each image is:
      1. Downscaled so its longest edge is at most 2000px (preserves aspect ratio).
      2. Rotated 90° clockwise if the pixels are landscape (width > height),
         so phone photos of receipts taken in portrait orientation read
         upright on the page.
      3. Converted to 8-bit grayscale.
      4. Re-encoded as JPEG at quality 80.
      5. Placed on an A4 portrait page, scaled to fit while preserving aspect
         ratio, and centred.

    Existing PDF files are not modified at the image level by Go. To
    actually shrink airline-receipt PDFs (which often wrap a large scan),
    the merged file is fed through Ghostscript as a final pass if 'gs' is
    on $PATH: PDFSETTINGS=/ebook with images downsampled to 150 DPI.
    Without Ghostscript installed, only the pdfcpu structural optimisation
    runs and large embedded images stay large.

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

	outAbs, err := filepath.Abs(outFile)
	if err != nil {
		log.Fatalf("error: resolve output path: %v", err)
	}

	usingTempDir := workDir == ""
	if usingTempDir {
		workDir, err = os.MkdirTemp("", "expense-*")
		if err != nil {
			log.Fatalf("error: create temp dir: %v", err)
		}
	} else if err := os.MkdirAll(workDir, 0o755); err != nil {
		log.Fatalf("error: create work dir: %v", err)
	}
	if usingTempDir && !keepWork {
		defer os.RemoveAll(workDir)
	}

	entries, err := os.ReadDir(inDir)
	if err != nil {
		log.Fatalf("error: read input dir: %v", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var sipsPath string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".heic" || ext == ".heif" {
			p, err := exec.LookPath("sips")
			if err != nil {
				log.Fatalf("error: HEIC/HEIF input detected (%s) but 'sips' was not found on $PATH.\n"+
					"Convert these files to JPEG or PNG by other means, then re-run.", e.Name())
			}
			sipsPath = p
			break
		}
	}

	var pdfPaths []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		src := filepath.Join(inDir, e.Name())
		if srcAbs, err := filepath.Abs(src); err == nil && srcAbs == outAbs {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))

		if ext == ".pdf" {
			pdfPaths = append(pdfPaths, src)
			continue
		}

		imgSrc := src
		switch ext {
		case ".jpg", ".jpeg", ".png":
			// use src directly
		case ".heic", ".heif":
			converted := filepath.Join(workDir, strings.TrimSuffix(e.Name(), ext)+".png")
			if err := sipsToPNG(sipsPath, src, converted); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skip %s: %v\n", src, err)
				continue
			}
			imgSrc = converted
		default:
			continue
		}

		out := filepath.Join(workDir, strings.TrimSuffix(e.Name(), ext)+".pdf")
		if err := imageToPDF(imgSrc, out); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skip %s: %v\n", src, err)
			continue
		}
		pdfPaths = append(pdfPaths, out)
	}

	if len(pdfPaths) == 0 {
		fmt.Fprintln(os.Stderr, "error: no supported input files found")
		os.Exit(2)
	}

	merged := filepath.Join(workDir, "_merged.pdf")
	if err := api.MergeCreateFile(pdfPaths, merged, false, nil); err != nil {
		log.Fatalf("error: merge: %v", err)
	}

	optimized := filepath.Join(workDir, "_optimized.pdf")
	if err := api.OptimizeFile(merged, optimized, nil); err != nil {
		log.Fatalf("error: optimise: %v", err)
	}

	if gs, err := exec.LookPath("gs"); err == nil {
		if err := runGhostscript(gs, optimized, outFile); err != nil {
			log.Fatalf("error: ghostscript: %v", err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "warning: 'gs' not on $PATH; embedded PDF images not recompressed")
		if err := copyFile(optimized, outFile); err != nil {
			log.Fatalf("error: write output: %v", err)
		}
	}

	if err := api.AddKeywordsFile(outFile, outFile, []string{expenseMergeMarker}, nil); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not tag output PDF: %v\n", err)
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
	if b := img.Bounds(); b.Dx() > b.Dy() {
		img = rotate90CW(img)
	}
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
	long := max(w, h)
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

func rotate90CW(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := range h {
		for x := range w {
			dst.Set(h-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func sipsToPNG(sips, src, dst string) error {
	cmd := exec.Command(sips, "-s", "format", "png", src, "--out", dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sips: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runGhostscript(gs, in, out string) error {
	cmd := exec.Command(gs,
		"-sDEVICE=pdfwrite",
		"-dCompatibilityLevel=1.5",
		"-dPDFSETTINGS=/ebook",
		"-dDownsampleColorImages=true", "-dColorImageResolution=150",
		"-dDownsampleGrayImages=true", "-dGrayImageResolution=150",
		"-dDownsampleMonoImages=true", "-dMonoImageResolution=300",
		"-dNOPAUSE", "-dQUIET", "-dBATCH",
		"-sOutputFile="+out,
		in,
	)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()
	_, err = io.Copy(d, s)
	return err
}
