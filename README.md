<div align="center">
  <img src=https://github.com/JudsonStevens/go-steg/assets/35241250/7be4023c-e948-4c62-86d0-09bf5c1b1cf0 width="300" height="300" />
</div>

# Go-Steg

## Example Images

### Image to be Embedded
![embedTest](https://github.com/JudsonStevens/go-steg/assets/35241250/e17643ba-99d9-41a6-bbeb-371ddb3a9dc1)
Photo by <a href="https://unsplash.com/fr/@danieljschwarz?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">Daniel J. Schwarz</a> on <a href="https://unsplash.com/?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">Unsplash</a>
  

### Un-embedded Carrier Images
![carrierPhoto2](https://github.com/JudsonStevens/go-steg/assets/35241250/2ccde0f2-7fcc-49f5-a70e-0b7508d9d83b)
![carrierPhoto1](https://github.com/JudsonStevens/go-steg/assets/35241250/d00deb2d-87d8-4929-8fd5-0cb85c3d3b66)

### Embedded Carrier Images
![carrierPhoto1-0-embedded](https://github.com/JudsonStevens/go-steg/assets/35241250/0a7b7606-58b9-424a-b076-fd7fab8f4c36)
![carrierPhoto2-1-embedded](https://github.com/JudsonStevens/go-steg/assets/35241250/8a7765c4-5929-4105-bc17-93d098ac620a)

## Decoded Image
![decoded_image-2023-07-04-22-12-24](https://github.com/JudsonStevens/go-steg/assets/35241250/3735b9c9-bcfd-43f3-9d7d-38d7c708b6b1)
Photo by <a href="https://unsplash.com/fr/@danieljschwarz?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">Daniel J. Schwarz</a> on <a href="https://unsplash.com/?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">Unsplash</a>
  
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
