# Product Requirements Document

## EPUB JPEG Resizer

**Status:** Draft
**Target platform:** Linux
**Implementation language:** Go
**Product type:** Command-line utility
**Working name:** `epubresize`

---

## 1. Product Overview

`epubresize` is a Linux command-line tool that reduces the size of EPUB books by resizing JPEG images contained inside the EPUB archive.

The tool reads an existing EPUB file, identifies JPEG images stored inside it, selectively resizes images according to user-configurable rules, re-encodes resized images at a configurable JPEG quality, and writes the result to a user-specified EPUB file.

Files that are not JPEG images must be preserved unchanged.

The tool must never upscale images.

A dry-run mode is a desired additional feature that calculates the expected size of the converted EPUB without writing the converted EPUB to disk.

---

## 2. Problem Statement

EPUB books, particularly illustrated books, comics, manga, graphic novels, and scanned books, can contain very large JPEG images.

Often these images have a resolution significantly higher than required by the target reading device. This results in unnecessarily large EPUB files and increased storage requirements.

Users need a simple Linux command-line utility that can:

* find JPEG images inside an EPUB;
* resize only images above a configurable minimum width;
* resize images to a fixed width or by a percentage;
* control JPEG output quality;
* produce a new EPUB at a user-specified location;
* report the amount of space saved.

The tool should operate without requiring users to extract and repack EPUB files manually.

---

## 3. Goals

### 3.1 Primary goals

The tool must:

1. Accept an EPUB file as input.
2. Find JPEG images inside the EPUB archive.
3. Allow the user to specify a minimum JPEG width for resizing.
4. Support fixed-width resizing.
5. Support percentage-based resizing.
6. Allow JPEG output quality to be configured.
7. Allow the output EPUB filename/path to be configured.
8. Preserve non-JPEG EPUB contents.
9. Never enlarge an image.
10. Produce a valid EPUB archive.
11. Work as a Linux command-line application.
12. Be implemented in Go.

### 3.2 Secondary goals

The tool should:

* provide useful progress/statistics output;
* report the number of images found and resized;
* report original and resulting EPUB sizes;
* preserve EPUB metadata where practical;
* fail safely without destroying the original EPUB;
* support large EPUB files without unnecessarily loading the complete archive into memory.

### 3.3 Nice-to-have goal

Provide a `--dry-run` mode that performs the image conversions in memory and calculates the resulting EPUB size without creating the output EPUB.

---

## 4. Non-Goals

The initial version will not:

* resize PNG, GIF, WebP, AVIF, or other image formats;
* convert non-JPEG images to JPEG;
* upscale images;
* modify EPUB HTML/XHTML/CSS;
* modify EPUB metadata;
* modify image orientation metadata unless required by the image-processing library;
* modify the EPUB directory structure;
* optimize arbitrary ZIP files;
* provide a graphical user interface;
* modify the input EPUB in place.

---

# 5. User Stories

## US-001 — Resize images to a fixed width

As a user, I want to resize large JPEG images to a fixed width so that an EPUB occupies less storage space.

Example:

```bash
epubresize \
    --min-width 1600 \
    --width 1200 \
    book.epub \
    book-small.epub
```

Expected behavior:

* JPEGs with width <= 1600 pixels are left unchanged.
* JPEGs wider than 1600 pixels are resized to 1200 pixels wide.
* Image aspect ratio is preserved.
* Images are never enlarged.

---

## US-002 — Resize images proportionally

As a user, I want to reduce images by a percentage so that their original relative sizes are maintained.

Example:

```bash
epubresize \
    --min-width 1600 \
    --percent 60 \
    book.epub \
    book-small.epub
```

For a 3000 × 4000 image:

```text
3000 × 4000
     ↓ 60%
1800 × 2400
```

---

## US-003 — Configure JPEG quality

As a user, I want to control JPEG output quality.

Example:

```bash
epubresize \
    --width 1200 \
    --quality 85 \
    book.epub \
    book-small.epub
```

Quality must be configurable from 1 to 100.

A sensible default should be:

```text
85
```

---

## US-004 — Avoid resizing small images

As a user, I want to define a minimum width so that small images and thumbnails are not unnecessarily recompressed.

Example:

```bash
--min-width 1600
```

An image of:

```text
1600 × 2400
```

must remain unchanged.

An image of:

```text
2000 × 3000
```

may be resized.

---

## US-005 — Select output EPUB

