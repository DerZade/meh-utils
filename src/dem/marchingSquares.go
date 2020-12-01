package dem

import (
	"fmt"

	"github.com/paulmach/orb"
)

type contourLineBit_ struct {
	Start     orb.Point
	End       orb.Point
	StartEdge cellEdge_
	EndEdge   cellEdge_
}

type cellEdge_ = byte

type cell_ struct {
	Col uint
	Row uint
}

const (
	topEdgeIndex    cellEdge_ = 0b0001
	leftEdgeIndex   cellEdge_ = 0b0010
	bottomEdgeIndex cellEdge_ = 0b0100
	rightEdgeIndex  cellEdge_ = 0b1000
)

// cellIndex returns the index of a cell
func cellIndex(raster *EsriASCIIRaster, cell cell_) uint {
	return cell.Row*(raster.Nrows-1) + cell.Col
}

// MarchingSquares calculates the contour lines for given raster and height
func MarchingSquares(raster *EsriASCIIRaster, height float64) []orb.LineString {
	finishedLines := []orb.LineString{}

	// each cell is represented by a cellEdge_ (byte). The first four bits of each cellEdge_
	// indicate which edges are already accounted for:
	// 0bXXXX
	//   │││└─ top
	//   ││└─ left
	//   │└─ bottom
	//   └─ right
	//
	// 0b1111 indicates that the cell is done (there are no pending countour line bits)
	//
	// i.e. we have a cell with two contour line bits. One from top to left and one from bottom
	// to right. If we've already calculated a contour line, which includes the bit from top to
	// left, but no line included the bit from bottom to right the value would be 0b0011
	visitedCells := make([]cellEdge_, (raster.Nrows-1)*(raster.Ncols-1))

	for col := uint(0); col < raster.Ncols-1; col++ {
		for row := uint(0); row < raster.Nrows-1; row++ {
			cell := cell_{col, row}
			index := cellIndex(raster, cell)
			visited := visitedCells[index]

			// check if all lines in this cell are done
			if visited == 0b1111 {
				continue
			}

			bits := calcBitsForColRow(raster, cell, height)

			for _, bit := range bits {
				// check if bit is already included in a finished line
				if bit.StartEdge & visited > 0 {
					continue
				}

				// make new ring containing the bit
				line := orb.LineString{bit.Start, bit.End}

				// calculate line in one direction
				nextCell, nextEdge, err := neighbourCell(raster, cell, bit.EndEdge)
				if err == nil {
					line = append(line, followLine(raster, height, nextEdge, nextCell, cell, &visitedCells)...)
				}
			
				// follow the line in the other direction if the line is not already closed (we made an ring)
				if line[0] != line[len(line)-1] {
					nextCell, nextEdge, err = neighbourCell(raster, cell, bit.StartEdge)
					if err == nil {
						endDirPoints := orb.LineString(followLine(raster, height, nextEdge, nextCell, cell, &visitedCells))
						endDirPoints.Reverse()
						line = append(endDirPoints, line...)
					}
				}

				finishedLines = append(finishedLines, line)
			}

			// mark cell as done
			visitedCells[index] = 0b1111
		}
	}

	return finishedLines
}

// followLine follows line recursively to either the edge of the raster of the start cell
func followLine(raster *EsriASCIIRaster, height float64, edge cellEdge_, cell, startCell cell_, visitedCells *[]cellEdge_) []orb.Point {
	bits := calcBitsForColRow(raster, cell, height)

	// find bit which starts at startEdge
	var bit contourLineBit_
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
		return []orb.Point{}
	}

	// mark cell as visited
	if len(bits) == 1 {
		(*visitedCells)[cellIndex(raster, cell)] = 0b1111
	} else {
		(*visitedCells)[cellIndex(raster, cell)] |= bit.StartEdge
		(*visitedCells)[cellIndex(raster, cell)] |= bit.EndEdge
	}

	// calculate next cell
	nextCell, nextEdge, err := neighbourCell(raster, cell, bit.EndEdge)
	if err != nil || (nextCell.Col == startCell.Col && nextCell.Row == startCell.Row) {
		return []orb.Point{ bit.End }
	}
	
	// recurse to next cell
	return append([]orb.Point{bit.End}, followLine(raster, height, nextEdge, nextCell, startCell, visitedCells)...)
}

