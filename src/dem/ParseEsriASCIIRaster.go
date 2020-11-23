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
	remainingHeaders := []string{"NCOLS", "NROWS", "XLLCENTER", "XLLCORNER", "YLLCENTER", "YLLCORNER", "CELLSIZE", "NODATA_VALUE"}
	stillIsHeader := true
	rowIndex := uint(0)
	var esriData [][]float64

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// first field as upper case
		keyword := strings.ToUpper(fields[0])

		if stillIsHeader && contains(remainingHeaders, keyword) {
			remainingHeaders = remove(remainingHeaders, keyword)

			// there can either be corner or center not both
			if keyword == "XLLCENTER" || keyword == "YLLCENTER" {
				remainingHeaders = remove(remainingHeaders, "XLLCORNER")
				remainingHeaders = remove(remainingHeaders, "YLLCORNER")
			}
			if keyword == "XLLCORNER" || keyword == "YLLCORNER" {
				remainingHeaders = remove(remainingHeaders, "XLLCENTER")
				remainingHeaders = remove(remainingHeaders, "YLLCENTER")
			}

			err := parseHeaderLine(fields, &raster)

			if err != nil {
				return raster, err
			}
		} else {
			if stillIsHeader { // this is the first data line, if stillIsHeader is true
				// we're just going to remove the NODATA_VALUE if it is still present, because it's a optional header
				remainingHeaders = remove(remainingHeaders, "NODATA_VALUE")

				if len(remainingHeaders) > 0 {
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

// contains checks whether an array contains a string
func contains(array []string, element string) bool {
	for _, curElement := range array {
		if curElement == element {
			return true
		}
	}
	return false
}

// remove removes a string from an array
func remove(arr []string, element string) []string {
	var remaining []string

	for i := 0; i < len(arr); i++ {
		if element != arr[i] {
			remaining = append(remaining, arr[i])
		}
	}

	return remaining
}
