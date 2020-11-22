package dem

import (
	"fmt"

	"github.com/paulmach/orb"
)

type contourLineBit struct {
	Start     orb.Point
	End       orb.Point
	StartEdge cellEdge
	EndEdge   cellEdge
}

type cellEdge = uint8

const (
	topIndex    cellEdge = 0b0001
	leftIndex   cellEdge = 0b0010
	bottomIndex cellEdge = 0b0100
	rightIndex  cellEdge = 0b1000
)

func cellIndex(raster *EsriASCIIRaster, col, row uint) uint {
	return row*raster.Nrows + col
}

// MarchingSquares calculates the contour lines for given raster and height and adds those to the given array
func MarchingSquares(raster *EsriASCIIRaster, height float64) []orb.LineString {
	finishedLines := []orb.LineString{}

	visitedCells := make([]cellEdge, raster.Nrows*raster.Ncols)

	for col := uint(0); col < raster.Ncols-1; col++ {
		for row := uint(0); row < raster.Nrows-1; row++ {
			index := cellIndex(raster, col, row)
			visited := visitedCells[index]

			if visited == 0b1111 {
				continue
			}

			bits := calcBitsForColRow(raster, col, row, height)

			for _, bit := range bits {
				if bit.StartEdge & visited > 0 {
					continue
				}
				finishedLines = append(finishedLines, calculateLine(raster, col, row, height, bit, &visitedCells))
			}

			visitedCells[index] = 0b1111
		}
	}

	return finishedLines
}

func calculateLine(raster *EsriASCIIRaster, col, row uint, height float64, bit contourLineBit, visitedCells *[]cellEdge) orb.LineString {
	ring := orb.Ring{bit.Start, bit.End}

	nextCol, nextRow, nextEdge, err := nextCell(raster, col, row, bit.EndEdge)
	if err == nil {
		ring = append(ring, followLineRecursive(raster, height, nextEdge, nextCol, nextRow, col, row, visitedCells)...)
	}

	if !ring.Closed() {
		nextCol, nextRow, nextEdge, err = nextCell(raster, col, row, bit.StartEdge)
		if err == nil {
			endDirPoints := orb.Ring(followLineRecursive(raster, height, nextEdge, nextCol, nextRow, col, row, visitedCells))
			endDirPoints.Reverse()
			ring = append(endDirPoints, ring...)
		}
	}

	return orb.LineString(ring)
}

func followLineRecursive(raster *EsriASCIIRaster, height float64, edge cellEdge, col, row, startCol, startRow uint, visitedCells *[]cellEdge) []orb.Point {
	if col == startCol && row == startRow {
		return []orb.Point{}
	}

	bits := calcBitsForColRow(raster, col, row, height)

	// find bit which starts at startEdge
	var bit contourLineBit
	found := false
	for _, b := range bits {
		if b.StartEdge == edge {
			found = true
			bit = b
			break
		} else if b.EndEdge == edge {
			found = true
			bit = b
			// reverse bit
			oldStart := bit.Start
			oldStartEdge := bit.StartEdge
			bit.Start = bit.End
			bit.StartEdge = bit.EndEdge
			bit.End = oldStart
			bit.EndEdge = oldStartEdge

			break
		}
	}
	if !found {
		// TODO: Throw error
		return []orb.Point{}
	}

	if len(bits) == 1 {
		(*visitedCells)[cellIndex(raster, col, row)] = 0b1111
	} else {
		(*visitedCells)[cellIndex(raster, col, row)] |= bit.StartEdge
		(*visitedCells)[cellIndex(raster, col, row)] |= bit.EndEdge
	}

	nextCol, nextRow, nextEdge, err := nextCell(raster, col, row, bit.EndEdge)

	if err == nil {
		return append([]orb.Point{bit.End}, followLineRecursive(raster, height, nextEdge, nextCol, nextRow, startCol, startRow, visitedCells)...)
	}

	return []orb.Point{
		bit.End,
	}
}

