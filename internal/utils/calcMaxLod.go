package utils

import (
	"image"
	"math"
)

const tileSizeInPx = 256

// CalcMaxLodFromImage calculates maximum LOD based on the width of the combinedSatImage
func CalcMaxLodFromImage(image *image.RGBA) uint8 {
	w := float64(image.Bounds().Dy())

	tilesPerRowCol := math.Ceil(w / tileSizeInPx)

	return uint8(math.Ceil(math.Log2(tilesPerRowCol)))
}
