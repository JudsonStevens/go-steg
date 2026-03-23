<div align="center">
  <img
    src=https://github.com/JudsonStevens/go-steg/assets/35241250/7be4023c-e948-4c62-86d0-09bf5c1b1cf0
    width="300"
    height="300"
    alt="Cartoonish and stylized large dinosaur on a small rocky island with greenery, and an orange and red round
      backdrop showing towers of stone in the background. The dinosaur is a mixture of a stegosaurus and a
      tyrannosaurus rex."
  />
</div>

# Go-Steg

`go-steg` is a steganography toolkit that hides arbitrary files inside PNG and JPEG carrier images using Least Significant Bit (LSB) embedding. It supports multi-carrier splitting, variable bit depth, Huffman compression, Reed-Solomon error correction, and password-derived indiscernibility masking.

Built in Go, `go-steg` began as an exploration of [steganography](https://www.kaspersky.com/resource-center/definitions/what-is-steganography) — the practice of hiding information in plain sight. It has since grown into a full-featured pipeline for embedding, protecting, and recovering hidden data.

## Features

- **Any file type** — embed documents, archives, images, or any binary data (not just images)
- **Multi-carrier splitting** — split data across multiple carrier images for larger payloads
- **Variable bit depth (1-4 bits)** — trade stealth for capacity per channel
- **Huffman compression** — password-derived compression to reduce payload size
- **Reed-Solomon error correction** — recover data even after minor carrier corruption
- **Indiscernibility masking** — password-derived pixel selection mask that resists steganalysis detection
- **Self-describing headers** — encoded metadata (format, bit depth, compression, RS level, checksums) allows decode to auto-detect all settings
- **PNG and JPEG carriers** — accepts both formats as input (output is always PNG to preserve LSBs)

## Getting Started

1. Clone the repository
2. Run `go install` (requires Go 1.20+)
3. Run `go-steg` to see the help menu

## Usage

### Encode

```bash
# Basic encode with password masking
go-steg encode -e secret.pdf -c carrier.png -p mypassword -o output/ -u

# Multi-carrier encode (splits data across carriers)
go-steg encode -e largefile.zip -c carrier1.png,carrier2.png -p mypassword -o output/ -u

# Full pipeline: higher bit depth + compression + error correction
go-steg encode -e document.pdf -c carrier.png -p mypassword -o output/ -u \
  -b 3 --huffman --rs --rsLevel high

# Encode without masking (faster, less stealthy)
go-steg encode -e data.bin -c carrier.png -p mypassword -o output/
```

### Decode

```bash
# Decode automatically detects all encoding settings from the header
go-steg decode -c output/carrier-0-embedded.png -p mypassword -o decoded/

# Multi-carrier decode (order must match encoding order)
go-steg decode -c output/carrier1-0-embedded.png,output/carrier2-1-embedded.png \
  -p mypassword -o decoded/
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--embedFileName` | `-e` | File to embed into carrier(s) | required |
| `--carrierFileNames` | `-c` | Carrier image(s), comma-separated | required |
| `--password` | `-p` | Password for masking and Huffman key | required |
| `--outputFileDir` | `-o` | Output directory | required |
| `--useMask` | `-u` | Enable indiscernibility mask | `false` |
| `--bitDepth` | `-b` | Bits per channel (1-4) | `2` |
| `--huffman` | | Enable Huffman compression | `false` |
| `--rs` | | Enable Reed-Solomon error correction | `false` |
| `--rsLevel` | | RS redundancy: `standard` or `high` | `standard` |

## Example Images

### Image to be Embedded

![embedTest](https://github.com/JudsonStevens/go-steg/assets/35241250/e17643ba-99d9-41a6-bbeb-371ddb3a9dc1)

Photo
by <a href="https://unsplash.com/fr/@danieljschwarz?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">
Daniel J. Schwarz</a>
on <a href="https://unsplash.com/?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">Unsplash</a>

### Un-embedded Carrier Images

![carrierPhoto2](https://github.com/JudsonStevens/go-steg/assets/35241250/2ccde0f2-7fcc-49f5-a70e-0b7508d9d83b)
![carrierPhoto1](https://github.com/JudsonStevens/go-steg/assets/35241250/d00deb2d-87d8-4929-8fd5-0cb85c3d3b66)

### Embedded Carrier Images

![carrierPhoto1-0-embedded](https://github.com/JudsonStevens/go-steg/assets/35241250/0a7b7606-58b9-424a-b076-fd7fab8f4c36)
![carrierPhoto2-1-embedded](https://github.com/JudsonStevens/go-steg/assets/35241250/8a7765c4-5929-4105-bc17-93d098ac620a)

## Decoded Image

![decoded_image-2023-07-04-22-12-24](https://github.com/JudsonStevens/go-steg/assets/35241250/3735b9c9-bcfd-43f3-9d7d-38d7c708b6b1)

Photo
by <a href="https://unsplash.com/fr/@danieljschwarz?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">
Daniel J. Schwarz</a>
on <a href="https://unsplash.com/?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">Unsplash</a>

## How It Works

### Encoding Pipeline

Data flows through an optional processing pipeline before being embedded into carrier pixels:

```
Input File
  → Huffman Compression (if --huffman)
  → Reed-Solomon Encoding (if --rs)
  → Bit Splitting (split bytes into N-bit chunks)
  → LSB Embedding (write chunks into carrier pixel channels)
  → Header Writing (metadata into reserved pixels 0-33)
  → PNG Output
```

### Header Format

The first 34 pixels of column 0 store a self-describing header:

| Pixels | Content |
|--------|---------|
| 0-7 | Photo ID (64-bit) |
| 8 | Photo number (for multi-carrier ordering) |
| 9-12 | Data count (embedded chunk count) |
| 13-14 | Version marker (new format detection) |
| 15-25 | File extension (up to 8 chars) |
| 26 | Encoding flags (bit depth, Huffman, RS, RS level) |
| 27-28 | CRC checksum (12-bit) |
| 29-30 | Byte count modulo (12-bit) |
| 31-33 | Reserved |

The header uses 2-bit operations regardless of the payload bit depth, ensuring backward compatibility.

### Indiscernibility Masking

When enabled (`-u`), the password generates a deterministic pixel selection mask via SHA-256 hashing. Only pixels that pass the mask filter are used for embedding, making the modification pattern unpredictable without the password. This increases resistance to statistical steganalysis at the cost of reduced capacity.

For more on this technique, see: [Indiscernibility Mask Key for Image Steganography](https://www.researchgate.net/publication/341300833_Indiscernibility_Mask_Key_for_Image_Steganography).

### Capacity

Embedding capacity depends on carrier dimensions, bit depth, and whether masking is enabled:

```
Raw capacity (bytes) = (width × (height - 34) × 3) / ceil(8 / bitDepth)
```

For a 1080x1350 carrier at bit depth 2: ~1,065,600 bytes (~1 MB).

Pipeline processing affects effective capacity:
- **Huffman** — typically reduces payload size (compression), increasing effective capacity
- **Reed-Solomon Standard** — adds ~14% overhead
- **Reed-Solomon High** — adds ~34% overhead
- **Masking** — reduces available pixels (varies by password and carrier content)

## Reed-Solomon Error Correction

Reed-Solomon codes are error-correcting codes based on polynomial arithmetic over Galois Field GF(256). Originally developed by Irving Reed and Gustave Solomon in 1960, they are used in deep-space communication, QR codes, CDs, DVDs, and RAID storage.

When RS is enabled (`--rs`), the pipeline divides the payload into blocks and appends parity bytes before embedding:

| Level | Code | Parity bytes | Max correctable errors | Overhead |
|-------|------|-------------|----------------------|----------|
| Standard (default) | RS(255,223) | 32 per 223-byte block | 16 byte errors per block | ~14% |
| High (`--rsLevel high`) | RS(255,191) | 64 per 191-byte block | 32 byte errors per block | ~34% |

The level is recorded in the header, so the decoder applies the correct parameters automatically.

**RS will correct:**
- Minor channel-value rounding from PNG re-saves
- Minor bit-level corruption from slight image processing (brightness/contrast, color space conversions) within the block error limit

**RS cannot correct:**
- JPEG recompression (DCT quantization destroys LSBs entirely)
- Header area damage (header pixels are not RS-protected)

## Architecture

```
go-steg/
├── cli/                          # Cobra CLI (encode/decode commands)
│   ├── cmd/
│   └── helpers/
├── go_steg/
│   ├── bit_manipulation/         # Bit-level operations (split, construct, LSB get/set)
│   ├── huffman/                  # Huffman codec (password-derived encoding)
│   ├── image_processing/         # Core encode/decode, header, multi-carrier, masking
│   ├── pipeline/                 # Encode/decode pipeline orchestration
│   └── reed_solomon/             # GF(256) arithmetic, RS encoder/decoder
```

## Background

Some background on how LSB steganography works with digital images:

- Digital images are made up of pixels, each with color channels (Red, Green, Blue, Alpha).
- Each channel is a byte (0-255). A color like (255, 0, 0, 255) is fully opaque red.
- The least significant bits of each channel carry the least visual information — changing them is imperceptible to the human eye.
- At bit depth 2, changing the last 2 bits of a channel shifts its value by at most 3 out of 255 — a ~1.2% change that is visually undetectable.

## Resources

- [Hiding Images in Plain Sight: Deep Steganography](https://towardsdatascience.com/hiding-images-in-plain-sight-deep-steganography-8d4f6e5e8f2f)
- [Protecting Information with Subcodstanography](https://www.researchgate.net/publication/313687159_Protecting_Information_with_Subcodstanography)
- [Indiscernibility Mask Key for Image Steganography](https://www.researchgate.net/publication/341300833_Indiscernibility_Mask_Key_for_Image_Steganography)
- [Data Masking: A New Approach for Steganography](https://www.researchgate.net/publication/220540605_Data_Masking_A_New_Approach_for_Steganography)
- [Steganography Tools](https://0xrick.github.io/lists/stego/)
