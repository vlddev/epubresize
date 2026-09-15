package epub

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"testing"

	"epubresize/internal/imgresize"
)

func TestProcessResizesJPEGAndPreservesEPUB(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.epub")
	outputPath := filepath.Join(dir, "output.epub")

	smallJPEG := mustJPEG(t, 8, 4)
	largeJPEG := mustJPEG(t, 20, 10)
	pngBytes := []byte("not actually a png, but should be copied")
	cssBytes := []byte("body { color: black; }")

	writeTestEPUB(t, inputPath, map[string][]byte{
		"META-INF/container.xml":    []byte("<container/>"),
		"OEBPS/style.css":           cssBytes,
		"OEBPS/images/small.jpg":    smallJPEG,
		"OEBPS/images/large.JPG":    largeJPEG,
		"OEBPS/images/keep.png":     pngBytes,
		"OEBPS/chapter.xhtml":       []byte("<html><img src='images/large.JPG'/></html>"),
		"OEBPS/images/photo.jpeg":   smallJPEG,
		"OEBPS/images/not-jpeg.txt": []byte("hello"),
	})

	result, err := NewProcessor().Process(Options{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Image: imgresize.Options{
			MinWidth: 10,
			Width:    10,
			Quality:  85,
		},
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.JPEGFiles != 3 {
		t.Fatalf("JPEGFiles = %d, want 3", result.JPEGFiles)
	}
	if result.JPEGResized != 1 {
		t.Fatalf("JPEGResized = %d, want 1", result.JPEGResized)
	}
	if result.JPEGQualityOnly != 2 {
		t.Fatalf("JPEGQualityOnly = %d, want 2", result.JPEGQualityOnly)
	}
	if result.JPEGUnchanged() != 0 {
		t.Fatalf("JPEGUnchanged = %d, want 0", result.JPEGUnchanged())
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("output is not a readable zip: %v", err)
	}
	defer reader.Close()

	if len(reader.File) == 0 || reader.File[0].Name != "mimetype" {
		t.Fatal("mimetype must be the first ZIP entry")
	}
	if reader.File[0].Method != zip.Store {
		t.Fatalf("mimetype method = %d, want Store", reader.File[0].Method)
	}
	if got := readZipEntry(t, &reader.Reader, "mimetype"); string(got) != epubMimetype {
		t.Fatalf("mimetype = %q", got)
	}
	if got := readZipEntry(t, &reader.Reader, "OEBPS/style.css"); !bytes.Equal(got, cssBytes) {
		t.Fatal("CSS entry changed")
	}
	if got := readZipEntry(t, &reader.Reader, "OEBPS/images/keep.png"); !bytes.Equal(got, pngBytes) {
		t.Fatal("PNG entry changed")
	}
	if got := readZipEntry(t, &reader.Reader, "OEBPS/images/small.jpg"); bytes.Equal(got, smallJPEG) {
		t.Fatal("small JPEG should be re-encoded at requested quality")
	} else if dimensions := jpegDimensions(t, got); dimensions != (imgresize.Dimensions{Width: 8, Height: 4}) {
		t.Fatalf("small JPEG dimensions = %+v, want 8x4", dimensions)
	}
	if got := jpegDimensions(t, readZipEntry(t, &reader.Reader, "OEBPS/images/large.JPG")); got != (imgresize.Dimensions{Width: 10, Height: 5}) {
		t.Fatalf("large JPEG dimensions = %+v, want 10x5", got)
	}
}

func TestProcessDryRunCreatesNoOutput(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.epub")
	outputPath := filepath.Join(dir, "output.epub")

	writeTestEPUB(t, inputPath, map[string][]byte{
		"OEBPS/images/large.jpg": mustJPEG(t, 20, 10),
	})

	result, err := NewProcessor().Process(Options{
		InputPath:  inputPath,
		OutputPath: outputPath,
		DryRun:     true,
		Image: imgresize.Options{
			Width:   10,
			Quality: 85,
		},
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.ResultSize <= 0 {
		t.Fatalf("ResultSize = %d, want positive", result.ResultSize)
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("dry run created output path or stat failed unexpectedly: %v", err)
	}
}

func TestProcessRejectsInvalidMimetype(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.epub")
	writeZip(t, inputPath, []zipEntry{{Name: "mimetype", Method: zip.Store, Body: []byte("text/plain")}})

	_, err := NewProcessor().Process(Options{
		InputPath:  inputPath,
		OutputPath: filepath.Join(dir, "output.epub"),
		Image:      imgresize.Options{Width: 10, Quality: 85},
	})
	if err == nil {
		t.Fatal("Process() returned nil error")
	}
}

func writeTestEPUB(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	zipEntries := []zipEntry{{Name: "mimetype", Method: zip.Store, Body: []byte(epubMimetype)}}
	for name, body := range entries {
		zipEntries = append(zipEntries, zipEntry{Name: name, Method: zip.Deflate, Body: body})
	}
	writeZip(t, path, zipEntries)
}

type zipEntry struct {
	Name   string
	Method uint16
	Body   []byte
}

func writeZip(t *testing.T, path string, entries []zipEntry) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test zip: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.Name, Method: entry.Method}
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("create entry %s: %v", entry.Name, err)
		}
		if _, err := entryWriter.Write(entry.Body); err != nil {
			t.Fatalf("write entry %s: %v", entry.Name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close test zip: %v", err)
	}
}

func mustJPEG(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 11), B: 120, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func readZipEntry(t *testing.T, reader *zip.Reader, name string) []byte {
	t.Helper()

	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", name, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read zip entry %s: %v", name, err)
		}
		return data
	}
	t.Fatalf("missing zip entry %s", name)
	return nil
}

func jpegDimensions(t *testing.T, data []byte) imgresize.Dimensions {
	t.Helper()

	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode jpeg config: %v", err)
	}
	return imgresize.Dimensions{Width: cfg.Width, Height: cfg.Height}
}
