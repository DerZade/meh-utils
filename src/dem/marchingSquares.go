package dem

import "github.com/paulmach/orb"

// MarchingSquares calculates the contour lines for given raster and height and adds those to the given array
func MarchingSquares(raster *EsriASCIIRaster, height float64) []orb.LineString {
	lines := []orb.LineString{}

	for col := uint(0); col < raster.Ncols-1; col++ {
		for row := uint(0); row < raster.Nrows-1; row++ {
			newLines := calcLinesForColRow(raster, col, row, height)

			for _, newLine := range newLines {
				didCombine := false

				for j := 0; j < len(lines); j++ {
					didCombineCur, combinedLine := combineLines(newLine, lines[j])

					didCombine = didCombineCur || didCombine

					if didCombineCur {
						lines[j] = combinedLine
						break
					}
				}

				if !didCombine {
					lines = append(lines, newLine)
				}
			}
		}
	}

	return dissolveLines(lines)
}

func dissolveLines(lines []orb.LineString) []orb.LineString {
	for i := 0; i < len(lines); i++ {
		prevLen := len(lines) + 1
		for prevLen > len(lines) {
			prevLen = len(lines)
			for j := i + 1; j < len(lines); j++ {
				didCombine, combinedLine := combineLines(lines[i], lines[j])

				if didCombine {
					// add combined line as "current" line
					lines[i] = combinedLine

					// remove element
					lines[j] = lines[len(lines)-1] // Copy last element to index i.
					lines[len(lines)-1] = nil      // Erase last element (write zero value).
					lines = lines[:len(lines)-1]   // Truncate slice.

					// because now the previously last element is on the current index we want to visit that index again
					j--
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
		index = index | 8
	}
	if trHeight > height {
		index = index | 4
	}
	if brHeight > height {
		index = index | 2
	}
	if blHeight > height {
		index = index | 1
	}

	topEdgePoint := func() orb.Point {
		return orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY}
	}
	leftEdgePoint := func() orb.Point {
		return orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)}
	}
	bottomEdgePoint := func() orb.Point {
		return orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}
	}
	rightEdgePoint := func() orb.Point {
		return orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}
	}

	switch index {
	case 0:
		return []orb.LineString{}
	case 1, 14:
		// one line from bottom to left edge
		l1 := orb.LineString{bottomEdgePoint(), leftEdgePoint()}
		return []orb.LineString{l1}
	case 2, 13:
		// one line from right to bottom edge
		l1 := orb.LineString{rightEdgePoint(), bottomEdgePoint()}
		return []orb.LineString{l1}
	case 3, 12:
		// one line from right to left edge
		l1 := orb.LineString{rightEdgePoint(), leftEdgePoint()}
		return []orb.LineString{l1}
	case 4, 11:
		// one line from top to right edge
		l1 := orb.LineString{topEdgePoint(), rightEdgePoint()}
		return []orb.LineString{l1}
	case 5:
		// one line from left to top edge and one line from bottom to right edge
		l1 := orb.LineString{leftEdgePoint(), topEdgePoint()}
		l2 := orb.LineString{bottomEdgePoint(), rightEdgePoint()}
		return []orb.LineString{l1, l2}
	case 6, 9:
		// one line from top to bottom edge
		l1 := orb.LineString{topEdgePoint(), bottomEdgePoint()}
		return []orb.LineString{l1}
	case 7, 8:
		// one line from left to top edge
		l1 := orb.LineString{leftEdgePoint(), topEdgePoint()}
		return []orb.LineString{l1}
	case 10:
		// one line from left to bottom edge and one line from top to right edge
		l1 := orb.LineString{leftEdgePoint(), bottomEdgePoint()}
		l2 := orb.LineString{topEdgePoint(), rightEdgePoint()}
		return []orb.LineString{l1, l2}
	case 15:
		// no lines
		return []orb.LineString{}
	}

	return []orb.LineString{}
}

func interpolate(c0, h0, c1, h1, height float64) float64 {
	return (c0*(h1-height) + c1*(height-h0)) / (h1 - h0)
}

// combineLines checks wether line1 and line2 can be combined. If they can the combined-line will be returned
func combineLines(l1 orb.LineString, l2 orb.LineString) (bool, orb.LineString) {
	len1 := len(l1) - 1
	len2 := len(l2) - 1

	if l1[len1].Equal(l2[0]) {
		return true, stitchLines(l1, l2)
	}

	if l2[len2].Equal(l1[0]) {
		return true, stitchLines(l2, l1)
	}

	l2.Reverse()

	if l1[len1].Equal(l2[0]) {
		return true, stitchLines(l1, l2)
	}

	if l2[len2].Equal(l1[0]) {
		return true, stitchLines(l2, l1)
	}

	return false, nil
}

// stitchLines appends all points of line2 (except the first one) to line1
func stitchLines(line1 orb.LineString, line2 orb.LineString) orb.LineString {
	// 1 because we assume last point of line1 is equal to first point of line2
	for i := 1; i < len(line2); i++ {
		line1 = append(line1, line2[i])
	}

	return line1
}
