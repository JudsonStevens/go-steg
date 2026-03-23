# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-steg is a CLI tool for image steganography using Least Significant Bit (LSB) manipulation. It hides arbitrary files inside PNG and JPEG carrier images by modifying the least significant bits of each color channel (R, G, B). Features include multi-carrier splitting, variable bit depth (1-4 bits), Huffman compression, Reed-Solomon error correction, and password-derived indiscernibility masking.

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
# Basic encode with password masking
go-steg encode -e secret.pdf -c carrier.png -p mypassword -o output/ -u

# Multi-carrier encode
go-steg encode -e largefile.zip -c carrier1.png,carrier2.png -p mypassword -o output/ -u

# Full pipeline: higher bit depth + compression + error correction
go-steg encode -e document.pdf -c carrier.png -p mypassword -o output/ -u \
  -b 3 --huffman --rs --rsLevel high

# Decode (auto-detects encoding settings from header)
go-steg decode -c output/carrier-0-embedded.png -p mypassword -o decoded/

# Multi-carrier decode (order must match encoding order)
go-steg decode -c output/carrier1-0-embedded.png,output/carrier2-1-embedded.png \
  -p mypassword -o decoded/
```

The `-u` flag enables the indiscernibility mask. Carrier order must match between encode and decode.

## Architecture

### Core Packages

- **`go_steg/image_processing/`** — Central package containing encode/decode logic, image resizing, and header management. This is where the steganography algorithm lives.
- **`go_steg/bit_manipulation/`** — Low-level bit operations: splitting bytes into N-bit chunks, setting/getting least significant bits, and mask difference calculations.
- **`go_steg/huffman/`** — Huffman codec using password-derived encoding trees for compression.
- **`go_steg/pipeline/`** — Encode/decode pipeline orchestration (Huffman compression, Reed-Solomon, bit splitting).
- **`go_steg/reed_solomon/`** — GF(256) arithmetic, Reed-Solomon encoder/decoder for error correction.
- **`go_steg/file_processing/`** — File saving/resizing helpers for a multipart upload flow (designed for a web API use case).
- **`go_steg/logging/`** — Thin wrapper around `zap.SugaredLogger`.
- **`cli/cmd/`** — Cobra CLI commands (`encode`, `decode`, `files`). Entry point is `main.go` → `cmd.Execute()`.
- **`cli/helpers/`** — CLI validation helpers and the global `UseMask` flag.

### Key Concepts

**Variable bit depth encoding**: Each byte of embed data is split into N-bit chunks (controlled by `--bitDepth`, default 2). Each chunk is stored in the least significant bits of one color channel. At bit depth 2, 1 byte requires 4 channel slots; at bit depth 4, only 2 slots.

**Encoding pipeline**: Data flows through optional processing stages before embedding:
```
Input File → Huffman Compression (optional) → Reed-Solomon (optional) → Bit Splitting → LSB Embedding → Header Writing → PNG Output
```

**Header layout** (first 34 pixels of column x=0):
- Pixels 0-7: Photo ID (64-bit)
- Pixel 8: Photo number (for multi-carrier ordering)
- Pixels 9-12: Data count (embedded chunk count)
- Pixels 13-14: Version marker (new format detection)
- Pixels 15-25: File extension (up to 8 chars)
- Pixel 26: Encoding flags (bit depth, Huffman, RS, RS level)
- Pixels 27-28: CRC checksum (12-bit)
- Pixels 29-30: Byte count modulo (12-bit)
- Pixels 31-33: Reserved

The header always uses 2-bit operations regardless of payload bit depth.

**Indiscernibility mask**: Generated from a SHA-256 hash of the password, used to seed a deterministic PRNG that produces mask parameters. Only pixels passing the mask filter are used for embedding, making the modification pattern unpredictable without the password.

**Multi-carrier encoding**: Embed data is split into equal chunks across multiple carrier images. Carriers must be decoded in the same order they were encoded.

**Pixel traversal order**: Column-major (x outer loop, y inner loop), starting at y=34 (after header).

**Reed-Solomon error correction**: RS(255,223) standard or RS(255,191) high redundancy. Corrects minor bit-level corruption but cannot recover from JPEG recompression or header damage.

### Image Constraints

Instagram-oriented sizing constants: max carrier 1080x1350, embed images capped at half those dimensions (540x675). The `imaging` library (Lanczos filter) handles resizing. Output is always PNG to preserve LSBs.
