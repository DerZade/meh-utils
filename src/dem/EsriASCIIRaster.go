package dem

// EsriASCIIRaster represents a ESRI ASCII Grid
type EsriASCIIRaster struct {
	Ncols, Nrows     uint
	Xcenter, Ycenter float64
	Xcorner, Ycorner float64
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
	// TODO: Use Xcenter, Xcorner
	return float64(c) * raster.CellSize
}

// Y returns the coordinate for the row at the index r.
// It will panic if r is out of bounds for the grid.
func (raster EsriASCIIRaster) Y(r uint) float64 {
	// TODO: Use Ycenter, Ycorner
	return float64(raster.Ncols)*raster.CellSize - float64(r)*raster.CellSize
}
