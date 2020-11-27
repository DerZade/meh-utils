package mvt

import (
	"context"
	"runtime"
	"sync"

	"github.com/paulmach/orb/geojson"
	"golang.org/x/sync/semaphore"

	dem "../dem"
)

func buildContours(raster *dem.EsriASCIIRaster, elevOffset float64, worldSize float64, layers *map[string]*geojson.FeatureCollection) {

	// find max / min elevation in DEM
	maxElevation := float64(0)
	minElevation := float64(0)
	for row := uint(0); row < raster.Nrows; row++ {
		for col := uint(0); col < raster.Ncols; col++ {
			d := raster.Data[row][col]

			if d < minElevation {
				minElevation = d
			}

			if d > maxElevation {
				maxElevation = d
			}
		}
	}

	waitGrp := sync.WaitGroup{}

	contours := geojson.NewFeatureCollection()

	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	var layersMux = sync.Mutex{}

	for elevation := int(minElevation - 1); elevation < int(maxElevation+1); elevation++ {
		waitGrp.Add(1)
		sem.Acquire(context.Background(), 1)
		go func(elev int) {
			defer waitGrp.Done()

			lines := dem.MarchingSquares(raster, float64(elev))

			layersMux.Lock()
			// add lines to correct feature collection
			for _, line := range lines {
				f := geojson.NewFeature(line)
				f.Properties["elevation"] = float64(elev) + elevOffset
				f.Properties["dem_elevation"] = elev
				contours.Append(f)
			}

			layersMux.Unlock()
			sem.Release(1)
		}(elevation)
	}

	waitGrp.Wait()

	(*layers)["contours"] = contours

	// intentionally empty, because these will be filled after contours have been simplified
	(*layers)["contours/01"] = geojson.NewFeatureCollection()
	(*layers)["contours/05"] = geojson.NewFeatureCollection()
	(*layers)["contours/10"] = geojson.NewFeatureCollection()
	(*layers)["contours/50"] = geojson.NewFeatureCollection()
	(*layers)["contours/100"] = geojson.NewFeatureCollection()

	if minElevation < 0 && maxElevation > 0 {
		(*layers)["water"] = geojson.NewFeatureCollection()
	}

}
