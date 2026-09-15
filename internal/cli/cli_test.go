package cli

import "testing"

func TestParseArgsFixedWidth(t *testing.T) {
	config, err := ParseArgs([]string{"--min-width", "1600", "--width", "1200", "--quality", "75", "input.epub", "output.epub"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if config.EPUB.Image.MinWidth != 1600 {
		t.Fatalf("MinWidth = %d, want 1600", config.EPUB.Image.MinWidth)
	}
	if config.EPUB.Image.Width != 1200 {
		t.Fatalf("Width = %d, want 1200", config.EPUB.Image.Width)
	}
	if config.EPUB.Image.Quality != 75 {
		t.Fatalf("Quality = %d, want 75", config.EPUB.Image.Quality)
	}
	if config.EPUB.DryRun {
		t.Fatal("DryRun = true, want false")
	}
}

func TestParseArgsDryRun(t *testing.T) {
	config, err := ParseArgs([]string{"--dry-run", "--percent", "60", "input.epub"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if !config.EPUB.DryRun {
		t.Fatal("DryRun = false, want true")
	}
	if config.EPUB.OutputPath != "" {
		t.Fatalf("OutputPath = %q, want empty", config.EPUB.OutputPath)
	}
}

func TestParseArgsRejectsInvalidCombinations(t *testing.T) {
	tests := [][]string{
		{"input.epub", "output.epub"},
		{"--width", "1200", "--percent", "60", "input.epub", "output.epub"},
		{"--percent", "100", "input.epub", "output.epub"},
		{"--percent", "0", "input.epub", "output.epub"},
		{"--quality", "0", "--width", "1200", "input.epub", "output.epub"},
		{"--width", "1200", "book.epub", "book.epub"},
		{"--dry-run", "--width", "1200", "input.epub", "output.epub"},
	}

	for _, args := range tests {
		if _, err := ParseArgs(args); err == nil {
			t.Fatalf("ParseArgs(%v) returned nil error", args)
		}
	}
}