As a user, I want to specify where the resulting EPUB is written.

Example:

```bash
epubresize \
    --width 1200 \
    input.epub \
    /books/output/book-small.epub
```

The input EPUB must never be overwritten by default.

---

## US-006 — Preview expected savings

As a user, I want to know approximately how much space will be saved before creating an EPUB.

Example:

```bash
epubresize \
    --min-width 1600 \
    --width 1200 \
    --quality 85 \
    --dry-run \
    book.epub
```

The tool should report:

```text
Original EPUB:  184.7 MiB
Result EPUB:     71.4 MiB
Difference:    -113.3 MiB (-61.3%)

94 JPEGs would be resized.
93 JPEGs would remain unchanged.

No output file created.
```

The dry-run should perform the actual resize and JPEG encoding in memory so that the resulting size calculation is meaningful.

---

# 6. Command-Line Interface

## 6.1 Basic syntax

```text
epubresize [options] input.epub output.epub
```

Example:

```bash
epubresize book.epub book-small.epub
```

---

## 6.2 Options

### `--min-width`

Minimum JPEG width that triggers resizing.

```text
--min-width <pixels>
```

Example:

```bash
--min-width 1600
```

Default:

```text
0
```

Semantics:

```text
width <= min-width
    → don't resize

width > min-width
    → resize
```

---

### `--width`

Resize qualifying images to a fixed width.

```text
--width <pixels>
```

Example:

```bash
--width 1200
```

`--width` and `--percent` are mutually exclusive.

---

### `--percent`

Resize qualifying images to a percentage of their original width.

```text
--percent <percentage>
```

Example:

```bash
--percent 60
```

Valid range:

```text
0 < percent < 100
```

The initial version should reject values >= 100 because the primary purpose of the tool is image reduction.

---

### `--quality`

JPEG output quality.

```text
--quality <1-100>
```

Default:

```text
85
```

Example:

```bash
--quality 90
```

Only JPEGs that are actually resized should be re-encoded.

JPEGs below the minimum width should be copied unchanged.

---

### `--dry-run`

Calculate the resulting EPUB size without creating an output file.

```bash
--dry-run
```

When specified, only the input EPUB is required:

```bash
epubresize \
    --min-width 1600 \
    --width 1200 \
    --quality 85 \
    --dry-run \
    book.epub
```

---

### `--verbose`

Display information about each resized image.

Example:

```text
resize images/page001.jpg  2400x3200 -> 1200x1600
resize images/page002.jpg  3000x4000 -> 1200x1600
```

---

# 7. Resize Rules

## 7.1 JPEG detection

The tool must recognize:

```text
.jpg
.jpeg
```

case-insensitively.

Examples:

```text
page001.jpg
page002.jpeg
PAGE003.JPG
```

The file extension should be used as the initial selection mechanism.

The implementation may additionally validate that the file is actually a JPEG before processing.

---

## 7.2 Minimum width

For:

```text
--min-width 1600
```

the behavior must be:

| Original width | Resize? |
| -------------: | :-----: |
|            800 |    No   |
|           1200 |    No   |
|           1600 |    No   |
|           1601 |   Yes   |
|           2400 |   Yes   |
|           4000 |   Yes   |

---

## 7.3 Fixed-width strategy

For:

```text
--width 1200
```

an image:

```text
2400 × 3200
```

becomes:

```text
1200 × 1600
```

An image:

```text
3000 × 2000
```

becomes:

```text
1200 × 800
```

The original aspect ratio must always be preserved.

---

## 7.4 Percentage strategy

For:

```text
--percent 60
```

an image:

```text
3000 × 4000
```

becomes approximately:

```text
1800 × 2400
```

The implementation must use appropriate rounding when calculating the resulting dimensions.

---

## 7.5 No upscaling

The tool must never increase image dimensions.

For example:

```bash
--width 2000
```

must not turn:

```text
1200 × 1800
```

into:

```text
2000 × 3000
```

The image must remain unchanged.

---

# 8. EPUB Handling

## 8.1 Input

The input must be a valid EPUB/ZIP archive.

The tool should validate that the archive contains the required EPUB `mimetype` entry.

The original EPUB must not be modified.

---

## 8.2 Output

The tool creates a new EPUB.

All non-JPEG entries must be preserved.

JPEG entries that do not meet the resize criteria must preferably be copied without re-encoding.

JPEG entries that are resized are replaced by their newly encoded JPEG data.

---

