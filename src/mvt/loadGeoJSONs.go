package mvt

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	geojson "github.com/paulmach/orb/geojson"
)

func loadGeoJSONs(inputPath string, layers *map[string]*geojson.FeatureCollection) {
	var layersMux = sync.Mutex{}
	filePaths := []string{}

	pattern, _ := regexp.Compile("\\.geojson\\.gz$")

	err := filepath.Walk(inputPath, func(path string, f os.FileInfo, err error) error {
		if pattern.MatchString(path) {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	waitGrp := sync.WaitGroup{}

	for _, layerPath := range filePaths {
		waitGrp.Add(1)
		go func(path string) {
			defer waitGrp.Done()

			layerName := pathToLayerName(path, inputPath)
			fc := readGzippedGeoJSON(path)

			// we want to have the color of the houses as a rgba-string not as an array [r,g,b] with r, g and b beeing numbers from 0 to 255
			if layerName == "house" {
				for _, feature := range (*fc).Features {
					color := feature.Properties["color"].([]interface{})

					feature.Properties["color"] = fmt.Sprintf("rgb(%.0f, %.0f, %.0f)", color[0], color[1], color[2])
				}
			}

			layersMux.Lock()
			(*layers)[layerName] = fc
			layersMux.Unlock()

		}(layerPath)
	}

	waitGrp.Wait()
}

func pathToLayerName(filePath string, geojsonPath string) string {
	r, _ := filepath.Rel(geojsonPath, filePath)
	return filepath.ToSlash(strings.Replace(r, ".geojson.gz", "", -1))
}

func readGzippedGeoJSON(geoJSONPath string) *geojson.FeatureCollection {
	file, err := os.Open(geoJSONPath)

	if err != nil {
		log.Fatal(err)
	}

	gz, err := gzip.NewReader(file)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	defer gz.Close()

	var features []geojson.Feature

	json.NewDecoder(gz).Decode(&features)

	pointers := make([]*geojson.Feature, len(features))

	for i := 0; i < len(features); i++ {
		pointers[i] = &features[i]
	}

	return &geojson.FeatureCollection{Features: pointers}
}
