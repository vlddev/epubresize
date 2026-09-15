package epub

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"epubresize/internal/imgresize"
	"epubresize/internal/stats"
)

const epubMimetype = "application/epub+zip"

var (
	ErrInvalidEPUB = errors.New("invalid EPUB archive")
	ErrOutputFile  = errors.New("output file error")
)

type ImageProcessor interface {
	InspectJPEG(io.Reader) (imgresize.Dimensions, error)
	ResizeJPEG(io.Reader, io.Writer, imgresize.Dimensions, int) error
	ReencodeJPEG(io.Reader, io.Writer, int) error
}

type DefaultImageProcessor struct{}

func (DefaultImageProcessor) InspectJPEG(r io.Reader) (imgresize.Dimensions, error) {
	return imgresize.InspectJPEG(r)
}

func (DefaultImageProcessor) ResizeJPEG(r io.Reader, w io.Writer, target imgresize.Dimensions, quality int) error {
	return imgresize.ResizeJPEG(r, w, target, quality)
}

func (DefaultImageProcessor) ReencodeJPEG(r io.Reader, w io.Writer, quality int) error {
	return imgresize.ReencodeJPEG(r, w, quality)
}

type Options struct {
	InputPath  string
	OutputPath string
	DryRun     bool

	Image         imgresize.Options
	VerboseWriter io.Writer
}

type Processor struct {
	ImageProcessor ImageProcessor
}

func NewProcessor() *Processor {
	return &Processor{ImageProcessor: DefaultImageProcessor{}}
}

func (p *Processor) Process(opts Options) (stats.Stats, error) {
	if p.ImageProcessor == nil {
		p.ImageProcessor = DefaultImageProcessor{}
	}

	result := stats.Stats{
		InputPath:    opts.InputPath,
		OutputPath:   opts.OutputPath,
		DryRun:       opts.DryRun,
		StrategyName: opts.Image.StrategyName(),
		TargetWidth:  opts.Image.Width,
		Percent:      opts.Image.Percent,
		MinWidth:     opts.Image.MinWidth,
		Quality:      opts.Image.Quality,
	}

	info, err := os.Stat(opts.InputPath)
	if err != nil {
		return result, fmt.Errorf("input file %q: %w", opts.InputPath, err)
	}
	if info.IsDir() {
		return result, fmt.Errorf("input file %q is a directory", opts.InputPath)
	}
	result.OriginalSize = info.Size()

	reader, err := zip.OpenReader(opts.InputPath)
	if err != nil {
		return result, fmt.Errorf("%w: %s: %v", ErrInvalidEPUB, opts.InputPath, err)
	}
	defer reader.Close()

	mimetype, err := findMimetype(&reader.Reader)
	if err != nil {
		return result, err
	}
	if err := validateMimetype(mimetype); err != nil {
		return result, err
	}

	if opts.DryRun {
		counter := &countingWriter{}
		zipWriter := zip.NewWriter(counter)
		if err := p.writeArchive(&reader.Reader, zipWriter, mimetype, opts, &result); err != nil {
			_ = zipWriter.Close()
			return result, err
		}
		if err := zipWriter.Close(); err != nil {
			return result, err
		}
		result.ResultSize = counter.N
		return result, nil
	}

	tempFile, tempPath, err := createTempOutput(opts.OutputPath)
	if err != nil {
		return result, err
	}

	keepTemp := false
	defer func() {
		if !keepTemp {
			_ = os.Remove(tempPath)
		}
	}()

	zipWriter := zip.NewWriter(tempFile)
	writeErr := p.writeArchive(&reader.Reader, zipWriter, mimetype, opts, &result)
	closeZipErr := zipWriter.Close()
	closeFileErr := tempFile.Close()

	if writeErr != nil {
		return result, writeErr
	}
	if closeZipErr != nil {
		return result, fmt.Errorf("%w: %v", ErrOutputFile, closeZipErr)
	}
	if closeFileErr != nil {
		return result, fmt.Errorf("%w: %v", ErrOutputFile, closeFileErr)
	}
	if err := os.Rename(tempPath, opts.OutputPath); err != nil {
		return result, fmt.Errorf("%w: rename %q to %q: %v", ErrOutputFile, tempPath, opts.OutputPath, err)
	}
	keepTemp = true

	outputInfo, err := os.Stat(opts.OutputPath)
	if err != nil {
		return result, fmt.Errorf("%w: stat %q: %v", ErrOutputFile, opts.OutputPath, err)
	}
	result.ResultSize = outputInfo.Size()
	return result, nil
}

