package mvt

import (
	"context"
	"runtime"
	"sync"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
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

	contours01 := geojson.NewFeatureCollection()
	contours05 := geojson.NewFeatureCollection()
	contours10 := geojson.NewFeatureCollection()
	contours50 := geojson.NewFeatureCollection()
	contours100 := geojson.NewFeatureCollection()

	waterLines := []orb.LineString{}
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
				contours01.Append(f)
				if elev%5 == 0 {
					contours05.Append(f)
				}
				if elev%10 == 0 {
					contours10.Append(f)
				}
				if elev%50 == 0 {
					contours50.Append(f)
				}
				if elev%100 == 0 {
					contours100.Append(f)
				}
			}

			if elev == 0 {
				waterLines = lines
			}
			layersMux.Unlock()
			sem.Release(1)
		}(elevation)
	}

	waitGrp.Wait()

	(*layers)["contours/01"] = contours01
	(*layers)["contours/05"] = contours05
	(*layers)["contours/10"] = contours10
	(*layers)["contours/50"] = contours50
	(*layers)["contours/100"] = contours100

	// build water
	if len(waterLines) > 0 {
		(*layers)["water"] = buildWater(waterLines, worldSize, raster)
	}

}

func buildWater(lines []orb.LineString, worldSize float64, raster *dem.EsriASCIIRaster) *geojson.FeatureCollection {
	rings := make(map[int]orb.Ring)

	// normalize rings
	for index, line := range lines {
		r := orb.Ring(line)

		// close all rings
		if !r.Closed() {
			r = append(r, r[0])
		}

		// make sure the ring is winding order = clockwise
		// https://stackoverflow.com/a/1165943
		sum := float64(0)
		for i := 1; i < len(r); i++ {
			p1 := r[i-1]
			p2 := r[i]
			sum += (p2[0] - p1[0]) * (p2[1] + p1[1])
		}
		if sum < 0 {
			r.Reverse()
		}

		rings[index] = r
	}

	// ring-id -> array of rings which this rings contains
	ringsByParent := make(map[int][]int)

	// ring-id -> number of parents
	ringNumberOfParents := make(map[int]int)

	// fill ringsByParent and ringNumberOfParents
	for id, ring := range rings {
		childIndices := []int{}

		for childID, childRing := range rings {
			// we don't need to compare the ring to itself
			if id == childID {
				continue
			}

			if ringContainsRing(&ring, &childRing) {
				childIndices = append(childIndices, childID)
				ringNumberOfParents[childID]++
			}
		}

		ringsByParent[id] = childIndices
	}

	// find pos in DEM which is "significally" above / below 0
	col := uint(0)
	row := uint(0)
	height := raster.Z(col, row)
	for height < 0.1 && height > -0.1 {
		col++

		if col >= raster.Ncols {
			row++
			col = 0
		}

		height = raster.Z(col, row)
	}
	point := orb.Point{raster.X(col), raster.Y(row)}

	// find number of rings which contain point
	numOfContainingRings := 0
	for _, ring := range rings {
		if planar.RingContains(ring, point) {
			numOfContainingRings++
		}
	}

	// A: height > 0
	// B: numOfContainingRings%2 == 0
	//
	// if point is above 0 and the number of rings, which contain point is..
	//     ...even -> map isn't island (A && B)
	//     ...odd -> map is island (A && !B)
	// if point is below 0 and the number of rings, which contain point is..
	//     ...even -> map is island (!A && B)
	//     ...odd -> map isn't island (!A && !B)
	isIsland := (height > 0) != (numOfContainingRings%2 == 0)

	if isIsland {
		wholeMapRingIndex := -1

		wholeMapRing := orb.Ring{
			orb.Point{0, 0},
			orb.Point{0, worldSize},
			orb.Point{worldSize, worldSize},
			orb.Point{worldSize, 0},
			orb.Point{0, 0},
		}

		childRings := make([]int, len(rings))
		for id := range rings {
			childRings[id] = id

			ringNumberOfParents[id]++
		}

		ringsByParent[wholeMapRingIndex] = childRings
		rings[wholeMapRingIndex] = wholeMapRing
	}

	maxNumOfParents := 0

	// make sure rings are right winding order
	for id, ring := range rings {
		numOfParents := ringNumberOfParents[id]

		if numOfParents > maxNumOfParents {
			maxNumOfParents = numOfParents
		}

		if numOfParents%2 == 1 {
			ring.Reverse()
		}
	}

	waterFeatureCollection := geojson.NewFeatureCollection()

	// create actual features
	for level := maxNumOfParents - maxNumOfParents%2; level >= 0; level = level - 2 {
		for ringID, ring := range rings {
			if ringNumberOfParents[ringID] != level {
				continue
			}

			poly := orb.Polygon{ring}
			delete(rings, ringID)

			// add all holes that are contained in current ring
			holes := ringsByParent[ringID]
			for _, id := range holes {
				hole, found := rings[id]

				if found {
					poly = append(poly, hole)
					delete(rings, id)
				}
			}

			waterFeatureCollection.Append(geojson.NewFeature(poly))
		}
	}

	return waterFeatureCollection
}

func ringContainsRing(parent *orb.Ring, child *orb.Ring) bool {
	for _, point := range *child {
		contains := planar.RingContains(*parent, point)

		if !contains {
			return false
		}
	}

	return true
}
