package mvt

import (
	"context"
	"fmt"
	"math"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/mvt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/project"
	"github.com/paulmach/orb/simplify"

	dem "../dem"
	"../utils"
)

const tileSize = mvt.DefaultExtent

func buildVectorTiles(outputPath string, collectionsPtr *map[string]*geojson.FeatureCollection, maxLod uint8, worldSize float64, layerSettings *[]layerSetting, raster *dem.EsriASCIIRaster) {

	for lod := uint8(0); lod <= maxLod; lod++ {
		lodDir := path.Join(outputPath, fmt.Sprintf("%d", lod))
		start := time.Now()

		// create LOD directory
		if !utils.IsDirectory(path.Dir(lodDir)) {
			err := os.MkdirAll(lodDir, os.ModePerm)
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		buildLODVectorTiles(lod, maxLod, lodDir, collectionsPtr, worldSize, layerSettings, raster)

		fmt.Println("    ✔️  Finished tiles for LOD", lod, "in", time.Now().Sub(start).String())
	}
}

func buildLODVectorTiles(lod, maxLod uint8, lodDir string, collectionsPtr *map[string]*geojson.FeatureCollection, worldSize float64, layerSettings *[]layerSetting, raster *dem.EsriASCIIRaster) {
	// how many tiles one row / col has
	tilesPerRowCol := uint32(math.Pow(2, float64(lod)))

	layers := findLODLayers(collectionsPtr, layerSettings, lod, maxLod)

	// project features from arma coordinates to pixel coordinates
	pixels := uint64(tileSize) * uint64(tilesPerRowCol) // how many pixels one row / col has
	factor := float64(pixels) / worldSize               // factor to convert from arma coordinates to pixel Coords
	projectLayersInPlace(layers, func(p orb.Point) orb.Point {
		return orb.Point{
			p[0] * factor,
			(worldSize - p[1]) * factor,
		}
	})

	// set layer version to v2
	for _, l := range layers {
		l.Version = 2
	}

	simplifyLayers(&layers, lod == maxLod)

	layers = fillContourLayers(layers, worldSize, raster)

	tileWaitGroup := sync.WaitGroup{}

	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))

	for col := uint32(0); col < tilesPerRowCol; col++ {
		// create column directory
		colPath := path.Join(lodDir, fmt.Sprintf("%d", col))
		if !utils.IsDirectory(colPath) {
			err := os.MkdirAll(colPath, os.ModePerm)
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		for row := uint32(0); row < tilesPerRowCol; row++ {
			tileWaitGroup.Add(1)
			go func(c, r uint32) {
				defer tileWaitGroup.Done()

				sem.Acquire(context.Background(), 1)

				data, err := createTile(c, r, layers)
				if err != nil {
					fmt.Printf("Error while creating tile %d/%d/%d\n", lod, c, r)
					return
				}

				sem.Release(1)

				tilePath := path.Join(colPath, fmt.Sprintf("%d.pbf", r))
				err = writeTile(tilePath, data)
				if err != nil {
					fmt.Printf("Error while writing tile %d/%d/%d\n", lod, c, r)
					return
				}

			}(col, row)
		}
	}

	tileWaitGroup.Wait()
}

// findLODLayers return a mvt.Layers object which includes all layers valid for given LOD
func findLODLayers(allCollections *map[string]*geojson.FeatureCollection, settingsPtr *[]layerSetting, lod uint8, maxLod uint8) mvt.Layers {

	lodCollections := make(map[string]*geojson.FeatureCollection)

	for layerName, fc := range *allCollections {
		// find layer settings for layerName
		layerSettings := layerSetting{Layer: layerName, MinZoom: nil, MaxZoom: nil}
		for _, setting := range *settingsPtr {
			if setting.Layer == layerName {
				layerSettings = setting
				break
			}
		}

		// make sure minZoom is not bigger than the maximum calculated layer
		if layerSettings.MinZoom != nil && *layerSettings.MinZoom > maxLod {
			layerSettings.MinZoom = &maxLod
		}

		if layerSettings.MaxZoom == nil && layerSettings.MinZoom == nil {
			// both min- and maxzoom are not set
			lodCollections[layerName] = utils.DeepCloneFeatureCollection(fc)
		} else if layerSettings.MinZoom == nil {
			// only maxzoom is set
			if *layerSettings.MaxZoom >= lod {
				lodCollections[layerName] = utils.DeepCloneFeatureCollection(fc)
			}
		} else if layerSettings.MaxZoom == nil {
			// only minzoom is set
			if *layerSettings.MinZoom <= lod {
				lodCollections[layerName] = utils.DeepCloneFeatureCollection(fc)
			}
		} else {
			// both min- and maxzoom are set
			if *layerSettings.MinZoom <= lod && *layerSettings.MaxZoom >= lod {
				lodCollections[layerName] = utils.DeepCloneFeatureCollection(fc)
			}
		}
	}

	return mvt.NewLayers(lodCollections)
}

