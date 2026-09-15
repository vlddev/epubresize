package imgresize

import (
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

type Strategy int

const (
	StrategyNone Strategy = iota
	StrategyFixedWidth
	StrategyPercent
)

type Options struct {
	MinWidth int
	Width    int
	Percent  int
	Quality  int
}

type Dimensions struct {
	Width  int
	Height int
}

func (o Options) Strategy() Strategy {
	switch {
	case o.Width > 0:
		return StrategyFixedWidth
	case o.Percent > 0:
		return StrategyPercent
	default:
		return StrategyNone
	}
}

func (o Options) StrategyName() string {
	switch o.Strategy() {
	case StrategyFixedWidth:
		return "fixed width"
	case StrategyPercent:
		return "percentage"
	default:
		return "none"
	}
}

func IsJPEGName(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".jpg" || ext == ".jpeg"
}

func InspectJPEG(r io.Reader) (Dimensions, error) {
	cfg, err := jpeg.DecodeConfig(r)
	if err != nil {
		return Dimensions{}, err
	}
	return Dimensions{Width: cfg.Width, Height: cfg.Height}, nil
}

func TargetDimensions(src Dimensions, opts Options) (Dimensions, bool) {
	if src.Width <= 0 || src.Height <= 0 {
		return src, false
	}
	if src.Width <= opts.MinWidth {
		return src, false
	}

	var target Dimensions
	switch opts.Strategy() {
	case StrategyFixedWidth:
		target.Width = opts.Width
		scale := float64(target.Width) / float64(src.Width)
		target.Height = roundedDimension(float64(src.Height) * scale)
	case StrategyPercent:
		scale := float64(opts.Percent) / 100
		target.Width = roundedDimension(float64(src.Width) * scale)
		target.Height = roundedDimension(float64(src.Height) * scale)
	default:
		return src, false
	}

	target.Width = clampDimension(target.Width, src.Width)
	target.Height = clampDimension(target.Height, src.Height)
	if target.Width == src.Width && target.Height == src.Height {
		return src, false
	}
	return target, true
}

func ResizeJPEG(r io.Reader, w io.Writer, target Dimensions, quality int) error {
	if target.Width <= 0 || target.Height <= 0 {
		return fmt.Errorf("invalid target dimensions %dx%d", target.Width, target.Height)
	}
	if err := validateJPEGQuality(quality); err != nil {
		return err
	}

	src, err := jpeg.Decode(r)
	if err != nil {
		return err
	}

	dst := image.NewRGBA(image.Rect(0, 0, target.Width, target.Height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return jpeg.Encode(w, dst, &jpeg.Options{Quality: quality})
}

func ReencodeJPEG(r io.Reader, w io.Writer, quality int) error {
	if err := validateJPEGQuality(quality); err != nil {
		return err
	}

	src, err := jpeg.Decode(r)
	if err != nil {
		return err
	}
	return jpeg.Encode(w, src, &jpeg.Options{Quality: quality})
}

func ReencodeWithoutResize(src Dimensions, opts Options) bool {
	return src.Width > 0 && src.Height > 0 && opts.MinWidth > 0 && src.Width < opts.MinWidth
}

func roundedDimension(v float64) int {
	n := int(math.Round(v))
	if n < 1 {
		return 1
	}
	return n
}

func clampDimension(target, original int) int {
	if target < 1 {
		return 1
	}
	if target > original {
		return original
	}
	return target
}

func validateJPEGQuality(quality int) error {
	if quality < 1 || quality > 100 {
		return fmt.Errorf("invalid JPEG quality %d", quality)
	}
	return nil
}
