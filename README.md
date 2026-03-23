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
`go-steg` is a package that allows you to embed images inside other images using Least Significant Bit manipulation.
This is a form of [steganography](https://www.kaspersky.com/resource-center/definitions/what-is-steganography),
which is the practice of hiding information in plain sight.
This package also allows you to embed an image with the use of a password.
This is used to generate
an [indiscernibility mask](https://www.researchgate.net/publication/341300833_Indiscernibility_Mask_Key_for_Image_Steganography)
that allows the image to be embedded in a way that makes it tough (to potentially impossible) to detect
by common steganography detection (also termed "steganalysis") tools.
Here's a list of useful tools for steganography in general, including detection:
[Steganography Tools](https://0xrick.github.io/lists/stego/).

The impetus for writing this tool was to explore and learn more about steganography, and to see how it could be used in
a practical way.
At the time of writing, there also were few Go-based tools for steganography, so this was a good opportunity to
contribute to the Go community.

Moving forward, the goal is to primarily allow for the embedding of images into "carrier" images that then can be
uploaded to image sharing sites.
This would allow for the sharing of images that contain other images, which could be used for a variety of purposes.

## Getting Started

1. Clone down the repository
2. Run `go install` to install the package (currently using Go 1.20)
3. Run `go-steg` to see the help menu

```bash
# Basic encode (backwards compatible)
go-steg encode -e embed.png -c carrier.png -p password -o output/ -u

# Encode any file type with all features
go-steg encode -e document.pdf -c carrier.png -p password -o output/ -u -b 3 --huffman --rs --rsLevel high

# Decode (automatically detects features from header)
go-steg decode -c output/carrier-0-embedded.png -p password -o output/
```

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

## Resources
- [Hiding Images in Plain Sight: Deep Steganography](https://towardsdatascience.com/hiding-images-in-plain-sight-deep-steganography-8d4f6e5e8f2f)
- [Protecting Information with Subcodstanography](https://www.researchgate.net/publication/313687159_Protecting_Information_with_Subcodstanography)
- [Indiscernability Mask Key for Image Steganography](https://www.researchgate.net/publication/341300833_Indiscernibility_Mask_Key_for_Image_Steganography)
- [Data Masking: A New Approach for Steganography](https://www.researchgate.net/publication/220540605_Data_Masking_A_New_Approach_for_Steganography)

## Reed-Solomon Error Correction

### What is Reed-Solomon?

Reed-Solomon codes are a class of error-correcting codes based on polynomial arithmetic over finite fields — specifically Galois Field GF(256) in this implementation. Originally developed by Irving Reed and Gustave Solomon in 1960, they are ubiquitous in modern data reliability: deep-space communication (the Voyager probes rely on them), QR codes, CDs, DVDs, and RAID storage all use Reed-Solomon in some form.

### How it works

Data is treated as coefficients of a polynomial over GF(256). Parity bytes are computed by evaluating this polynomial at specific points. On decode, if errors are present, the *syndromes* (evaluations at those same points) reveal that something went wrong. The Berlekamp-Massey algorithm identifies which byte positions are corrupted, and the Forney algorithm computes the exact correction values needed to restore the original data — without any retransmission or second copy of the data.

### How go-steg uses it

When RS is enabled (`--rs`), the encoding pipeline divides the payload into blocks and appends parity bytes to each block before embedding. Two protection levels are available:

| Level | Code | Parity bytes | Max correctable errors | Overhead |
|-------|------|-------------|----------------------|----------|
| Standard (default) | RS(255,223) | 32 per 223-byte block | 16 byte errors per block | ~14% |
| High (`--rsLevel high`) | RS(255,191) | 64 per 191-byte block | 32 byte errors per block | ~34% |

The level used during encoding is recorded in the stego header, so the decoder automatically applies the correct RS parameters — no flags are needed at decode time.

### What RS can and cannot protect against

**RS will correct:**
- Minor channel-value rounding from PNG re-saves — changes of ±1 in a pixel channel produce correctable bit-level errors well within the block capacity.
- Minor bit-level corruption introduced by slight image processing (brightness/contrast adjustments, color space conversions) as long as the number of affected bytes per 255-byte block stays within the level's limit.

**RS cannot correct:**
- JPEG recompression — DCT quantization destroys LSBs entirely. The extracted bit stream is not "slightly corrupted data" that RS can fix; it is essentially garbage relative to the original payload.
- Header area damage — the stego header is not RS-protected. If the header pixels are altered, decoding fails before any RS decoding even occurs.

### Choosing a redundancy level

**Standard (~14% overhead)** is the right default for most use cases. It handles the minor corruption that results from PNG re-saving or slight image processing, and the capacity cost is modest.

**High (~34% overhead)** is appropriate when the carrier image may undergo multiple re-saves, color space conversions, or light processing before decoding. If reliability matters more than raw embedding capacity, the overhead is worth it.

## Notes

- Use `go install` to install the package.

Some background on images:

- Digital images are typically made up of pixels.
- Each pixel has different color channels - your general RGBA digital image has 4 channels, one each for Red,
  Green, Blue, and Alpha (transparency).
- Each channel is typically represented by a byte, so each pixel is 4 bytes.
- When you see a color written out as (255, 0, 0, 255), that's the RGBA representation of the color red. The first 3
  bytes are the RGB values, and the last byte is the alpha value.
- The alpha value is typically 255 for opaque images, but can be anything from 0 to 255. 0 is completely
  transparent, and 255 is completely opaque.