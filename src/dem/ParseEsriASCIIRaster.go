package dem

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ParseEsriASCIIRaster does shizzlze
func ParseEsriASCIIRaster(reader io.Reader) (EsriASCIIRaster, error) {

	raster := EsriASCIIRaster{}
	remHeaderKeywords := []string{"NCOLS", "NROWS", "XLLCENTER", "XLLCORNER", "YLLCENTER", "YLLCORNER", "CELLSIZE", "NODATA_VALUE"}
	stillIsHeader := true
	rowIndex := uint(0)
	var esriData [][]float64

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// first field as upper case
		keyword := strings.ToUpper(fields[0])

		if stillIsHeader && contains(remHeaderKeywords, keyword) {
			remHeaderKeywords = remove(remHeaderKeywords, keyword)

			// there can either be corner or center not both
			if keyword == "XLLCENTER" || keyword == "YLLCENTER" {
				remHeaderKeywords = remove(remHeaderKeywords, "XLLCORNER")
				remHeaderKeywords = remove(remHeaderKeywords, "YLLCORNER")
			}
			if keyword == "XLLCORNER" || keyword == "YLLCORNER" {
				remHeaderKeywords = remove(remHeaderKeywords, "XLLCENTER")
				remHeaderKeywords = remove(remHeaderKeywords, "YLLCENTER")
			}

			err := parseHeaderLine(fields, &raster)

			if err != nil {
				return raster, err
			}
		} else {
			if stillIsHeader { // this is the first data line, if stillIsHeader is true
				// we're just going to remove the NODATA_VALUE if it is still present, because it's a optional header
				remHeaderKeywords = remove(remHeaderKeywords, "NODATA_VALUE")

				if len(remHeaderKeywords) > 0 {
					return raster, fmt.Errorf("DEM doesn't include all mandatory headers")
				}

				stillIsHeader = false

				esriData = make([][]float64, raster.Nrows)
			}

			row, err := parseDataLine(fields, raster.Ncols)
			if err != nil {
				return raster, err
			}

			esriData[rowIndex] = row
			rowIndex++

			if rowIndex >= raster.Nrows {
				break
			}
		}
	}

	raster.Data = esriData

	return raster, nil
}

func parseHeaderLine(fields []string, grid *EsriASCIIRaster) error {
	if len(fields) != 2 {
		return fmt.Errorf("Header line must have excatly two fields")
	}

	switch strings.ToUpper(fields[0]) {
	case "NCOLS":
		i, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return err
		}
		(*grid).Ncols = uint(i)
	case "NROWS":
		i, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return err
		}
		(*grid).Nrows = uint(i)
	case "XLLCENTER":
		f, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return err
		}
		(*grid).Xcenter = &f
	case "XLLCORNER":
		f, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return err
		}
		(*grid).Xcorner = &f

	case "YLLCENTER":
		f, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return err
		}
		(*grid).Ycenter = &f

	case "YLLCORNER":
		f, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return err
		}
		(*grid).Ycorner = &f

	case "CELLSIZE":
		f, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return err
		}
		if f <= 0.0 {
			return fmt.Errorf("CELLSIZE must be greater than 0")
		}
		(*grid).CellSize = f

	case "NODATA_VALUE":
		f, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return err
		}
		(*grid).NoDataValue = f

	default:
		return fmt.Errorf("Unknown header keyword: %s", fields[0])
	}

	return nil
}

func parseDataLine(fields []string, cols uint) ([]float64, error) {
	row := make([]float64, cols)

	if uint(len(fields)) < cols {
		return row, fmt.Errorf("DEM data row is too short")
	}

	for i := uint(0); i < cols; i++ {
		f, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return row, err
		}
		row[i] = f
	}

	return row, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func remove(s []string, e string) []string {
	var r []string

	for i := 0; i < len(s); i++ {
		if e != s[i] {
			r = append(r, s[i])
		}
	}

	return r
}
