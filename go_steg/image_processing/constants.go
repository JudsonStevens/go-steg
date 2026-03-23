package image_processing

// Legacy header layout (pixels 0-12)
const photoIDHeaderReservedPixels = 8
const photoNumberHeaderReservedPixels = 1
const dataSizeHeaderReservedPixels = 4
const legacyTotalReservedPixels = photoIDHeaderReservedPixels + photoNumberHeaderReservedPixels + dataSizeHeaderReservedPixels // 13

// New header layout (pixels 13-33)
const versionMarkerPixels = 2
const fileExtensionPixels = 11
const encodingFlagsPixels = 1
const checksumPixels = 2
const byteCountModuloPixels = 2
const reservedPixels = 3

const totalReservedPixels = legacyTotalReservedPixels + versionMarkerPixels + fileExtensionPixels +
	encodingFlagsPixels + checksumPixels + byteCountModuloPixels + reservedPixels // 34

// Version marker magic: 101010 110011 across 2 pixels (12 bits)
// Pixel 13 R/G/B last-2-bits: 10, 10, 10
// Pixel 14 R/G/B last-2-bits: 11, 00, 11
var versionMarkerBytes = [6]byte{2, 2, 2, 3, 0, 3}

const instagramMaxImageWidth = 1080
const instagramMaxImageHeight = 1350
const instagramHalfMaxWidth = instagramMaxImageWidth / 2
const instagramHalfMaxHeight = instagramMaxImageHeight / 2
const minCarrierHeight = totalReservedPixels
