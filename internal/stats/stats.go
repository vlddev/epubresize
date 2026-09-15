package stats

import (
	"fmt"
	"io"
	"math"
)

type Stats struct {
	InputPath  string
	OutputPath string
	DryRun     bool

	StrategyName string
	TargetWidth  int
	Percent      int
	MinWidth     int
	Quality      int

	JPEGFiles       int
	JPEGResized     int
	JPEGQualityOnly int

	OriginalSize int64
	ResultSize   int64
}

func (s Stats) JPEGUnchanged() int {
	return s.JPEGFiles - s.JPEGResized - s.JPEGQualityOnly
}

func (s Stats) DifferenceBytes() int64 {
	return s.ResultSize - s.OriginalSize
}

func (s Stats) SavedBytes() int64 {
	return s.OriginalSize - s.ResultSize
}

func (s Stats) ReductionPercent() float64 {
	if s.OriginalSize <= 0 {
		return 0
	}
	return float64(s.SavedBytes()) / float64(s.OriginalSize) * 100
}

func (s Stats) PrintPreview(w io.Writer) {
	fmt.Fprintln(w, "EPUB resize preview")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Input:            %s\n", s.InputPath)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Resize strategy:  %s\n", s.StrategyName)
	if s.TargetWidth > 0 {
		fmt.Fprintf(w, "Target width:     %d px\n", s.TargetWidth)
	}
	if s.Percent > 0 {
		fmt.Fprintf(w, "Target percent:   %d%%\n", s.Percent)
	}
	fmt.Fprintf(w, "Minimum width:    %d px\n", s.MinWidth)
	fmt.Fprintf(w, "JPEG quality:     %d\n", s.Quality)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "JPEG files:          %d\n", s.JPEGFiles)
	fmt.Fprintf(w, "Would resize:        %d\n", s.JPEGResized)
	fmt.Fprintf(w, "Would quality-only:  %d\n", s.JPEGQualityOnly)
	fmt.Fprintf(w, "Would preserve:      %d\n", s.JPEGUnchanged())
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Original size:    %s\n", FormatBytes(s.OriginalSize))
	fmt.Fprintf(w, "Result size:      %s\n", FormatBytes(s.ResultSize))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Difference:       %s\n", FormatBytes(s.DifferenceBytes()))
	fmt.Fprintf(w, "Reduction:        %.2f%%\n", s.ReductionPercent())
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Dry run: no output file created.")
}

func (s Stats) PrintComplete(w io.Writer) {
	fmt.Fprintln(w, "EPUB resize complete")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "JPEG files found:     %d\n", s.JPEGFiles)
	fmt.Fprintf(w, "JPEG files resized:   %d\n", s.JPEGResized)
	fmt.Fprintf(w, "JPEG quality-only:    %d\n", s.JPEGQualityOnly)
	fmt.Fprintf(w, "JPEG files unchanged: %d\n", s.JPEGUnchanged())
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Original EPUB:         %s\n", FormatBytes(s.OriginalSize))
	fmt.Fprintf(w, "Output EPUB:           %s\n", FormatBytes(s.ResultSize))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Reduction:             %s (%.2f%%)\n", FormatBytes(s.SavedBytes()), s.ReductionPercent())
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Created: %s\n", s.OutputPath)
}

func FormatBytes(n int64) string {
	sign := ""
	value := float64(n)
	if value < 0 {
		sign = "-"
		value = math.Abs(value)
	}

	const (
		kib = 1024
		mib = kib * 1024
		gib = mib * 1024
	)

	switch {
	case value >= gib:
		return fmt.Sprintf("%s%.2f GiB", sign, value/gib)
	case value >= mib:
		return fmt.Sprintf("%s%.2f MiB", sign, value/mib)
	case value >= kib:
		return fmt.Sprintf("%s%.2f KiB", sign, value/kib)
	default:
		return fmt.Sprintf("%s%d B", sign, int64(value))
	}
}