func nextCell(raster *EsriASCIIRaster, col, row uint, edge cellEdge) (uint, uint, cellEdge, error) {
	switch edge {
	case topIndex:
		if row == 0 {
			return 0, 0, 0, fmt.Errorf("Out of bounds")
		}
		return col, row - 1, bottomIndex, nil
	case leftIndex:
		if col == 0 {
			return 0, 0, 0, fmt.Errorf("Out of bounds")
		}
		return col - 1, row, rightIndex, nil
	case bottomIndex:
		if row == raster.Nrows-2 {
			return 0, 0, 0, fmt.Errorf("Out of bounds")
		}
		return col, row + 1, topIndex, nil
	case rightIndex:
		if col == raster.Ncols-2 {
			return 0, 0, 0, fmt.Errorf("Out of bounds")
		}
		return col + 1, row, leftIndex, nil
	}

	return 0, 0, 0, fmt.Errorf("No valid edge")
}

// calcBitsForColRow calculates contour line bits for cell which 
func calcBitsForColRow(raster *EsriASCIIRaster, col, row uint, height float64) []contourLineBit {
	tlHeight := raster.Z(col, row)
	trHeight := raster.Z(col+1, row)
	brHeight := raster.Z(col+1, row+1)
	blHeight := raster.Z(col, row+1)

	leftX := raster.X(col)
	rightX := raster.X(col + 1)
	bottomY := raster.Y(row + 1)
	topY := raster.Y(row)

	// find MS "case"
	index := uint8(0)
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
		return []contourLineBit{}
	case 1, 14:
		return []contourLineBit{
			// one line from bottom to left edge
			contourLineBit{
				StartEdge: bottomIndex,
				EndEdge:   leftIndex,
				Start:     orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
				End:       orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)},   // LEFT EDGE
			},
		}
	case 2, 13:
		return []contourLineBit{
			// one line from right to bottom edge
			contourLineBit{
				StartEdge: rightIndex,
				EndEdge:   bottomIndex,
				Start:     orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)},  // RIGHT EDGE
				End:       orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
			},
		}
	case 3, 12:
		return []contourLineBit{
			// one line from right to left edge
			contourLineBit{
				StartEdge: rightIndex,
				EndEdge:   leftIndex,
				Start:     orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}, // RIGHT EDGE
				End:       orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)},  // LEFT EDGE
			},
		}
	case 4, 11:
		return []contourLineBit{
			// one line from top to right edge
			contourLineBit{
				StartEdge: topIndex,
				EndEdge:   rightIndex,
				Start:     orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},   // TOP EDGE
				End:       orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}, // RIGHT EDGE
			},
		}
	case 5:
		return []contourLineBit{
			// one line from left to top edge
			contourLineBit{
				StartEdge: leftIndex,
				EndEdge:   topIndex,
				Start:     orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)}, // LEFT EDGE
				End:       orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},  // TOP EDGE
			},
			// one line from bottom to right edge
			contourLineBit{
				StartEdge: bottomIndex,
				EndEdge:   rightIndex,
				Start:     orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
				End:       orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)},  // RIGHT EDGE
			},
		}
	case 6, 9:
		return []contourLineBit{
			// one line from top to bottom edge
			contourLineBit{
				StartEdge: topIndex,
				EndEdge:   bottomIndex,
				Start:     orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},    // TOP EDGE
				End:       orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
			},
		}
	case 7, 8:
		return []contourLineBit{
			// one line from left to top edge
			contourLineBit{
				StartEdge: leftIndex,
				EndEdge:   topIndex,
				Start:     orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)}, // LEFT EDGE
				End:       orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},  // TOP EDGE
			},
		}
	case 10:
		return []contourLineBit{
			// one line from left to bottom edge
			contourLineBit{
				StartEdge: leftIndex,
				EndEdge:   bottomIndex,
				Start:     orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)},   // LEFT EDGE
				End:       orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
			},
			// one line from top to right edge
			contourLineBit{
				StartEdge: topIndex,
				EndEdge:   rightIndex,
				Start:     orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},   // TOP EDGE
				End:       orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}, // RIGHT EDGE
			},
		}
	case 15:
		// no lines
		return []contourLineBit{}
	}

	return []contourLineBit{}
}

// linear interpolations between two known points
func interpolate(c0, h0, c1, h1, height float64) float64 {
	return (c0*(h1-height) + c1*(height-h0)) / (h1 - h0)
}
