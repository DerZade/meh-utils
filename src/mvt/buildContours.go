package mvt

import (
	"compress/gzip"
	"log"
	"os"
	"sync"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"

	dem "../dem"
)

func buildContours(demPath string, elevOffset float64, worldSize float64, layers *map[string]*geojson.FeatureCollection) {
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
	water := geojson.NewFeatureCollection()
	land := geojson.NewFeatureCollection()

	waterLines := []orb.LineString{}

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
			if int(height) == 0 {
				waterLines = lines
			}
		}(height)
	}

	waitGrp.Wait()

	p1 := orb.Point{0, 0}
	p2 := orb.Point{worldSize, 0}
	p3 := orb.Point{worldSize, worldSize}
	p4 := orb.Point{0, worldSize}
	wholeMapFeature := geojson.NewFeature(orb.Polygon{orb.Ring{p1, p2, p3, p4, p1}})

	// build water
	if len(waterLines) > 0 {
		poly := orb.Polygon{}

		for _, line := range waterLines {
			r := orb.Ring(line)

			if !r.Closed() {
				r = append(r, r[0])
			}

			poly = append(poly, r)
		}

		polygonIsLand := false

		// find pos in DEM which is above / below 0
		col := uint(0)
		row := uint(0)
		height := raster.Z(col, row)
		for height < 0.1 && height > -0.1 {
			col++

			if col >= raster.Ncols {
				row++
				col = 0
			}
		}

		// polygon represents land if point is in poly and height is above 0 or point isn't in poly and height is below 0
		point := orb.Point{raster.X(col), raster.Y(row)}
		if planar.PolygonContains(poly, point) {
			polygonIsLand = height > 0
		} else {
			polygonIsLand = height < 0
		}

		polyFeature := geojson.NewFeature(poly)
		if polygonIsLand {
			water.Append(wholeMapFeature)
			land.Append(polyFeature)
		} else {
			land.Append(wholeMapFeature)
			water.Append(polyFeature)
		}

	} else {
		land.Append(wholeMapFeature)
	}

	(*layers)["contours/01"] = contours01
	(*layers)["contours/05"] = contours05
	(*layers)["contours/10"] = contours10
	(*layers)["contours/50"] = contours50
	(*layers)["contours/100"] = contours100
	(*layers)["water"] = water
	(*layers)["land"] = land
}
