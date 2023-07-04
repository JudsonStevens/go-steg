![go-steg-logo](https://github.com/JudsonStevens/go-steg/assets/35241250/7be4023c-e948-4c62-86d0-09bf5c1b1cf0 =450x450)

# Go-Steg

## Notes
Some background on images:
- Digital images are typically made up of pixels.
- Each pixel has different color channels - your general RGBA digital image has 4 channels, one each for Red,
Green, Blue, and Alpha (transparency).
- Each channel is typically represented by a byte, so each pixel is 4 bytes.
- When you see a color written out as (255, 0, 0, 255), that's the RGBA representation of the color red. The first 3
bytes are the RGB values, and the last byte is the alpha value.
- The alpha value is typically 255 for opaque images, but can be anything from 0 to 255. 0 is completely
transparent, and 255 is completely opaque.
