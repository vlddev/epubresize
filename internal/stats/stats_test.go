package stats

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := map[int64]string{
		0:               "0 B",
		512:             "512 B",
		1536:            "1.50 KiB",
		2 * 1024 * 1024: "2.00 MiB",
		-1536:           "-1.50 KiB",
	}

	for input, want := range tests {
		if got := FormatBytes(input); got != want {
			t.Fatalf("FormatBytes(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestReductionPercent(t *testing.T) {
	stats := Stats{OriginalSize: 200, ResultSize: 50}
	if got := stats.ReductionPercent(); got != 75 {
		t.Fatalf("ReductionPercent() = %v, want 75", got)
	}
}

func TestJPEGUnchangedAccountsForQualityOnlyReencoding(t *testing.T) {
	stats := Stats{JPEGFiles: 5, JPEGResized: 2, JPEGQualityOnly: 1}
	if got := stats.JPEGUnchanged(); got != 2 {
		t.Fatalf("JPEGUnchanged() = %d, want 2", got)
	}
}
