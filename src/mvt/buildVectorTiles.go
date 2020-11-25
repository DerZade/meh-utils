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

		buildLODVectorTiles(lod, lodDir, collectionsPtr, worldSize, layerSettings)

		fmt.Println("    ✔️  Finished tiles for LOD", lod, "in", time.Now().Sub(start).String())
	}
}

func buildLODVectorTiles(lod uint8, lodDir string, collectionsPtr *map[string]*geojson.FeatureCollection, worldSize float64, layerSettings *[]layerSetting) {
	// how many tiles one row / col has
	tilesPerRowCol := uint32(math.Pow(2, float64(lod)))

	layers := findLODLayers(collectionsPtr, layerSettings, uint16(lod))

	// project features to pixels
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

	// simplify
	layers.Simplify(simplify.DouglasPeucker(1.0))
	layers.RemoveEmpty(10.0, 20.0)

	// TODO: Remove points too close together

	colWaitGrp := sync.WaitGroup{}

	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))

	for col := uint32(0); col < tilesPerRowCol; col++ {
		colWaitGrp.Add(1)
		go func(col uint32) {
			defer colWaitGrp.Done()
			// create column directory
			colPath := path.Join(lodDir, fmt.Sprintf("%d", col))
			if !utils.IsDirectory(colPath) {
				err := os.MkdirAll(colPath, os.ModePerm)
				if err != nil {
					fmt.Println(err)
					return
				}
			}

			rowWaitGrp := sync.WaitGroup{}

			for row := uint32(0); row < tilesPerRowCol; row++ {
				rowWaitGrp.Add(1)
				go func(row uint32) {
					defer rowWaitGrp.Done()

					sem.Acquire(context.Background(), 1)

					data, err := createTile(col, row, layers)
					if err != nil {
						fmt.Printf("Error while creating tile %d/%d/%d\n", lod, col, row)
						return
					}

					tilePath := path.Join(colPath, fmt.Sprintf("%d.pbf", row))
					writeTile(tilePath, data)

					sem.Release(1)

				}(row)
			}

			rowWaitGrp.Wait()
		}(col)
	}

	colWaitGrp.Wait()
}

// findLODLayers return a mvt.Layers object which includes all layers valid for given LOD
func findLODLayers(allCollections *map[string]*geojson.FeatureCollection, settingsPtr *[]layerSetting, lod uint16) mvt.Layers {

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
	layersClone.RemoveEmpty(0, 0)
	// Clip doesn't remove empty features so we'll have to do that ourselves
	// for _, layer := range layersClone {
	// 	count := 0
	// 	for i := 0; i < len(layer.Features); i++ {
	// 		feature := layer.Features[i]
	// 		if feature.Geometry == nil {
	// 			continue
	// 		}

	// 		layer.Features[count] = feature
	// 	}
	// 	layer.Features = layer.Features[:count]
	// }

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