func createTile(x uint32, y uint32, layers mvt.Layers) ([]byte, error) {
	layersClone := utils.DeepCloneLayers(layers)

	xOffset := float64(x * tileSize)
	yOffset := float64(y * tileSize)
	projectLayersInPlace(layersClone, func(p orb.Point) orb.Point {
		return orb.Point{
			p[0] - xOffset,
			p[1] - yOffset,
		}
	})

	layersClone.Clip(mvt.MapboxGLDefaultExtentBound)
	// Clip doesn't remove empty features so we'll have to do that ourselves
	for _, layer := range layersClone {
		count := 0
		for i := 0; i < len(layer.Features); i++ {
			feature := layer.Features[i]
			if feature.Geometry == nil {
				continue
			}

			layer.Features[count] = feature
			count++
		}
		layer.Features = layer.Features[:count]
	}

	// marshal tile
	data, err := mvt.MarshalGzipped(layersClone)
	if err != nil {
		return []byte{}, err
	}

	return data, nil
}

func writeTile(tilePath string, data []byte) error {
	f, err := os.Create(tilePath)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

// projectLayersInPlace projects all features of a layer
func projectLayersInPlace(layers mvt.Layers, projection orb.Projection) {
	for _, layer := range layers {
		for _, feature := range (*layer).Features {
			feature.Geometry = project.Geometry(feature.Geometry, projection)
		}
	}
}

func simplifyMounts(layer *mvt.Layer, threshold float64) {
	keepCount := 0
	for i := 0; i < len(layer.Features); i++ {
		feature := layer.Features[i]

		// make sure distance to all keep features is lower than threshold
		keep := true
		for j := 0; j < keepCount; j++ {
			if planar.Distance(feature.Geometry.(orb.Point), layer.Features[j].Geometry.(orb.Point)) < threshold {
				keep = false
				break
			}
		}

		if keep {
			layer.Features[keepCount] = feature
			keepCount++
		}
	}
	layer.Features = layer.Features[:keepCount]
}

func simplifyLayers(layers *mvt.Layers, isMaxLod bool) {

	simplifySkipLayers := []string{"bunker", "chapel", "church", "cross", "fuelstation", "lighthouse", "rock", "shipwreck", "transmitter", "watertower", "fortress", "fountain", "view-tower", "quay", "hospital", "busstop", "stack", "ruin", "tourism", "powersolar", "powerwave", "powerwind", "tree", "bush"}

	skip := func(value string) bool {
		for _, v := range simplifySkipLayers {
			if v == value {
				return true
			}
		}
		return false
	}

	for _, layer := range *layers {
		name := layer.Name

		if strings.HasPrefix(name, "locations") || skip(name) {
			continue
		}

		if name == "mount" {
			thres := float64(1000)
			if isMaxLod {
				thres = 100
			}
			simplifyMounts(layer, thres)
		} else if name == "railway" || name == "powerline" || name == "water" {
			if isMaxLod {
				layer.Simplify(simplify.DouglasPeucker(1))
			} else {
				layer.Simplify(simplify.DouglasPeucker(10))
			}
			layer.RemoveEmpty(10, 20)
		} else if strings.HasPrefix(name, "contours") {
			layer.Simplify(simplify.DouglasPeucker(10))
			layer.RemoveEmpty(100, 500)
		} else {
			layers.Simplify(simplify.DouglasPeucker(1))

			if isMaxLod {
				layers.RemoveEmpty(0, 0)
			} else {
				layers.RemoveEmpty(10, 20)
			}
		}
	}
}

func fillContourLayers(layers mvt.Layers, worldSize float64, raster *dem.EsriASCIIRaster) mvt.Layers {
	contourLayerIndex := -1
	c01Index := -1
	c05Index := -1
	c10Index := -1
	c50Index := -1
	c100Index := -1
	waterIndex := -1

	for index, layer := range layers {
		if layer.Name == "contours" {
			contourLayerIndex = index
		} else if layer.Name == "contours/01" {
			c01Index = index
		} else if layer.Name == "contours/05" {
			c05Index = index
		} else if layer.Name == "contours/10" {
			c10Index = index
		} else if layer.Name == "contours/50" {
			c50Index = index
		} else if layer.Name == "contours/100" {
			c100Index = index
		} else if layer.Name == "water" {
			waterIndex = index
		}
	}

	if contourLayerIndex == -1 {
		return layers
	}

	contourLayer := layers[contourLayerIndex]

	waterLines := []orb.LineString{}

	for _, feature := range contourLayer.Features {
		elev := feature.Properties["dem_elevation"].(int)

		if c01Index > -1 {
			layers[c01Index].Features = append(layers[c01Index].Features, feature)
		}
		if c05Index > -1 && elev%5 == 0 {
			layers[c05Index].Features = append(layers[c05Index].Features, feature)
		}
		if c10Index > -1 && elev%10 == 0 {
			layers[c10Index].Features = append(layers[c10Index].Features, feature)
		}
		if c50Index > -1 && elev%50 == 0 {
			layers[c50Index].Features = append(layers[c50Index].Features, feature)
		}
		if c100Index > -1 && elev%100 == 0 {
			layers[c100Index].Features = append(layers[c100Index].Features, feature)
		}

		if elev == 0 {
			waterLines = append(waterLines, feature.Geometry.(orb.LineString))
		}
	}

	if len(waterLines) > 0 {
		layers[waterIndex] = buildWater(waterLines, worldSize, raster)
	}

	layers[contourLayerIndex] = layers[len(layers)-1]
	return layers[:len(layers)-1]
}

func buildWater(lines []orb.LineString, worldSize float64, raster *dem.EsriASCIIRaster) *mvt.Layer {
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

	return mvt.NewLayer("water", waterFeatureCollection)
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
