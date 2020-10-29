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

	"github.com/paulmach/orb"
	"golang.org/x/sync/semaphore"

	"github.com/paulmach/orb/project"
	"github.com/paulmach/orb/simplify"

	"../utils"
	"github.com/paulmach/orb/encoding/mvt"
	"github.com/paulmach/orb/geojson"
)

const tileSize = mvt.DefaultExtent

func buildVectorTiles(outputPath string, collectionsPtr *map[string]*geojson.FeatureCollection, maxLod uint16, worldSize float64, layerSettings *[]layerSetting) {

	for lod := uint16(0); lod <= maxLod; lod++ {
		lodPath := path.Join(outputPath, fmt.Sprintf("%d", lod))
		start := time.Now()

		// create LOD directory
		if !utils.IsDirectory(path.Dir(lodPath)) {
			err := os.MkdirAll(lodPath, os.ModePerm)
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		buildLODVectorTiles(lod, lodPath, collectionsPtr, worldSize, layerSettings)

		fmt.Println("    ✔️  Finished tiles for LOD", lod, "in", time.Now().Sub(start).String())
	}
}

func buildLODVectorTiles(lod uint16, lodDir string, collectionsPtr *map[string]*geojson.FeatureCollection, worldSize float64, layerSettings *[]layerSetting) {
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

	layers.Simplify(simplify.DouglasPeucker(1.0))
	layers.RemoveEmpty(1.0, 1.0)

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

// findLODLayers return a map which includes all layers needed for given LOD
func findLODLayers(allLayersPtr *map[string]*geojson.FeatureCollection, settingsPtr *[]layerSetting, lod uint16) mvt.Layers {

	lodMap := make(map[string]*geojson.FeatureCollection)

	for layerName, fc := range *allLayersPtr {
		layerSet := findLayerSettings(settingsPtr, layerSetting{Layer: layerName, MinZoom: nil, MaxZoom: nil}, layerName)

		if layerSet.MaxZoom == nil && layerSet.MinZoom == nil {
			// both min- and maxzoom are not set
			lodMap[layerName] = utils.DeepCloneFeatureCollection(fc)
		} else if layerSet.MinZoom == nil {
			// only maxzoom is set
			if *layerSet.MaxZoom >= lod {
				lodMap[layerName] = utils.DeepCloneFeatureCollection(fc)
			}
		} else if layerSet.MaxZoom == nil {
			// only minzoom is set
			if *layerSet.MinZoom <= lod {
				lodMap[layerName] = utils.DeepCloneFeatureCollection(fc)
			}
		} else {
			// both min- and maxzoom are set
			if *layerSet.MinZoom <= lod && *layerSet.MaxZoom >= lod {
				lodMap[layerName] = utils.DeepCloneFeatureCollection(fc)
			}
		}
	}
	return mvt.NewLayers(lodMap)
}

func findLayerSettings(allSettings *[]layerSetting, defaults layerSetting, layer string) layerSetting {
	for _, setting := range *allSettings {
		if setting.Layer == layer {
			return setting
		}
	}

	return defaults
}

func createTile(x uint32, y uint32, layers mvt.Layers) ([]byte, error) {
	lClone := utils.DeepCloneLayers(layers)

	projectLayersInPlace(lClone, func(p orb.Point) orb.Point {
		return orb.Point{
			p[0] - float64(x*tileSize),
			p[1] - float64(y*tileSize),
		}
	})

	// add tileSize/4 as padding to make sure geos are not cut directly at the tile border
	lClone.Clip(orb.Bound{Min: orb.Point{-tileSize / 4, -tileSize / 4}, Max: orb.Point{tileSize + tileSize/4, tileSize + tileSize/4}})
	lClone.RemoveEmpty(0, 0)

	data, err := mvt.MarshalGzipped(lClone)
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

func projectLayersInPlace(ls mvt.Layers, p orb.Projection) {
	for _, l := range ls {
		for _, f := range (*l).Features {
			f.Geometry = project.Geometry(f.Geometry, p)
		}
	}
}
