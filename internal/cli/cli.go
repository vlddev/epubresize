package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"epubresize/internal/epub"
	"epubresize/internal/imgresize"
)

const (
	ExitSuccess     = 0
	ExitGeneral     = 1
	ExitInvalidArgs = 2
	ExitInvalidEPUB = 3
	ExitOutputFile  = 4
)

var (
	ErrInvalidArgs = errors.New("invalid command-line arguments")
	ErrHelp        = errors.New("help requested")
)

type Config struct {
	EPUB    epub.Options
	Verbose bool
}

func Run(args []string, stdout, stderr io.Writer) int {
	config, err := ParseArgs(args)
	if err != nil {
		if errors.Is(err, ErrHelp) {
			PrintUsage(stdout)
			return ExitSuccess
		}
		if errors.Is(err, ErrInvalidArgs) {
			fmt.Fprintf(stderr, "error: %s\n\n", trimWrappedMessage(err, ErrInvalidArgs))
			PrintUsage(stderr)
			return ExitInvalidArgs
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitGeneral
	}

	if config.Verbose {
		config.EPUB.VerboseWriter = stdout
	}

	processor := epub.NewProcessor()
	result, err := processor.Process(config.EPUB)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		switch {
		case errors.Is(err, epub.ErrInvalidEPUB):
			return ExitInvalidEPUB
		case errors.Is(err, epub.ErrOutputFile):
			return ExitOutputFile
		default:
			return ExitGeneral
		}
	}

	if result.DryRun {
		result.PrintPreview(stdout)
	} else {
		result.PrintComplete(stdout)
	}
	return ExitSuccess
}

func ParseArgs(args []string) (Config, error) {
	fs := flag.NewFlagSet("epubresize", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	minWidth := fs.Int("min-width", 0, "minimum JPEG width that triggers resizing")
	width := fs.Int("width", 0, "resize qualifying images to this width")
	percent := fs.Int("percent", 0, "resize qualifying images to this percentage")
	quality := fs.Int("quality", 85, "JPEG quality from 1 to 100")
	dryRun := fs.Bool("dry-run", false, "calculate output size without writing an EPUB")
	verbose := fs.Bool("verbose", false, "display each processed image")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Config{}, ErrHelp
		}
		return Config{}, argError(err.Error())
	}

	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		seen[f.Name] = true
	})

	if *minWidth < 0 {
		return Config{}, argError("--min-width must be >= 0")
	}
	if seen["width"] && *width <= 0 {
		return Config{}, argError("--width must be > 0")
	}
	if seen["percent"] && (*percent <= 0 || *percent >= 100) {
		return Config{}, argError("--percent must be greater than 0 and less than 100")
	}
	if *quality < 1 || *quality > 100 {
		return Config{}, argError("--quality must be between 1 and 100")
	}
	if seen["width"] && seen["percent"] {
		return Config{}, argError("--width and --percent are mutually exclusive")
	}
	if !seen["width"] && !seen["percent"] {
		return Config{}, argError("either --width or --percent is required")
	}

	positionals := fs.Args()
	switch {
	case *dryRun && len(positionals) != 1:
		return Config{}, argError("--dry-run expects exactly one input EPUB argument")
	case !*dryRun && len(positionals) != 2:
		return Config{}, argError("expected input.epub and output.epub")
	}

	inputPath := positionals[0]
	outputPath := ""
	if !*dryRun {
		outputPath = positionals[1]
		if samePath(inputPath, outputPath) {
			return Config{}, argError("input and output paths must be different")
		}
	}

	config := Config{
		EPUB: epub.Options{
			InputPath:  inputPath,
			OutputPath: outputPath,
			DryRun:     *dryRun,
			Image: imgresize.Options{
				MinWidth: *minWidth,
				Width:    *width,
				Percent:  *percent,
				Quality:  *quality,
			},
		},
		Verbose: *verbose,
	}
	return config, nil
}

func PrintUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  epubresize --width <pixels> [options] input.epub output.epub
  epubresize --percent <1-99> [options] input.epub output.epub
  epubresize --dry-run --width <pixels> [options] input.epub
  epubresize --dry-run --percent <1-99> [options] input.epub

Options:
  --min-width <pixels>  Minimum JPEG width that triggers resizing (default 0)
  --width <pixels>      Resize qualifying JPEGs to a fixed width
  --percent <1-99>      Resize qualifying JPEGs by a percentage
  --quality <1-100>     JPEG output quality (default 85)
  --dry-run             Calculate resulting EPUB size without writing output
  --verbose             Print each processed JPEG
`)
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	if filepath.Clean(absA) == filepath.Clean(absB) {
		return true
	}

	infoA, statErrA := os.Stat(absA)
	infoB, statErrB := os.Stat(absB)
	if statErrA == nil && statErrB == nil {
		return os.SameFile(infoA, infoB)
	}
	return false
}

func argError(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidArgs, message)
}

func trimWrappedMessage(err error, sentinel error) string {
	prefix := sentinel.Error() + ": "
	return strings.TrimPrefix(err.Error(), prefix)
}
