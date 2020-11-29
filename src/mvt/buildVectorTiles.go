package mvt

import (
	"context"
	"fmt"
	"math"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/paulmach/orb/clip"
	"github.com/paulmach/orb/simplify"

	"golang.org/x/sync/semaphore"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/mvt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/project"

	"../utils"
)

const tileSize = mvt.DefaultExtent

func buildVectorTiles(outputPath string, collectionsPtr *map[string]*geojson.FeatureCollection, maxLod uint8, worldSize float64, layerSettings *[]layerSetting) {
	allLayers := make(map[string]*mvt.Layer)

	// set layer version to v2
	for _, l := range mvt.NewLayers(*collectionsPtr) {
		l.Version = 2
		allLayers[l.Name] = l
	}

	// project features from arma coordinates to pixel coordinates
	tilesPerRowCol := uint32(math.Pow(2, float64(maxLod))) // how many tiles each row has
	pixels := uint64(tileSize) * uint64(tilesPerRowCol)    // how many pixels one row / col has
	factor := float64(pixels) / worldSize                  // factor to convert from arma coordinates to pixel Coords
	projectLayersInPlace(allLayers, func(p orb.Point) orb.Point {
		return orb.Point{
			p[0] * factor,
			(worldSize - p[1]) * factor,
		}
	})

	for lod := maxLod; lod >= 0; lod-- {
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

		// project from last LOD to this LOD
		if lod != maxLod {
			projectLayersInPlace(allLayers, func(p orb.Point) orb.Point {
				return orb.Point{
					p[0] / 2,
					p[1] / 2,
				}
			})
		}

		// simplify layers
		if lod != maxLod {
			for _, layer := range allLayers {
				if strings.HasPrefix(layer.Name, "locations") {
					continue
				}

				switch layer.Name {
				case "bunker", "chapel", "church", "cross", "fuelstation", "lighthouse", "rock", "shipwreck", "transmitter", "watertower", "fortress", "fountain", "view-tower", "quay", "hospital", "busstop", "stack", "ruin", "tourism", "powersolar", "powerwave", "powerwind", "tree", "bush":
					continue
				case "mount":
					simplifyMounts(layer, 1000)
				case "railway", "powerline":
					layer.Simplify(simplify.DouglasPeucker(1))
				case "house":
					layer.RemoveEmpty(0, 200)
				case "contours":
					layer.Simplify(simplify.DouglasPeucker(2))
					layer.RemoveEmpty(100, 0)
				case "water":
					layer.Simplify(simplify.DouglasPeucker(2))
					layer.RemoveEmpty(0, 0)

					// RemoveEmpty does not remove rings of holes smaller
					// than threshold so we'll have to do that ourselves
					for _, feature := range layer.Features {
						poly := feature.Geometry.(orb.Polygon)

						keepCount := 0
						for _, r := range poly {
							if planar.Length(r) < 150 {
								continue
							}

							poly[keepCount] = r
							keepCount++
						}

						feature.Geometry = poly[:keepCount]
					}
				default:
					layer.Simplify(simplify.DouglasPeucker(1))
					layer.RemoveEmpty(100, 200)
				}
				// TODO: Simplify streets
			}
		}

		lodLayers := findLODLayers(allLayers, layerSettings, lod, maxLod)
		fillContourLayers(lodLayers, allLayers["contours"])

		buildLODVectorTiles(lod, lodDir, lodLayers)

		fmt.Println("    ✔️  Finished tiles for LOD", lod, "in", time.Now().Sub(start).String())

		// if we don't manually break uint will bamboozle
		// us because 0-1 is just 255 and that's >= 0
		if lod == 0 {
			break
		}
	}
}

