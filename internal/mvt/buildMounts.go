package mvt

import (
	"fmt"
	"math"
	"math/rand"

	dem "github.com/gruppe-adler/meh-utils/internal/dem"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

func buildMounts(raster *dem.EsriASCIIRaster, elevOffset float64, layers *map[string]*geojson.FeatureCollection) {

	mounts := geojson.NewFeatureCollection()

	// for all cells (except edges)
	for row := uint(1); row < raster.Nrows-1; row++ {
		for col := uint(1); col < raster.Ncols-1; col++ {
			elevation := raster.Data[row][col]

			// we'll only create mounts for peaks, which are above the water level
			if elevation <= 0 {
				continue
			}

			hasHigherNeighbours := false
			hasLowerNeighbours := false

			// compare cell with all direct neighbours
			for compareRow := row - 1; compareRow <= row+1; compareRow++ {
				// no peak, if we have lower and higher neighbours -> break
				if hasHigherNeighbours && hasLowerNeighbours {
					break
				}
				for compareCol := col - 1; compareCol <= col+1; compareCol++ {
					// no peak, if we have lower and higher neighbours -> break
					if hasHigherNeighbours && hasLowerNeighbours {
						break
					}

					// we don't want to compare to the reference cell
					if row == compareRow && col == compareCol {
						continue
					}

					compareElev := raster.Data[compareRow][compareCol]

					// we'll count same elvation as both a high and low neighbour because we
					// don't want to generate a "mount" for cells that are in the middle of a plane
					if compareElev == elevation {
						hasHigherNeighbours = true
						hasLowerNeighbours = true
						break
					}

					hasHigherNeighbours = hasHigherNeighbours || compareElev > elevation
					hasLowerNeighbours = hasLowerNeighbours || compareElev < elevation
				}
			}

			// add mount if all neighbours are lower (= this is a peak)
			if hasLowerNeighbours && !hasHigherNeighbours {
				feature := geojson.NewFeature(orb.Point{raster.X(col), raster.Y(row)})
				feature.Properties["elevation"] = elevation + elevOffset
				feature.Properties["text"] = fmt.Sprintf("%.0f", math.Round(elevation+elevOffset))

				mounts.Append(feature)
			}
		}
	}

	mounts.Features = quicksortMounts(mounts.Features)

	(*layers)["mount"] = mounts

}

func quicksortMounts(a []*geojson.Feature) []*geojson.Feature {
	if len(a) < 2 {
		return a
	}

	left, right := 0, len(a)-1

	pivot := rand.Int() % len(a)

	a[pivot], a[right] = a[right], a[pivot]

	for i := 0; i < len(a); i++ {
		if a[i].Properties["elevation"].(float64) < a[right].Properties["elevation"].(float64) {
			a[left], a[i] = a[i], a[left]
			left++
		}
	}

	a[left], a[right] = a[right], a[left]

	quicksortMounts(a[:left])
	quicksortMounts(a[left+1:])

	return a
}
