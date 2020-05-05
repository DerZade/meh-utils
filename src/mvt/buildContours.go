package mvt

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/paulmach/orb/geojson"

	"github.com/paulmach/orb"

	dem "../dem"
)

func buildContours(demPath string, elevOffset float64, layers *map[string]*geojson.FeatureCollection) {
	file, err := os.Open(demPath)

	if err != nil {
		log.Fatal(err)
	}

	gz, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	defer gz.Close()

	raster, err := dem.ParseEsriASCIIRaster(gz)
	if err != nil {
		log.Fatal(err)
	}

	// find max / min height in DEM
	max := float64(0)
	min := float64(0)
	for row := uint64(0); row < raster.Nrows; row++ {
		for col := uint64(0); col < raster.Ncols; col++ {
			d := raster.Data[row][col]

			if d < min {
				min = d
			}

			if d > max {
				max = d
			}
		}
	}
	heightsToQuery := []float64{}
	for i := float64(int(min) - 1); i < max; i++ {
		heightsToQuery = append(heightsToQuery, i)
	}

	heightMap := make(map[float64][]orb.LineString)
	for i := 0; i < len(heightsToQuery); i++ {
		heightMap[heightsToQuery[i]] = []orb.LineString{}
	}

	dem.Conrec(raster, heightsToQuery, func(i, j int, l orb.LineString, height float64) {
		heightMap[height] = append(heightMap[height], l)
	})

	wg := sync.WaitGroup{}

	// dissolve lines as far as possible
	for height, lines := range heightMap {
		wg.Add(1)
		go func(height float64, lines []orb.LineString) {
			defer wg.Done()
			defer fmt.Println("yo")
			heightMap[height] = dissolveLines(lines)
		}(height, lines)
	}

	wg.Wait()

	contours01 := geojson.NewFeatureCollection()
	contours05 := geojson.NewFeatureCollection()
	contours10 := geojson.NewFeatureCollection()
	contours50 := geojson.NewFeatureCollection()
	contours100 := geojson.NewFeatureCollection()

	for height, lines := range heightMap {
		for i := 0; i < len(lines); i++ {
			f := geojson.NewFeature(lines[i])

			h := int(height)

			f.Properties["elevation"] = h

			contours01.Append(f)

			if h%5 == 0 {
				contours05.Append(f)
			}

			if h%10 == 0 {
				contours10.Append(f)
			}

			if h%50 == 0 {
				contours50.Append(f)
			}

			if h%100 == 0 {
				contours100.Append(f)
			}
		}
	}

	(*layers)["contours/01"] = contours01
	(*layers)["contours/05"] = contours05
	(*layers)["contours/10"] = contours10
	(*layers)["contours/50"] = contours50
	(*layers)["contours/100"] = contours100
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
