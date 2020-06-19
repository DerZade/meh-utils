package mvt

import (
	"compress/gzip"
	"log"
	"os"
	"sync"

	"github.com/paulmach/orb/geojson"

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
	for row := uint(0); row < raster.Nrows; row++ {
		for col := uint(0); col < raster.Ncols; col++ {
			d := raster.Data[row][col]

			if d < min {
				min = d
			}

			if d > max {
				max = d
			}
		}
	}

	waitGrp := sync.WaitGroup{}

	contours01 := geojson.NewFeatureCollection()
	contours05 := geojson.NewFeatureCollection()
	contours10 := geojson.NewFeatureCollection()
	contours50 := geojson.NewFeatureCollection()
	contours100 := geojson.NewFeatureCollection()

	// height will
	for height := float64(int(min) - 1); height < max; height++ {
		waitGrp.Add(1)
		go func(height float64) {
			defer waitGrp.Done()

			lines := dem.MarchingSquares(&raster, height)

			// add lines to correct feature collection
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
		}(height)
	}

	waitGrp.Wait()

	(*layers)["contours/01"] = contours01
	(*layers)["contours/05"] = contours05
	(*layers)["contours/10"] = contours10
	(*layers)["contours/50"] = contours50
	(*layers)["contours/100"] = contours100
}