// neighbourCell calculates the neighbouring cell on given edge
func neighbourCell(raster *EsriASCIIRaster, cell cell_, edge cellEdge_) (cell_, cellEdge_, error) {
	switch edge {
	case topEdgeIndex:
		if cell.Row == 0 {
			return cell_{}, 0, fmt.Errorf("Out of bounds")
		}
		return cell_{cell.Col, cell.Row - 1}, bottomEdgeIndex, nil
	case leftEdgeIndex:
		if cell.Col == 0 {
			return cell_{}, 0, fmt.Errorf("Out of bounds")
		}
		return cell_{cell.Col - 1, cell.Row}, rightEdgeIndex, nil
	case bottomEdgeIndex:
		if cell.Row == raster.Nrows-2 {
			return cell_{}, 0, fmt.Errorf("Out of bounds")
		}
		return cell_{cell.Col, cell.Row + 1}, topEdgeIndex, nil
	case rightEdgeIndex:
		if cell.Col == raster.Ncols-2 {
			return cell_{}, 0, fmt.Errorf("Out of bounds")
		}
		return cell_{cell.Col + 1, cell.Row}, leftEdgeIndex, nil
	}

	return cell_{}, 0, fmt.Errorf("No valid edge")
}

// calcBitsForColRow calculates contour line bits for given cell and height
func calcBitsForColRow(raster *EsriASCIIRaster, cell cell_, height float64) []contourLineBit_ {
	tlHeight := raster.Z(cell.Col, cell.Row)
	trHeight := raster.Z(cell.Col+1, cell.Row)
	brHeight := raster.Z(cell.Col+1, cell.Row+1)
	blHeight := raster.Z(cell.Col, cell.Row+1)

	leftX := raster.X(cell.Col)
	rightX := raster.X(cell.Col + 1)
	bottomY := raster.Y(cell.Row + 1)
	topY := raster.Y(cell.Row)

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
		return []contourLineBit_{}
	case 1, 14:
		return []contourLineBit_{
			// one line from bottom to left edge
			contourLineBit_{
				StartEdge: bottomEdgeIndex,
				EndEdge:   leftEdgeIndex,
				Start:     orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
				End:       orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)},   // LEFT EDGE
			},
		}
	case 2, 13:
		return []contourLineBit_{
			// one line from right to bottom edge
			contourLineBit_{
				StartEdge: rightEdgeIndex,
				EndEdge:   bottomEdgeIndex,
				Start:     orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)},  // RIGHT EDGE
				End:       orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
			},
		}
	case 3, 12:
		return []contourLineBit_{
			// one line from right to left edge
			contourLineBit_{
				StartEdge: rightEdgeIndex,
				EndEdge:   leftEdgeIndex,
				Start:     orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}, // RIGHT EDGE
				End:       orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)},  // LEFT EDGE
			},
		}
	case 4, 11:
		return []contourLineBit_{
			// one line from top to right edge
			contourLineBit_{
				StartEdge: topEdgeIndex,
				EndEdge:   rightEdgeIndex,
				Start:     orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},   // TOP EDGE
				End:       orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}, // RIGHT EDGE
			},
		}
	case 5:
		return []contourLineBit_{
			// one line from left to top edge
			contourLineBit_{
				StartEdge: leftEdgeIndex,
				EndEdge:   topEdgeIndex,
				Start:     orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)}, // LEFT EDGE
				End:       orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},  // TOP EDGE
			},
			// one line from bottom to right edge
			contourLineBit_{
				StartEdge: bottomEdgeIndex,
				EndEdge:   rightEdgeIndex,
				Start:     orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
				End:       orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)},  // RIGHT EDGE
			},
		}
	case 6, 9:
		return []contourLineBit_{
			// one line from top to bottom edge
			contourLineBit_{
				StartEdge: topEdgeIndex,
				EndEdge:   bottomEdgeIndex,
				Start:     orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},    // TOP EDGE
				End:       orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
			},
		}
	case 7, 8:
		return []contourLineBit_{
			// one line from left to top edge
			contourLineBit_{
				StartEdge: leftEdgeIndex,
				EndEdge:   topEdgeIndex,
				Start:     orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)}, // LEFT EDGE
				End:       orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},  // TOP EDGE
			},
		}
	case 10:
		return []contourLineBit_{
			// one line from left to bottom edge
			contourLineBit_{
				StartEdge: leftEdgeIndex,
				EndEdge:   bottomEdgeIndex,
				Start:     orb.Point{leftX, interpolate(bottomY, blHeight, topY, tlHeight, height)},   // LEFT EDGE
				End:       orb.Point{interpolate(leftX, blHeight, rightX, brHeight, height), bottomY}, // BOTTOM EDGE
			},
			// one line from top to right edge
			contourLineBit_{
				StartEdge: topEdgeIndex,
				EndEdge:   rightEdgeIndex,
				Start:     orb.Point{interpolate(leftX, tlHeight, rightX, trHeight, height), topY},   // TOP EDGE
				End:       orb.Point{rightX, interpolate(bottomY, brHeight, topY, trHeight, height)}, // RIGHT EDGE
			},
		}
	case 15:
		// no lines
		return []contourLineBit_{}
	}

	return []contourLineBit_{}
}

// linear interpolations between two known points
func interpolate(c0, h0, c1, h1, height float64) float64 {
	return (c0*(h1-height) + c1*(height-h0)) / (h1 - h0)
}
