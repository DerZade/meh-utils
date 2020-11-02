package terrainrgb

import (
	"image/color"
	"math"
)

/*
	The Mapbox Terrain-RGB Tiles use the following equation to decode
	height values from rgb.

	height = -10000 + ((R * 256 * 256 + G * 256 + B) * 0.1)

	To make things easier we'll replace (R * 256 * 256 + G * 256 + B) with x to get the following equation:
	height = -10000 + (x * 0.1)
	now we can solve the equation for x and get:
	x = 10 * height + 100000

	To get the r, g and b value from x we'll use a little trick:
	We could write (R * 256 * 256 + G * 256 + B) as (R * 256^2 + G * 256^1 + B * 256^0)
	That should ring a bell for every computer scientist. Looks a awful lot like a numeral system conversion from Base256
	So we'll just convert x to as Base256 number. Position 2 will be r, position 1 will be g and position 0 will be b
*/

var MAX_X = int64(math.Pow(256, 3) - 1)

// HeightToRgb calculates rgb values from height
func HeightToRgb(height float64) color.RGBA {
	x := int64(10*height+100000) % MAX_X

	b := uint8(x % 256)
	x = int64(x / 256)

	g := uint8(x % 256)
	x = int64(x / 256)

	r := uint8(x % 256)
	x = int64(x / 256)

	return color.RGBA{
		R: r,
		G: g,
		B: b,
		A: 255,
	}
}

// RgbToHeight calculates heright from given rgb values
func RgbToHeight(color color.RGBA) float64 {
	x := int64(color.R)*int64(256)*int64(256) + int64(color.G)*int64(256) + int64(color.B)

	return float64(-10000.0) + (float64(x) * 0.1)
}
