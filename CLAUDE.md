# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-steg is a CLI tool for image steganography using Least Significant Bit (LSB) manipulation. It embeds images inside carrier images by modifying the last 2 bits of each color channel (R, G, B). It supports an optional indiscernibility mask (password-derived) to make embedded data harder to detect via steganalysis.

## Build & Run

```bash
go install          # Install the binary
go build -o go-steg # Build locally
go test ./...       # Run all tests
go test ./go_steg/image_processing/  # Run tests for a specific package
go test -run TestEncode ./go_steg/image_processing/  # Run a single test
```

## CLI Usage

```bash
# Encode (embed an image into carrier images)
go-steg encode -e <embed_file> -c <carrier1>,<carrier2> -p <password> -o <output_dir> -u

# Decode (extract embedded image from carrier images)
go-steg decode -c <carrier1>,<carrier2> -p <password> -o <output_dir>
```

The `-u` flag enables the indiscernibility mask. Carrier order must match between encode and decode.

## Architecture

### Core Packages

- **`go_steg/image_processing/`** — Central package containing encode/decode logic, image resizing, and header management. This is where the steganography algorithm lives.
- **`go_steg/bit_manipulation/`** — Low-level bit operations: splitting bytes into 2-bit quarters, setting/getting last 2 bits, and mask difference calculations. All steganographic data storage works in 2-bit units.
- **`go_steg/huffman/`** — Huffman tree implementation (work in progress, not yet integrated into the encoding pipeline).
- **`go_steg/file_processing/`** — File saving/resizing helpers for a multipart upload flow (designed for a web API use case).
- **`go_steg/logging/`** — Thin wrapper around `zap.SugaredLogger`.
- **`cli/cmd/`** — Cobra CLI commands (`encode`, `decode`, `files`). Entry point is `main.go` → `cmd.Execute()`.
- **`cli/helpers/`** — CLI validation helpers and the global `UseMask` flag.

### Key Concepts

**2-bit encoding**: Each byte of embed data is split into four 2-bit quarters (`SplitByteIntoQuarters`). Each quarter is stored in the last 2 bits of one color channel of one pixel. So 1 byte of data requires ~4 channel slots (across R, G, B channels of pixels).

**Header layout** (first 13 pixels of column x=0):
- Pixels 0-7: 48-bit photo ID (unique identifier)
- Pixel 8: photo number/order (for multi-carrier)
- Pixels 9-12: 24-bit data count (number of channel slots used)

**Indiscernibility mask**: Generated from a SHA-256 hash of the password, used to seed a deterministic PRNG that produces mask parameters (a 32-bit mask value, multiplier, two index positions, and a change boolean). Only pixels where `ReturnMaskDifference` matches the change boolean are used for data storage.

**Multi-carrier encoding**: Embed data is split into equal chunks across multiple carrier images. Carriers must be decoded in the same order they were encoded.

**Pixel traversal order**: Column-major (x outer loop, y inner loop), starting at y=13 (after header).

### Image Constraints

Instagram-oriented sizing constants: max carrier 1080x1350, embed images capped at half those dimensions (540x675). The `imaging` library (Lanczos filter) handles resizing.
