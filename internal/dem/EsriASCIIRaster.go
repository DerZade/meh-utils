package dem

// EsriASCIIRaster represents a ESRI ASCII Grid
type EsriASCIIRaster struct {
	Ncols, Nrows     uint
	Xcenter, Ycenter *float64
	Xcorner, Ycorner *float64
	CellSize         float64
	NoDataValue      float64
	Data             [][]float64
}

// Dims returns the dimensions of the grid.
func (raster EsriASCIIRaster) Dims() (c, r uint) {
	return raster.Ncols, raster.Nrows
}

// Z returns the value of a grid value at (c, r).
// It will panic if c or r are out of bounds for the grid.
func (raster EsriASCIIRaster) Z(c, r uint) float64 {
	return raster.Data[r][c]
}

// X returns the coordinate for the column at the index c.
// It will panic if c is out of bounds for the grid.
func (raster EsriASCIIRaster) X(c uint) float64 {
	var left float64
	if raster.Xcenter != nil {
		left = *raster.Xcenter - raster.CellSize*float64(raster.Ncols)/2
	} else {
		left = *raster.Xcorner
	}

	return left + float64(c)*raster.CellSize
}

// Y returns the coordinate for the row at the index r.
// It will panic if r is out of bounds for the grid.
func (raster EsriASCIIRaster) Y(r uint) float64 {
	// we'll subract r from Nrows, because the Raster has its origin
	// in the lower left corner and our data starts in the top left
	normalizedRow := float64(raster.Nrows - r)

	var bottom float64
	if raster.Ycenter != nil {
		bottom = *raster.Ycenter - raster.CellSize*float64(raster.Nrows)/2
	} else {
		bottom = *raster.Ycorner
	}

	return bottom + normalizedRow*raster.CellSize
}
