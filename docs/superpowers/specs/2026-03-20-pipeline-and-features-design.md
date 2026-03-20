# Pipeline Architecture & New Features Design

## Overview

Extend go-steg with four new capabilities, organized as a layered transform pipeline:

1. Arbitrary file type embedding (raw bytes with file extension in header)
2. Huffman compression (password-derived tree)
3. Reed-Solomon error correction
4. Adjustable bit depth (1-4 bits per channel)

Encode: `raw bytes → Huffman compress → Reed-Solomon encode → bit-depth writer → pixels`
Decode: `pixels → bit-depth reader → Reed-Solomon decode → Huffman decompress → raw bytes`

Each stage is a standalone `[]byte → []byte` transform. Stages can be toggled independently.

---

## 1. Header Layout

**Important: The header is always read and written at 2 bits per channel, regardless of the payload bit depth setting.** This avoids a chicken-and-egg problem (the header declares the bit depth, so it can't itself use that bit depth) and ensures backwards compatibility.

At 2 bits/channel, each pixel provides 6 bits (3 channels × 2 bits) = 0.75 bytes per pixel.

Current header uses 13 pixels (column x=0):

- Pixels 0-7: photo ID (48 bits)
- Pixel 8: photo number (6 bits)
- Pixels 9-12: data count (24 bits)

New header extends to 34 pixels:

- Pixels 0-7: photo ID (unchanged)
- Pixel 8: photo number (unchanged)
- Pixels 9-12: data count (unchanged)
- Pixels 13-14: version marker — a magic bit pattern `101010 110011` (12 bits across 2 pixels) providing a 1-in-4096 false-positive rate against natural image LSBs. The decoder reads all header pixels first, then validates: if pixels 13-14 match the magic pattern AND the bit depth field in pixel 26 contains a valid value (0-3, mapped to 1-4), treat as new-format. If either check fails, fall back to legacy mode (bit depth 2, no Huffman, no RS, output as `.png`).
- Pixels 15-25: file extension — 11 pixels × 6 bits = 66 bits = 8 bytes + 2 padding bits. Stores up to 8 ASCII characters (excluding the dot), null-terminated if shorter.
- Pixel 26: encoding flags — bit depth (2 bits: 0-3 mapped to 1-4, values outside this range are invalid and produce a decode error), Huffman enabled (1 bit), RS enabled (1 bit), RS redundancy level (1 bit, 0 = standard ~14%, 1 = high ~34%), unused (1 bit). Total: 6 bits, fits in one pixel.
- Pixels 27-28: data integrity checksum — low 12 bits of CRC-16 of the first 4 bytes of the decoded payload, stored across 2 pixels (12 bits). Provides a 1-in-4096 false-match rate. Used to detect wrong-password scenarios: after decoding, compute CRC-16 of the first 4 bytes, take the low 12 bits, and compare. Mismatch produces a clear "wrong password or corrupted data" error rather than silent garbage.
- Pixels 29-30: original data byte count — 2 pixels × 6 bits = 12 bits, storing the byte count of the pipeline output modulo 4096. Used alongside `dataCount` to recover exact byte boundaries, especially for 3-bit depth where the channel-slots-to-bytes conversion is lossy. The decoder computes candidate byte counts from `dataCount` and `bitDepth`, then selects the one matching this modulo value.
- Pixels 31-33: reserved for future use.

`totalReservedPixels` increases from 13 to 34. For a 1080x1350 image this is negligible (~0.002% of pixels). Carrier images must have a height of at least 34 pixels; reject smaller images with a clear error.

**Note:** The existing `setHeaderInformation` function has a bug where `c.R` and `c.G` are set using `SetLastTwoBits(c.B, ...)` instead of `SetLastTwoBits(c.R, ...)` and `SetLastTwoBits(c.G, ...)` respectively. This clobbers the upper 6 bits of R and G with B's upper bits. It "works" because the decoder only reads the last 2 bits, but it unnecessarily distorts the carrier image. Fix this during the header refactor.


---

## 2. Arbitrary File Type Embedding

### Encode Path

- Accept any file path as embed input, not just images.
- Extract the file extension (without the dot) from the embed file path and store it in the header (pixels 15-25).
- Read the embed file as raw bytes — no image decoding of the embed data.
- Carrier image format validation (`png`/`jpeg`) remains unchanged; it applies to the carrier, not the payload.

### Decode Path

- Read the file extension from the header.
- Write output as `decoded_file-<timestamp>.<original_ext>` instead of hardcoded `.png`.

### CLI Changes

- Rename `-e` flag description from "photo" to "file".
- No new flags needed — type is inferred from the file extension.

---

## 3. Huffman Compression

### Password-Derived Tree

Uses the same SHA-256 → seeded PRNG approach already in `generateMaskingInfo`:

1. Hash the password with SHA-256.
2. Derive a seed from the first 8 bytes of the hash.
3. Generate a deterministic frequency table: 256 entries (one per possible byte value), each assigned a pseudo-random frequency from the seeded PRNG.
4. Build the Huffman tree using the existing `BuildTree` function.

Same password always produces the same tree. No tree storage needed in the header.

### Trade-off

Since the tree is not based on actual data frequencies, compression ratios will vary. Some byte values get longer codes than necessary. On average with random frequencies the output is roughly size-neutral. The primary benefit is obscurity (variable-length substitution), not compression.

### Functions to Add

In the `go_steg/huffman/` package:

- `HuffmanEncode(data []byte, password string) []byte` — builds password-derived tree, encodes each byte to its variable-length code, packs bit strings into output bytes. Prepends a 4-byte little-endian length of the original data so the decoder knows when to stop.
- `HuffmanDecode(data []byte, password string) ([]byte, error)` — rebuilds the same tree from the password, reads bits and walks the tree to reconstruct original bytes using the prepended length. Returns an error if the data cannot be decoded (e.g., truncated input).

### Pipeline Position

First transform. Raw bytes go through Huffman before Reed-Solomon, so RS protects the compressed/transformed data.

---

## 4. Error Correction (Reed-Solomon)

### Purpose

Allow embedded data to survive minor corruption during PNG re-encoding (channel value rounding during color space conversions or re-save operations). **RS does not protect against JPEG recompression**, which destroys LSBs entirely via DCT quantization — the bit extraction itself returns garbage in that case, and RS cannot help. RS is effective against small bit-level errors in the extracted bitstream.

### Implementation

Implement a focused RS-ECC encoder/decoder in-house in a new `go_steg/reed_solomon/` package. Most Go RS libraries (`klauspost/reedsolomon`, `vivint/infectious`) implement erasure coding (shard-level, requires knowing which shards are corrupt), which does not match our failure mode (bit flips in a contiguous byte stream). We need classic polynomial-based RS error correction over GF(256) with syndrome-based error detection and correction. The implementation scope is bounded: RS(255,k) encoding/decoding over byte blocks using standard GF(256) arithmetic. Reference implementations are well-documented and the math is straightforward for fixed block sizes.

Two redundancy levels, stored in the header flag:

- **Standard — RS(255,223):** 32 parity bytes per 223 data bytes (~14% overhead). Corrects up to 16 byte errors per block.
- **High — RS(255,191):** 64 parity bytes per 191 data bytes (~34% overhead). Corrects up to 32 byte errors per block.

### Functions

- `RSEncode(data []byte, level RedundancyLevel) ([]byte, error)` — divides data into fixed-size blocks. Standard uses RS(255,223): 223 data bytes + 32 parity bytes per block. High uses RS(255,191): 191 data bytes + 64 parity bytes per block. Serializes with an 8-byte little-endian prefix: 4 bytes for block count (uint32), 4 bytes for original data length (uint32). The decoder uses the block count to iterate, and the data length to trim padding from the last block.
- `RSDecode(data []byte) ([]byte, error)` — reads the block layout prefix, applies RS error correction per block, returns the corrected original data. If any block has too many errors to correct, returns an error and aborts (no partial/best-effort output).

### Capacity Validation

Capacity validation must occur **after** the full pipeline runs (post-Huffman, post-RS), not before. The pipeline computes the final byte count, and that is checked against the carrier's available channel slots (accounting for bit depth and mask, if enabled). If the pipeline output exceeds carrier capacity, return a clear error before writing any pixels.

### Pipeline Position

Second transform. Operates on Huffman output (or raw bytes if Huffman is disabled). RS-encoded data then goes to the bit-depth writer.

### README Documentation

Include a large, well-written section in the README explaining how Reed-Solomon error correction works: the theory (polynomial evaluation, Galois fields at a high level), how it applies to steganography, what kinds of corruption it can and cannot recover from (PNG re-save rounding: yes; JPEG recompression: no), and practical guidance on choosing a redundancy level.

---

## 5. Adjustable Bit Depth

### Current State

Fixed at 2 bits per channel. Every byte splits into 4 quarters via `SplitByteIntoQuarters`.

### Change

Support 1, 2, 3, or 4 bits per channel, configurable via a new `-b` CLI flag (default 2).

### Trade-offs

| Depth | Chunks per byte | Detection risk | Capacity |
|-------|----------------|----------------|----------|
| 1 bit | 8 | Lowest | Lowest |
| 2 bit | 4 | Low (current) | Moderate |
| 3 bit | 3 (9 bits capacity, last chunk zero-padded on MSB side) | Moderate | High |
| 4 bit | 2 | Highest | Highest |

### 3-bit Depth Detail

8 bits / 3 bits = 2 full chunks + 2 remaining bits. This produces 3 chunks: two 3-bit chunks and one 2-bit chunk. The last chunk is zero-padded on the MSB side to fill a 3-bit slot (i.e., the 2 data bits occupy the LSB positions). The decoder knows the last chunk of every byte has only 2 real bits and discards the padding.

### dataCount Semantics

`dataCount` continues to count **channel slots used**, not bytes or bits. The decoder uses `dataCount` to know how many channel reads to perform. To recover the original byte count, the decoder uses the **original data byte count modulo** field stored in header pixels 29-30 (see Section 1). The formula `originalBytes = (dataCount * bitDepth) / 8` works for bit depths 1, 2, and 4 (which evenly divide 8), but produces incorrect results for 3-bit depth due to per-byte padding. The modulo field resolves this: the decoder computes a candidate byte count from `floor(dataCount * bitDepth / 8)` and adjusts +/- 1 to match the stored modulo value. When Huffman is enabled, the Huffman length prefix provides an independent cross-check.

### Implementation

Generalize the bit manipulation functions:

- `SplitByteIntoQuarters` → `SplitByte(b byte, bitsPerChunk int) []byte` — returns 8, 4, 3, or 2 chunks depending on depth.
- `ConstructByteFromQuarters` → `ConstructByte(chunks []byte, bitsPerChunk int) byte`
- `SetLastTwoBits` → `SetLastNBits(b byte, value byte, n int) byte`
- `GetLastTwoBits` → `GetLastNBits(b byte, n int) byte`
- `clearLastTwoBits` → `clearLastNBits(b byte, n int) byte`

**Mask interaction:** `ReturnMaskDifference` internally calls `clearLastTwoBits` on the color byte before comparing. This must be updated to `clearLastNBits` using the current bit depth, so the mask comparison is consistent with the number of bits being modified. The mask decides *whether* to use a pixel; the bit depth decides *how many bits* to read/write in that pixel's channels.

Keep the old 2-bit functions as wrappers around the new N-bit functions for backwards compatibility in tests and any code that explicitly uses 2-bit operations.

Bit depth is stored in the header so decode reads it automatically.

### CLI

- Encode: `go-steg encode ... -b 3`
- Decode: reads bit depth from header, no flag needed.

---

## 6. Pipeline Orchestration

### New Package

`go_steg/pipeline/` — coordinates the transform stages.

### Config

```go
type PipelineConfig struct {
    BitDepth       int             // 1-4
    HuffmanEnabled bool
    RSEnabled      bool
    RSLevel        RedundancyLevel // Standard or High
    FileExtension  string
    Password       string
}
```

Config is constructed from CLI flags on encode. On decode, it is reconstructed from the header fields plus the user-provided password.

### Integration

The `image_processing` package's `Encode`/`Decode` functions call into the pipeline for data transformation but still own pixel traversal and header read/write.

**Header and mask:** The mask is NOT applied during header pixel read/write. Header pixels (0 through `totalReservedPixels-1`) are always written/read unconditionally at 2 bits per channel, matching current behavior. The mask only applies to payload pixels starting at `totalReservedPixels`.

### Multi-Carrier Interaction

The pipeline (Huffman + RS) is applied to the **full data** before splitting into chunks for multi-carrier encoding. This means:

1. The full embed file goes through the pipeline: Huffman → RS → output bytes.
2. The output bytes are split into chunks, one per carrier (existing `MultiCarrierEncode` logic).
3. Each carrier gets the full new header (version marker, encoding flags, file extension, checksum, byte count modulo). The `dataCount` in each carrier's header reflects only that carrier's chunk size. The `photoNumber` field orders the carriers for reassembly.
4. On decode, chunks are extracted from each carrier (using per-carrier `dataCount`), concatenated in `photoNumber` order, then the inverse pipeline runs on the reassembled data: RS decode → Huffman decode → original file.

### Backwards Compatibility

When all new flags are off (no Huffman, no RS, bit depth 2), behavior matches the current implementation. Old encoded images are detected by the absence of the magic version marker in pixel 13 and decoded in legacy mode.

---

## 7. CLI Changes Summary

### New Flags (encode only)

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-b` / `--bitDepth` | int | 2 | Bits per channel (1-4) |
| `--huffman` | bool | false | Enable Huffman compression |
| `--rs` | bool | false | Enable Reed-Solomon error correction |
| `--rsLevel` | string | "standard" | RS redundancy: "standard" (~14%, 16 correctable errors/block) or "high" (~34%, 32 correctable errors/block) |

### Modified Flags

| Flag | Change |
|------|--------|
| `-e` / `--embedFileName` | Description updated from "photo" to "file"; accepts any file type |

### Decode

No new flags. Bit depth, Huffman, RS, and file extension are all read from the header. Password is still required (used for mask generation and Huffman tree derivation; if neither is active, the password is accepted but unused).

---

## 8. Package Structure

```
go_steg/
  pipeline/          # NEW — pipeline orchestration, PipelineConfig
  huffman/           # MODIFIED — add HuffmanEncode, HuffmanDecode, password-derived tree
  reed_solomon/      # NEW — in-house RS-ECC over GF(256), byte-level error correction
  bit_manipulation/  # MODIFIED — generalize to N-bit operations, keep old functions as wrappers
  image_processing/  # MODIFIED — new header layout, call pipeline for transforms
  file_processing/   # UNCHANGED
  logging/           # UNCHANGED
cli/
  cmd/               # MODIFIED — new flags, updated descriptions
  helpers/           # UNCHANGED
```
