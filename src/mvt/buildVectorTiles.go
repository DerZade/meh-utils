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

	"../utils"
	"github.com/paulmach/orb/encoding/mvt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/maptile"
	"golang.org/x/sync/semaphore"
)

func buildVectorTiles(outputPath string, collectionsPtr *map[string]*geojson.FeatureCollection, worldSize float64, layerSettings *[]layerSetting) {

	// TODO: calc maxLoad dynamically with worldSize
	maxLod := uint32(7)

	for lod := uint32(0); lod <= maxLod; lod++ {
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

		buildLODVectorTiles(lod, lodPath, collectionsPtr, layerSettings)

		fmt.Println("    ✔️  Finished tiles for LOD", lod, "in", time.Now().Sub(start).String())
	}
}

func buildLODVectorTiles(lod uint32, lodDir string, collectionsPtr *map[string]*geojson.FeatureCollection, layerSettings *[]layerSetting) {
	tilesPerRowCol := uint32(math.Pow(2, float64(lod)))

	colWaitGrp := sync.WaitGroup{}

	for col := uint32(0); col < tilesPerRowCol; col++ {
		colWaitGrp.Add(1)
		go func(col uint32) {
			defer colWaitGrp.Done()
			// create colDirectory
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
					tilePath := path.Join(colPath, fmt.Sprintf("%d.pbf", row))
					createTile(lod, col, row, collectionsPtr, tilePath, layerSettings)
				}(row)
			}

			rowWaitGrp.Wait()
		}(col)
	}

	colWaitGrp.Wait()
}

var sem = semaphore.NewWeighted(int64(runtime.NumCPU()))

func createTile(z uint32, x uint32, y uint32, collectionsPtr *map[string]*geojson.FeatureCollection, tilePath string, layerSettings *[]layerSetting) {

	sem.Acquire(context.Background(), 1)

	layers := getLayers(collectionsPtr, layerSettings, z, x, y)

	data, err := mvt.MarshalGzipped(layers)
	if err != nil {
		fmt.Printf("Error while creating tile %d/%d/%d\n", z, x, y)
		return
	}

	err = writeTile(tilePath, data)
	if err != nil {
		fmt.Printf("Error while creating tile %d/%d/%d\n", z, x, y)
	}

	sem.Release(1)
}

func getLayers(collectionsPtr *map[string]*geojson.FeatureCollection, settingsPtr *[]layerSetting, z uint32, x uint32, y uint32) mvt.Layers {
	layers := mvt.NewLayers(findLODLayers(collectionsPtr, settingsPtr, uint16(z)))

	layers.ProjectToTile(maptile.New(x, y, maptile.Zoom(z)))

	return layers
}

var lodLayerCache = make(map[uint16]map[string]*geojson.FeatureCollection)
var mutex = sync.Mutex{}

// findLodLayers return a map which includes all layers needed for given lod
func findLODLayers(allLayersPtr *map[string]*geojson.FeatureCollection, settingsPtr *[]layerSetting, lod uint16) map[string]*geojson.FeatureCollection {

	mutex.Lock()

	if lodLayerCache[lod] == nil {
		lodMap := make(map[string]*geojson.FeatureCollection)

		for layerName, fc := range *allLayersPtr {
			layerSet := findLayerSettings(settingsPtr, layerSetting{Layer: layerName, MinZoom: nil, MaxZoom: nil}, layerName)

			if layerSet.MaxZoom == nil && layerSet.MinZoom == nil {
				// both min- and maxzoom are not set
				lodMap[layerName] = fc
			} else if layerSet.MinZoom == nil {
				// only maxzoom is set
				if *layerSet.MaxZoom >= lod {
					lodMap[layerName] = fc
				}
			} else if layerSet.MaxZoom == nil {
				// only minzoom is set
				if *layerSet.MinZoom <= lod {
					lodMap[layerName] = fc
				}
			} else {
				// both min- and maxzoom are set
				if *layerSet.MinZoom <= lod && *layerSet.MaxZoom >= lod {
					lodMap[layerName] = fc
				}
			}
		}

		lodLayerCache[lod] = lodMap
	}

	mutex.Unlock()

	return lodLayerCache[lod]

}

func findLayerSettings(allSettings *[]layerSetting, defaults layerSetting, layer string) layerSetting {
	for _, setting := range *allSettings {
		if setting.Layer == layer {
			return setting
		}
	}

	return defaults
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
