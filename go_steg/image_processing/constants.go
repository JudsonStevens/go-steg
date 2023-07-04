package image_processing

// These constants are the reserved pixels for header information
// The dataSizeHeader will reserve 4 pixels or 24 channels to store the size of the data
// The ID header will store a 64-bit integer as a unique photo ID for the embed photo
// The number header will store the order that the carrier is in order to get the order
// correct for decoding
const dataSizeHeaderReservedPixels = 4
const photoIDHeaderReservedPixels = 8
const photoNumberHeaderReservedPixels = 1
const totalReservedPixels = dataSizeHeaderReservedPixels + photoIDHeaderReservedPixels + photoNumberHeaderReservedPixels

// Set some constants for image size
const instagramMaxImageWidth = 1080
const instagramMaxImageHeight = 1350
const instagramHalfMaxWidth = instagramMaxImageWidth / 2
const instagramHalfMaxHeight = instagramMaxImageHeight / 2