func (p *Processor) writeArchive(reader *zip.Reader, writer *zip.Writer, mimetype *zip.File, opts Options, result *stats.Stats) error {
	if err := writeMimetype(writer, mimetype); err != nil {
		return err
	}

	for _, file := range reader.File {
		if file.Name == "mimetype" {
			continue
		}
		if file.FileInfo().IsDir() || !imgresize.IsJPEGName(file.Name) {
			if err := copyEntry(writer, file); err != nil {
				return err
			}
			continue
		}
		if err := p.processJPEG(writer, file, opts, result); err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) processJPEG(writer *zip.Writer, file *zip.File, opts Options, result *stats.Stats) error {
	result.JPEGFiles++

	configReader, err := file.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", file.Name, err)
	}
	dimensions, inspectErr := p.ImageProcessor.InspectJPEG(configReader)
	closeErr := configReader.Close()
	if inspectErr != nil {
		return fmt.Errorf("JPEG decode failed for %s: %w", file.Name, inspectErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", file.Name, closeErr)
	}

	target, shouldResize := imgresize.TargetDimensions(dimensions, opts.Image)
	if !shouldResize {
		if imgresize.ReencodeWithoutResize(dimensions, opts.Image) {
			return p.reencodeJPEG(writer, file, opts, result)
		}
		return copyEntry(writer, file)
	}

	entryWriter, err := writer.CreateHeader(cloneFileHeader(file))
	if err != nil {
		return fmt.Errorf("%w: create entry %s: %v", ErrOutputFile, file.Name, err)
	}

	imageReader, err := file.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", file.Name, err)
	}
	resizeErr := p.ImageProcessor.ResizeJPEG(imageReader, entryWriter, target, opts.Image.Quality)
	closeErr = imageReader.Close()
	if resizeErr != nil {
		return fmt.Errorf("JPEG decode failed for %s: %w", file.Name, resizeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", file.Name, closeErr)
	}

	result.JPEGResized++
	if opts.VerboseWriter != nil {
		fmt.Fprintf(opts.VerboseWriter, "resize %s  %dx%d -> %dx%d\n", file.Name, dimensions.Width, dimensions.Height, target.Width, target.Height)
	}
	return nil
}

func (p *Processor) reencodeJPEG(writer *zip.Writer, file *zip.File, opts Options, result *stats.Stats) error {
	entryWriter, err := writer.CreateHeader(cloneFileHeader(file))
	if err != nil {
		return fmt.Errorf("%w: create entry %s: %v", ErrOutputFile, file.Name, err)
	}

	imageReader, err := file.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", file.Name, err)
	}
	reencodeErr := p.ImageProcessor.ReencodeJPEG(imageReader, entryWriter, opts.Image.Quality)
	closeErr := imageReader.Close()
	if reencodeErr != nil {
		return fmt.Errorf("JPEG decode failed for %s: %w", file.Name, reencodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", file.Name, closeErr)
	}

	result.JPEGQualityOnly++
	if opts.VerboseWriter != nil {
		fmt.Fprintf(opts.VerboseWriter, "quality %s\n", file.Name)
	}
	return nil
}

func copyEntry(writer *zip.Writer, file *zip.File) error {
	entryWriter, err := writer.CreateHeader(cloneFileHeader(file))
	if err != nil {
		return fmt.Errorf("%w: create entry %s: %v", ErrOutputFile, file.Name, err)
	}
	if file.FileInfo().IsDir() {
		return nil
	}

	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", file.Name, err)
	}
	defer reader.Close()

	if _, err := io.Copy(entryWriter, reader); err != nil {
		return fmt.Errorf("%w: write entry %s: %v", ErrOutputFile, file.Name, err)
	}
	return nil
}

