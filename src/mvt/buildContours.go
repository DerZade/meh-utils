package mvt

import (
	"compress/gzip"
	"log"
	"math"
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

	// build water
	if len(waterLines) > 0 {
		polys := buildWater(waterLines, worldSize, &raster)

		for _, poly := range polys {
			water.Append(geojson.NewFeature(poly))
		}
	}

	(*layers)["contours/01"] = contours01
	(*layers)["contours/05"] = contours05
	(*layers)["contours/10"] = contours10
	(*layers)["contours/50"] = contours50
	(*layers)["contours/100"] = contours100
	(*layers)["water"] = water
}

func buildWater(lines []orb.LineString, worldSize float64, raster *dem.EsriASCIIRaster) []orb.Polygon {
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

		maxNumOfParents = int(math.Max(float64(maxNumOfParents), float64(numOfParents)))

		if numOfParents%2 == 1 {
			ring.Reverse()
		}
	}

	// create polygons
	polys := make([]orb.Polygon, 0)
	for level := maxNumOfParents - maxNumOfParents%2; level >= 0; level = level - 2 {
		for ringID, ring := range rings {
			if ringNumberOfParents[ringID] != level {
				continue
			}

			poly := orb.Polygon{ring}
			delete(rings, ringID)

			// add all rings that are contained in current ring
			holes := ringsByParent[ringID]
			for _, id := range holes {
				hole, found := rings[id]

				if found {
					poly = append(poly, hole)
					delete(rings, id)
				}
			}

			polys = append(polys, poly)
		}
	}

	return polys
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
