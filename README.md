# epubresize

`epubresize` is a Linux command-line tool for reducing EPUB size by resizing JPEG images inside the EPUB archive.

It preserves non-JPEG files unchanged, never upscales images, and writes successful conversions through a temporary file before moving the final EPUB into place.

## Build

```bash
go build -buildvcs=false -o epubresize ./cmd/epubresize
```

## Usage

Resize qualifying JPEGs to a fixed width:

```bash
epubresize --min-width 300 --width 300 --quality 40 input.epub output.epub
```

Resize qualifying JPEGs proportionally:

```bash
epubresize --min-width 1600 --percent 60 --quality 85 input.epub output.epub
```

Preview the result without creating an output file:

```bash
epubresize --dry-run --min-width 1600 --width 1200 --quality 85 input.epub
```

Print each processed JPEG:

```bash
epubresize --verbose --width 1200 input.epub output.epub
```

## Options

`--width` and `--percent` are mutually exclusive, and one of them is required.

| Option | Description | Default |
| --- | --- | --- |
| `--min-width <pixels>` | Only JPEGs wider than this are eligible for resizing. | `0` |
| `--width <pixels>` | Resize eligible JPEGs to this fixed width. | none |
| `--percent <1-99>` | Resize eligible JPEGs to this percentage of original dimensions. | none |
| `--quality <1-100>` | JPEG quality used for resized and quality-only JPEG output. | `85` |
| `--dry-run` | Run the full conversion into an in-memory counting writer. | `false` |
| `--verbose` | Print each processed JPEG. | `false` |

## EPUB Handling

The input must be a ZIP/EPUB archive with a root `mimetype` entry containing exactly:

```text
application/epub+zip
```

The output EPUB writes `mimetype` as the first ZIP entry and stores it uncompressed, as required by EPUB readers.

JPEG files are selected by `.jpg` or `.jpeg` extension, case-insensitively. Files that are selected as JPEGs are validated through Go's JPEG decoder before resizing.

## Metadata Note

The current Go image pipeline decodes and re-encodes resized JPEG pixels. JPEGs narrower than `--min-width` are re-encoded at the requested quality without resizing. JPEG metadata such as EXIF, including EXIF orientation, is not preserved on resized or re-encoded images.
