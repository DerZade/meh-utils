package sat

import (
	"image"
	"math"
)

const tileSizeInPx = 256

// CalcMaxLod calculates maximum LOD based on the width of the combinedSatImage
func calcMaxLod(combinedSatImage *image.RGBA) uint8 {
	w := float64(combinedSatImage.Bounds().Dy())

	tilesPerRowCol := math.Ceil(w / tileSizeInPx)

	return uint8(math.Ceil(math.Log2(tilesPerRowCol)))
}