## 8.3 EPUB metadata

The following should be preserved where possible:

* entry names;
* directory structure;
* modification timestamps;
* ZIP permissions;
* compression method where practical;
* EPUB metadata files;
* XHTML/HTML;
* CSS;
* fonts;
* SVG;
* cover metadata;
* navigation documents.

The tool must not modify EPUB content references.

Since image filenames remain unchanged, no XHTML/CSS modification should be required.

---

## 8.4 `mimetype`

The EPUB `mimetype` entry has special ZIP requirements.

The implementation must ensure that the resulting EPUB remains compliant with the EPUB format.

Specifically, the tool must preserve/create the EPUB `mimetype` entry correctly:

```text
application/epub+zip
```

It must remain the first ZIP entry and must not be compressed.

---

# 9. Output File Safety

The tool must not overwrite the input file by default.

This must be rejected:

```bash
epubresize book.epub book.epub
```

unless a future explicit option such as:

```text
--in-place
```

is introduced.

The output should first be written to a temporary file in the destination directory.

Only after successful completion should the temporary file be renamed to the requested output filename.

If conversion fails:

```text
input.epub
```

must remain untouched.

---

# 10. Dry-Run Requirements

Dry-run is a nice-to-have feature but should be designed into the architecture from the beginning.

The conversion pipeline should be separated from the output destination.

Conceptually:

```text
                 JPEG processing
                       │
             ┌─────────┴─────────┐
             │                   │
       real ZIP writer     counting writer
             │                   │
       output.epub          --dry-run
```

In dry-run mode:

1. Read the EPUB.
2. Identify JPEGs.
3. Determine which images require resizing.
4. Decode qualifying JPEGs.
5. Resize them.
6. Encode them at the requested quality.
7. Feed the resulting data into a ZIP writer.
8. Count the resulting bytes.
9. Do not create the output EPUB.

This gives a substantially more accurate result than estimating the size from the original JPEG sizes.

Example output:

```text
EPUB resize preview

Input:            book.epub

Resize strategy:  fixed width
Target width:     1200 px
Minimum width:    1600 px
JPEG quality:     85

JPEG files:       187
Would resize:      94
Would preserve:    93

Original size:    184.73 MiB
Result size:       71.42 MiB

Difference:      -113.31 MiB
Reduction:        61.34%

Dry run: no output file created.
```

---

# 11. Statistics

After a successful conversion, the tool should report:

* number of JPEGs found;
* number resized;
* number left unchanged;
* original EPUB size;
* output EPUB size;
* size difference;
* percentage reduction.

Example:

```text
EPUB resize complete

JPEG files found:      187
JPEG files resized:     94
JPEG files unchanged:   93

Original EPUB:       184.73 MiB
Output EPUB:          71.42 MiB

Reduction:           113.31 MiB (61.34%)

Created: book-small.epub
```

---

# 12. Image Processing

## 12.1 Go libraries

The initial implementation should use:

* Go standard library `archive/zip`
* Go standard library `image/jpeg`
* Go standard library image types
* `golang.org/x/image/draw` for high-quality resizing

The architecture should isolate image processing behind an interface so that the implementation can later be replaced by another JPEG/image library if necessary.

---

## 12.2 JPEG quality

JPEG quality must be passed to the encoder.

The valid range is:

```text
1–100
```

Default:

```text
85
```

---

## 12.3 Color handling

The tool should preserve the source image's color information as far as supported by the selected JPEG decoder/encoder.

The initial version does not need advanced color-management functionality.

---

## 12.4 Orientation

The initial version should document its handling of EXIF orientation.

If the chosen Go image pipeline does not preserve EXIF metadata, this should be explicitly documented.

A future version may provide orientation/metadata preservation.

---

# 13. Performance Requirements

The tool should be designed to process large EPUB files efficiently.

### Requirements

* Do not load the entire EPUB into memory.
* Process ZIP entries individually.
* Do not fully decode JPEGs that are below the minimum width.
* Use JPEG configuration/dimension inspection before full decoding.
* Stream the generated EPUB to disk.
* Use a temporary output file.
* Avoid unnecessary JPEG re-encoding.

Expected memory usage should be approximately:

```text
O(largest JPEG being processed)
```

rather than:

```text
O(total EPUB size)
```

---

# 14. Error Handling

The tool must return a non-zero exit code on failure.

Errors should clearly identify the affected file where applicable.

Examples:

```text
error: input file does not exist
```

```text
error: invalid EPUB archive
```

```text
error: EPUB mimetype entry is invalid
```

```text
error: --width and --percent are mutually exclusive
```

```text
error: JPEG decode failed for images/page042.jpg
```

```text
error: cannot create output file
```

No partial output should be left behind after an unsuccessful conversion.

---

# 15. Exit Codes

Recommended initial exit codes:

| Code | Meaning                        |
| ---: | ------------------------------ |
|    0 | Success                        |
|    1 | General conversion error       |
|    2 | Invalid command-line arguments |
|    3 | Invalid input EPUB             |
|    4 | Output file error              |

The implementation may initially use standard Go error handling with exit code `1`, with more granular codes introduced if useful.

---

# 16. Compatibility

The initial target is:

```text
Linux amd64
```

The application should nevertheless avoid platform-specific APIs so that it can later be compiled for:

```text
Linux arm64
Linux armv7
FreeBSD
macOS
Windows
```

The preferred distribution model is a statically compiled executable where practical.

Example:

```bash
go build -o epubresize .
```

---

# 17. Technical Architecture

Recommended package structure:

```text
epubresize/
│
├── cmd/
│   └── epubresize/
│       └── main.go
│
├── epub/
│   ├── reader.go
│   ├── writer.go
│   └── validate.go
│
├── image/
│   └── resize.go
│
├── stats/
│   └── stats.go
│
├── go.mod
├── go.sum
└── README.md
```

For a very small implementation, this can initially be simplified to:

```text
epubresize/
├── main.go
├── epub.go
├── image.go
├── go.mod
└── README.md
```

---

# 18. Processing Pipeline

```text
                   input.epub
                       │
                       ▼
                Open ZIP archive
                       │
                       ▼
                Validate EPUB
                       │
                       ▼
                Iterate entries
                       │
          ┌────────────┴────────────┐
          │                         │
       JPEG?                    Not JPEG
          │                         │
         Yes                        │
          │                         │
    DecodeConfig                    │
          │                         │
    ┌─────┴─────┐                   │
    │           │                   │
 small       large                  │
    │           │                   │
 copy       resize                  │
 unchanged      │                   │
    │       JPEG encode             │
    │           │                   │
    └──────┬────┴───────────────────┘
           │
           ▼
      ZIP writer
           │
           ▼
      output.epub
```

For dry-run:

```text
                         ZIP writer
                             │
                             ▼
                     counting writer
                             │
                             ▼
                    calculated size
```

---

# 19. CLI Validation Rules

The following combinations must be rejected:

### Neither resize strategy specified

```bash
epubresize book.epub output.epub
```

unless a future default strategy is introduced.

### Both strategies specified

```bash
epubresize \
    --width 1200 \
    --percent 60 \
    book.epub output.epub
```

must fail.

### Invalid percentage

```bash
--percent 100
--percent 150
--percent 0
--percent -20
```

must fail.

### Invalid quality

```bash
--quality 0
--quality 101
```

must fail.

### Same input and output

```bash
epubresize book.epub book.epub
```

must fail.

---

# 20. Example Usage

## Fixed width

```bash
epubresize \
    --min-width 1600 \
    --width 1200 \
    --quality 85 \
    input.epub \
    output.epub
```

## Percentage

```bash
epubresize \
    --min-width 1600 \
    --percent 60 \
    --quality 85 \
    input.epub \
    output.epub
```

## Dry run

```bash
epubresize \
    --min-width 1600 \
    --width 1200 \
    --quality 85 \
    --dry-run \
    input.epub
```

## Verbose

```bash
epubresize \
    --min-width 1600 \
    --width 1200 \
    --quality 85 \
    --verbose \
    input.epub \
    output.epub
```

---

# 21. Acceptance Criteria

## AC-001 — Basic conversion

Given a valid EPUB containing JPEG images, when the user executes:

```bash
epubresize --width 1200 input.epub output.epub
```

then:

* `output.epub` is created;
* the input remains unchanged;
* JPEGs wider than 1200 pixels are resized to 1200 pixels;
* aspect ratios are preserved.

---

## AC-002 — Minimum width

Given:

```bash
--min-width 1600
```

then a JPEG with width 1600 pixels is unchanged and a JPEG with width 1601 pixels is eligible for resizing.

---

## AC-003 — Percentage resize

Given a 3000 × 4000 JPEG and:

```bash
--percent 50
```

