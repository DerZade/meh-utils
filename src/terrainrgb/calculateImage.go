package terrainrgb

import (
	"image"

	dem "../dem"
)

func calculateImage(dem dem.EsriASCIIRaster, elevationOffset float64) *image.RGBA {

	w, h := dem.Dims()

	img := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{int(w), int(h)}})

	for col := uint(0); col < w; col++ {
		for row := uint(0); row < h; row++ {
			color := HeightToRgb(dem.Z(col, row) + elevationOffset)

			img.SetRGBA(int(col), int(row), color)
		}
	}

	return img
}
