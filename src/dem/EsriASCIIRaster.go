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
func (raster EsriASCIIRaster) Dims() (c, r int) {
	return int(raster.Ncols), int(raster.Nrows)
}

// Z returns the value of a grid value at (c, r).
// It will panic if c or r are out of bounds for the grid.
func (raster EsriASCIIRaster) Z(c, r int) float64 {
	return raster.Data[r][c]
}

// X returns the coordinate for the column at the index c.
// It will panic if c is out of bounds for the grid.
func (raster EsriASCIIRaster) X(c int) float64 {
	return float64(c)*raster.CellSize + raster.CellSize/2
}

// Y returns the coordinate for the row at the index r.
// It will panic if r is out of bounds for the grid.
func (raster EsriASCIIRaster) Y(r int) float64 {
	return float64(r)*raster.CellSize + raster.CellSize/2
}