func buildLODVectorTiles(lod uint8, lodDir string, layers mvt.Layers) {
	// how many tiles one row / col has
	tilesPerRowCol := uint32(math.Pow(2, float64(lod)))

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
func findLODLayers(allLayers map[string]*mvt.Layer, settingsPtr *[]layerSetting, lod uint8, maxLod uint8) mvt.Layers {

	lodLayers := mvt.Layers{}

	for _, layer := range allLayers {
		layerName := layer.Name

		// layer "contours" will never be drawn
		if layerName == "contours" {
			continue
		}

		// find layer settings for layerName
		layerMinZoom := lod
		layerMaxZoom := lod
		for _, setting := range *settingsPtr {
			if setting.Layer == layerName {
				if setting.MinZoom != nil {
					layerMinZoom = *setting.MinZoom
				}
				if setting.MaxZoom != nil {
					layerMaxZoom = *setting.MaxZoom
				}
				break
			}
		}

		// make sure minZoom is not bigger than the maximum calculated layer
		if layerMinZoom > maxLod {
			layerMinZoom = maxLod
		}

		if lod < layerMinZoom || lod > layerMaxZoom {
			continue
		}

		lodLayers = append(lodLayers, layer)
	}

	return lodLayers
}

func fillContourLayers(lodLayers mvt.Layers, contours *mvt.Layer) {
	pattern, _ := regexp.Compile("^contours/\\d+$")

	for index, layer := range lodLayers {

		if !pattern.MatchString(layer.Name) {
			continue
		}

		interval, err := strconv.Atoi(layer.Name[len("contours/"):])
		if err != nil {
			continue
		}

		newLayer := mvt.Layer{
			Name:     layer.Name,
			Version:  layer.Version,
			Extent:   layer.Extent,
			Features: contours.Features,
		}

		if interval == 1 {
			newLayer.Features = contours.Features
		} else {
			intervalFeatures := make([]*geojson.Feature, 0)

			for _, f := range contours.Features {
				elev := f.Properties["dem_elevation"].(int)

				if elev%interval == 0 {
					intervalFeatures = append(intervalFeatures, f)
				}
			}

			newLayer.Features = intervalFeatures

		}
		lodLayers[index] = &newLayer
	}
}

func createTile(x uint32, y uint32, layers mvt.Layers) ([]byte, error) {
	xOffset := float64(x * tileSize)
	yOffset := float64(y * tileSize)

	tileBound := orb.Bound{
		Min: orb.Point{mvt.MapboxGLDefaultExtentBound.Min[0] + xOffset, mvt.MapboxGLDefaultExtentBound.Min[1] + yOffset},
		Max: orb.Point{mvt.MapboxGLDefaultExtentBound.Max[0] + xOffset, mvt.MapboxGLDefaultExtentBound.Max[1] + yOffset},
	}

	// projection from global coordinate space to tile coordinate space
	tileProjection := func(p orb.Point) orb.Point {
		return orb.Point{
			p[0] - xOffset,
			p[1] - yOffset,
		}
	}

	tileLayers := make(mvt.Layers, len(layers))

	for index, layer := range layers {
		features := make([]*geojson.Feature, len(layer.Features))
		keep := 0

		for _, f := range layer.Features {
			geo := orb.Clone(f.Geometry)
			geo = clip.Geometry(tileBound, geo)

			if geo == nil {
				continue
			}

			// project coordinates from global to tile coordinate space
			geo = project.Geometry(geo, tileProjection)

			// create feature
			newFeature := geojson.NewFeature(geo)
			newFeature.ID = f.ID
			newFeature.Type = f.Type
			newFeature.Properties = f.Properties.Clone()
			copy(newFeature.BBox, f.BBox)

			// save feature to new layer
			features[keep] = newFeature
			keep++
		}

		tileLayers[index] = &mvt.Layer{
			Name:     layer.Name,
			Version:  layer.Version,
			Extent:   layer.Extent,
			Features: features[:keep],
		}
	}

	// marshal tile
	data, err := mvt.MarshalGzipped(tileLayers)
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
func projectLayersInPlace(layers map[string]*mvt.Layer, projection orb.Projection) {
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
