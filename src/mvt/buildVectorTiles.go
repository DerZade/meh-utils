package mvt

import (
	"context"
	"fmt"
	"math"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/mvt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/project"
	"github.com/paulmach/orb/simplify"

	"../utils"
)

const tileSize = mvt.DefaultExtent

func buildVectorTiles(outputPath string, collectionsPtr *map[string]*geojson.FeatureCollection, maxLod uint8, worldSize float64, layerSettings *[]layerSetting) {

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

		buildLODVectorTiles(lod, maxLod, lodDir, collectionsPtr, worldSize, layerSettings)

		fmt.Println("    ✔️  Finished tiles for LOD", lod, "in", time.Now().Sub(start).String())
	}
}

func buildLODVectorTiles(lod, maxLod uint8, lodDir string, collectionsPtr *map[string]*geojson.FeatureCollection, worldSize float64, layerSettings *[]layerSetting) {
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

	mountThres := float64(1000)
	// simplify
	if lod != maxLod {
		layers.Simplify(simplify.DouglasPeucker(10))
		layers.RemoveEmpty(10, 30)
	} else {
		layers.Simplify(simplify.DouglasPeucker(1))
		layers.RemoveEmpty(10, 20)
		mountThres = 100
	}

	// simplify mounts
	for _, layer := range layers {
		if layer.Name == "mount" {
			simplifyMounts(layer, mountThres)
		}
	}

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
