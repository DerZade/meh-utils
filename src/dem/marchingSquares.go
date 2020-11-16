package dem

import (
	"github.com/paulmach/orb"
)

// MarchingSquares calculates the contour lines for given raster and height and adds those to the given array
func MarchingSquares(raster *EsriASCIIRaster, height float64) []orb.LineString {
	lines := []orb.LineString{}

	for col := uint(0); col < raster.Ncols-1; col++ {
		for row := uint(0); row < raster.Nrows-1; row++ {
			newLines := calcLinesForColRow(raster, col, row, height)

			for _, newLine := range newLines {
				// find all lines which can be combined with newLine
				combinableIndicies := []int{}
				for j := 0; j < len(lines); j++ {
					canCombine, _ := canCombineLines(newLine, lines[j])

					if canCombine {
						combinableIndicies = append(combinableIndicies, j)

						if len(combinableIndicies) == 2 {
							break
						}
					}
				}

				if len(combinableIndicies) == 0 {
					// no line was found which could be combined
					lines = append(lines, newLine)
				} else {
					// combine all combinable lines
					combinedLine := newLine
					for _, index := range combinableIndicies {
						_, combinedLine = combineLines(combinedLine, lines[index])
					}

					// add combined line to array
					lines[combinableIndicies[0]] = combinedLine

					if len(combinableIndicies) == 2 {
						// Remove the element at index combinableIndicies[1] from lines.
						lines[combinableIndicies[1]] = lines[len(lines)-1] // Copy last element to index combinableIndicies[1].
						lines[len(lines)-1] = nil                          // Erase last element (write zero value).
						lines = lines[:len(lines)-1]                       // Truncate slice.
					}
				}
			}
		}
	}

	return lines
}

func calcLinesForColRow(raster *EsriASCIIRaster, col uint, row uint, height float64) []orb.LineString {
	tlHeight := raster.Z(col, row)
	trHeight := raster.Z(col+1, row)
	brHeight := raster.Z(col+1, row+1)
	blHeight := raster.Z(col, row+1)

	leftX := raster.X(col)
	rightX := raster.X(col + 1)
	bottomY := raster.Y(row + 1)
	topY := raster.Y(row)

	// find MS "case"
	index := uint(0)
	if tlHeight > height {
		index |= 8
	}
	if trHeight > height {
		index |= 4
	}
	if brHeight > height {
		index |= 2
	}
	if blHeight > height {
		index |= 1
	}

	switch index {
	case 0:
		return []orb.LineString{}
	case 1, 14:
		return []orb.LineString{
			// one line from bottom to left edge
			orb.LineString{
				orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
				orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)},   // LEFT EDGE
			},
		}
	case 2, 13:
		return []orb.LineString{
			// one line from right to bottom edge
			orb.LineString{
				orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)},  // RIGHT EDGE
				orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
			},
		}
	case 3, 12:
		return []orb.LineString{
			// one line from right to left edge
			orb.LineString{
				orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}, // RIGHT EDGE
				orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)},  // LEFT EDGE
			},
		}
	case 4, 11:
		return []orb.LineString{
			// one line from top to right edge
			orb.LineString{
				orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},   // TOP EDGE
				orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}, // RIGHT EDGE
			},
		}
	case 5:
		return []orb.LineString{
			// one line from left to top edge
			orb.LineString{
				orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)}, // LEFT EDGE
				orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},  // TOP EDGE
			},
			// one line from bottom to right edge
			orb.LineString{
				orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
				orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)},  // RIGHT EDGE
			},
		}
	case 6, 9:
		return []orb.LineString{
			// one line from top to bottom edge
			orb.LineString{
				orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},    // TOP EDGE
				orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
			},
		}
	case 7, 8:
		return []orb.LineString{
			// one line from left to top edge
			orb.LineString{
				orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)}, // LEFT EDGE
				orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},  // TOP EDGE
			},
		}
	case 10:
		return []orb.LineString{
			// one line from left to bottom edge
			orb.LineString{
				orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)},   // LEFT EDGE
				orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
			},
			// one line from top to right edge
			orb.LineString{
				orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},   // TOP EDGE
				orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}, // RIGHT EDGE
			},
		}
	case 15:
		// no lines
		return []orb.LineString{}
	}

	return []orb.LineString{}
}

// linear interpolations between two known points
func interpolate(c0, h0, c1, h1, height float64) float64 {
	return (c0*(h1-height) + c1*(height-h0)) / (h1 - h0)
}

// canCombineLines checks wether two lines can be combined (second bool is whether l2, l1 have to be reversed to be combined)
func canCombineLines(l1 orb.LineString, l2 orb.LineString) (bool, bool) {
	len1 := len(l1) - 1
	len2 := len(l2) - 1

	if l1[len1].Equal(l2[0]) {
		return true, false
	}

	if l2[len2].Equal(l1[0]) {
		return true, true
	}

	l2.Reverse()

	if l1[len1].Equal(l2[0]) {
		return true, false
	}

	if l2[len2].Equal(l1[0]) {
		return true, true
	}

	return false, false
}

// combineLines checks wether line1 and line2 can be combined. If they can the combined-line will be returned
func combineLines(l1 orb.LineString, l2 orb.LineString) (bool, orb.LineString) {
	canCombine, reversed := canCombineLines(l1, l2)

	if !canCombine {
		return false, nil
	}

	if reversed {
		return true, stitchLines(l2, l1)
	}

	return true, stitchLines(l1, l2)
}

// stitchLines appends all points of line2 (except the first one) to line1
func stitchLines(line1 orb.LineString, line2 orb.LineString) orb.LineString {
	// 1 because we assume last point of line1 is equal to first point of line2
	for i := 1; i < len(line2); i++ {
		line1 = append(line1, line2[i])
	}

	return line1
}