the resulting JPEG dimensions must be approximately:

```text
1500 × 2000
```

---

## AC-004 — Quality

Given:

```bash
--quality 75
```

resized JPEGs must be encoded using JPEG quality 75.

---

## AC-005 — No upscaling

Given a 1000-pixel-wide image and:

```bash
--width 1200
```

the image must not be enlarged.

---

## AC-006 — Non-JPEG preservation

Given an EPUB containing:

```text
.xhtml
.css
.png
.svg
.ttf
.otf
.jpg
```

all non-JPEG entries must remain present in the resulting EPUB.

---

## AC-007 — Unmodified JPEG preservation

A JPEG below the configured minimum width must not be decoded/re-encoded and should be copied unchanged.

---

## AC-008 — EPUB validity

The resulting file must remain a valid EPUB archive and retain the required `mimetype` entry.

---

## AC-009 — Output safety

If conversion fails, the requested output file must not contain a partially generated EPUB.

---

## AC-010 — Dry run

With:

```bash
--dry-run
```

the tool must not create an output EPUB.

It must nevertheless calculate the resulting archive size by running the conversion pipeline against a non-persistent output/counting destination.

---

# 22. Testing Strategy

Tests should include:

### Unit tests

* command-line validation;
* fixed-width calculation;
* percentage calculation;
* aspect-ratio preservation;
* minimum-width filtering;
* no-upscaling behavior;
* JPEG detection;
* byte-size formatting.

### Integration tests

Create test EPUBs containing:

```text
mimetype
META-INF/container.xml
OEBPS/content.xhtml
OEBPS/style.css
OEBPS/images/small.jpg
OEBPS/images/large.jpg
OEBPS/images/image.png
```

Verify:

* EPUB opens as ZIP;
* `mimetype` is valid;
* small JPEG remains unchanged;
* large JPEG is resized;
* PNG remains unchanged;
* XHTML remains unchanged;
* output is readable by an EPUB reader.

### Regression tests

Test:

* JPEG with portrait orientation;
* landscape JPEG;
* very large JPEG;
* tiny JPEG;
* uppercase `.JPG`;
* `.jpeg`;
* corrupt JPEG;
* corrupt ZIP;
* EPUB with no JPEGs;
* EPUB with thousands of images;
* output path in a nonexistent directory;
* input/output path collision.

---

# 23. Future Features

Potential future options include:

```text
--max-width
--min-height
--min-megapixels
--threads
--workers
--keep-metadata
--strip-metadata
--progress
--in-place
--output-dir
```

Additional image formats could eventually be supported:

```text
PNG
WebP
AVIF
```

A future version could also support choosing the JPEG resampling algorithm.

---

# 24. MVP Definition

The MVP is complete when the following command works reliably:

```bash
epubresize \
    --min-width 1600 \
    --width 1200 \
    --quality 85 \
    input.epub \
    output.epub
```

and the following also works:

```bash
epubresize \
    --min-width 1600 \
    --percent 60 \
    --quality 85 \
    input.epub \
    output.epub
```

The MVP must:

* run on Linux;
* be implemented in Go;
* process JPEGs inside EPUB archives;
* support minimum-width filtering;
* support fixed-width resizing;
* support percentage resizing;
* support JPEG quality;
* preserve non-JPEG EPUB content;
* prevent upscaling;
* produce a valid EPUB;
* safely create a new output file.

`--dry-run` size prediction is a **Phase 2 / nice-to-have feature**, but the architecture should support it without requiring a major rewrite.

---

# 25. Recommended Default Behavior

For the first release:

```text
JPEG quality:       85
Minimum width:       0
Resize strategy:     explicitly required
Upscaling:           never
Input modification:  never
Output:              explicitly specified
Dry run:             optional
Verbose output:      optional
```

This makes the tool conservative by default and minimizes the possibility of unexpectedly degrading or modifying an EPUB.

---

# 26. Success Metrics

The tool is successful if a user can take a large EPUB such as:

```text
book.epub       450 MB
```

and run:

```bash
epubresize \
    --min-width 1600 \
    --width 1200 \
    --quality 85 \
    book.epub \
    book-optimized.epub
```

to obtain a significantly smaller EPUB while:

1. preserving the book's content;
2. preserving its EPUB structure;
3. maintaining readable image quality;
4. requiring no manual extraction/repacking;
5. requiring no external image-processing command-line utilities;
6. completing reliably on Linux.

