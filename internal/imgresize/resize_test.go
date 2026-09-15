package imgresize

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestIsJPEGName(t *testing.T) {
	tests := map[string]bool{
		"page.jpg":        true,
		"page.jpeg":       true,
		"PAGE.JPG":        true,
		"cover.JPEG":      true,
		"image.png":       false,
		"jpg-without-ext": false,
	}

	for name, want := range tests {
		if got := IsJPEGName(name); got != want {
			t.Fatalf("IsJPEGName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestTargetDimensionsFixedWidth(t *testing.T) {
	got, ok := TargetDimensions(
		Dimensions{Width: 3000, Height: 2000},
		Options{Width: 1200},
	)
	if !ok {
		t.Fatal("expected resize")
	}
	want := Dimensions{Width: 1200, Height: 800}
	if got != want {
		t.Fatalf("TargetDimensions() = %+v, want %+v", got, want)
	}
}

func TestTargetDimensionsPercent(t *testing.T) {
	got, ok := TargetDimensions(
		Dimensions{Width: 3000, Height: 4000},
		Options{Percent: 50},
	)
	if !ok {
		t.Fatal("expected resize")
	}
	want := Dimensions{Width: 1500, Height: 2000}
	if got != want {
		t.Fatalf("TargetDimensions() = %+v, want %+v", got, want)
	}
}

func TestTargetDimensionsMinimumWidth(t *testing.T) {
	opts := Options{MinWidth: 1600, Width: 1200}

	if _, ok := TargetDimensions(Dimensions{Width: 1600, Height: 2400}, opts); ok {
		t.Fatal("width equal to min-width must not resize")
	}
	if _, ok := TargetDimensions(Dimensions{Width: 1601, Height: 2400}, opts); !ok {
		t.Fatal("width greater than min-width should resize")
	}
}

func TestTargetDimensionsNoUpscale(t *testing.T) {
	if _, ok := TargetDimensions(
		Dimensions{Width: 1000, Height: 1500},
		Options{Width: 1200},
	); ok {
		t.Fatal("fixed width larger than source must not resize")
	}
}

func TestReencodeWithoutResize(t *testing.T) {
	opts := Options{MinWidth: 1600, Width: 1200}

	if !ReencodeWithoutResize(Dimensions{Width: 1599, Height: 2400}, opts) {
		t.Fatal("width less than min-width should be re-encoded")
	}
	if ReencodeWithoutResize(Dimensions{Width: 1600, Height: 2400}, opts) {
		t.Fatal("width equal to min-width should not be re-encoded")
	}
	if ReencodeWithoutResize(Dimensions{Width: 1599, Height: 2400}, Options{Width: 1200}) {
		t.Fatal("min-width must be set before quality-only re-encoding")
	}
}

func TestReencodeJPEGPreservesDimensions(t *testing.T) {
	source := mustTestJPEG(t, 16, 8, 90)

	var out bytes.Buffer
	if err := ReencodeJPEG(bytes.NewReader(source), &out, 40); err != nil {
		t.Fatalf("ReencodeJPEG() error = %v", err)
	}
	if bytes.Equal(out.Bytes(), source) {
		t.Fatal("re-encoded JPEG bytes should differ from source")
	}

	got, err := InspectJPEG(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("InspectJPEG() error = %v", err)
	}
	want := Dimensions{Width: 16, Height: 8}
	if got != want {
		t.Fatalf("re-encoded dimensions = %+v, want %+v", got, want)
	}
}

func mustTestJPEG(t *testing.T, width, height, quality int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 9), G: uint8(y * 13), B: 140, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}