func cloneFileHeader(file *zip.File) *zip.FileHeader {
	header := &zip.FileHeader{
		Name:     file.Name,
		Method:   file.Method,
		Comment:  file.Comment,
		Extra:    sanitizedExtra(file.Extra),
		NonUTF8:  file.NonUTF8,
		Modified: file.Modified,
	}
	header.SetMode(file.Mode())
	return header
}

func sanitizedExtra(extra []byte) []byte {
	if len(extra) == 0 {
		return nil
	}

	var out []byte
	remaining := extra
	for len(remaining) >= 4 {
		id := binary.LittleEndian.Uint16(remaining[0:2])
		size := int(binary.LittleEndian.Uint16(remaining[2:4]))
		if len(remaining[4:]) < size {
			return append([]byte(nil), extra...)
		}

		field := remaining[:4+size]
		if id != 0x0001 && id != 0x5455 {
			out = append(out, field...)
		}
		remaining = remaining[4+size:]
	}
	if len(remaining) > 0 {
		return append([]byte(nil), extra...)
	}
	return out
}

func writeMimetype(writer *zip.Writer, source *zip.File) error {
	header := &zip.FileHeader{
		Name:     "mimetype",
		Method:   zip.Store,
		Modified: source.Modified,
		NonUTF8:  source.NonUTF8,
	}
	header.SetMode(source.Mode())

	entryWriter, err := writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("%w: create mimetype entry: %v", ErrOutputFile, err)
	}
	if _, err := io.WriteString(entryWriter, epubMimetype); err != nil {
		return fmt.Errorf("%w: write mimetype entry: %v", ErrOutputFile, err)
	}
	return nil
}

func findMimetype(reader *zip.Reader) (*zip.File, error) {
	for _, file := range reader.File {
		if file.Name == "mimetype" {
			return file, nil
		}
	}
	return nil, fmt.Errorf("%w: missing mimetype entry", ErrInvalidEPUB)
}

func validateMimetype(file *zip.File) error {
	if file.FileInfo().IsDir() {
		return fmt.Errorf("%w: mimetype entry is a directory", ErrInvalidEPUB)
	}
	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("%w: cannot read mimetype entry: %v", ErrInvalidEPUB, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, int64(len(epubMimetype)+1)))
	if err != nil {
		return fmt.Errorf("%w: cannot read mimetype entry: %v", ErrInvalidEPUB, err)
	}
	if string(data) != epubMimetype {
		return fmt.Errorf("%w: EPUB mimetype entry is invalid", ErrInvalidEPUB)
	}
	return nil
}

func createTempOutput(outputPath string) (*os.File, string, error) {
	if strings.TrimSpace(outputPath) == "" {
		return nil, "", fmt.Errorf("%w: output path is required", ErrOutputFile)
	}
	destinationDir := filepath.Dir(outputPath)
	if info, err := os.Stat(destinationDir); err != nil {
		return nil, "", fmt.Errorf("%w: destination directory %q: %v", ErrOutputFile, destinationDir, err)
	} else if !info.IsDir() {
		return nil, "", fmt.Errorf("%w: destination %q is not a directory", ErrOutputFile, destinationDir)
	}

	file, err := os.CreateTemp(destinationDir, ".epubresize-*.epub")
	if err != nil {
		return nil, "", fmt.Errorf("%w: cannot create temporary output in %q: %v", ErrOutputFile, destinationDir, err)
	}
	return file, file.Name(), nil
}

type countingWriter struct {
	N int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.N += int64(len(p))
	return len(p), nil
}
