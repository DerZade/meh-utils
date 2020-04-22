package mvt

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"../metajson"

	"../utils"
	"../validate"
	geojson "github.com/paulmach/orb/geojson"
)

// Run is the program's entrypoint
func Run(flagSet *flag.FlagSet) {

	collections  := make(map[string]*geojson.FeatureCollection)
	var timer time.Time
	start := time.Now()

	outputPtr := flagSet.String("out", "", "Path to output directory")
	inputPtr := flagSet.String("in", "", "Path to grad_meh map directory")
	layerSettingsPtr := flagSet.String("layer-settings", "", "Path to layer_settings.json file")

	flagSet.Parse(os.Args[2:])

	// make sure both flags are present
	if *outputPtr == "" || *inputPtr == "" {
		flagSet.PrintDefaults()
		os.Exit(1)
	}

	// make sure layerSettings is either "" of a valid file
	if *layerSettingsPtr != "" && !utils.IsFile(*layerSettingsPtr) {
		log.Fatal(errors.New("LayerSettings is not a valid file"))
	}

	// make sure given output directory is a valid directory
	if !utils.IsDirectory(*outputPtr) {
		log.Fatal(errors.New("Output directory doesn't exists"))
	}

	// validate input directory structure
	err := validate.MehDirectory(*inputPtr)
	if err != nil {
		log.Fatal(errors.New("Output directory doesn't exists"))
	}
	fmt.Println("✔️  Validated input directory structure")

	// load meta.json
	timer = time.Now()
	fmt.Println("▶️  Loading meta.json")
	meta, err := metajson.Read(path.Join(*inputPtr, "meta.json"))
	if err != nil {
		log.Fatal(errors.New("Failed to read meta.json"))
	}
	fmt.Println("✔️  Loaded meta.json in", time.Now().Sub(timer).String())

	// load layerSettings
	layerSettings := loadLayerSettings(*layerSettingsPtr)

	// contour lines
	timer = time.Now()
	fmt.Println("▶️  Building countour lines")
	buildContours(path.Join(*inputPtr, "dem.asc.gz"), meta.ElevationOffset)
	fmt.Println("✔️  Built contour lines in", time.Now().Sub(timer).String())

	// loading GeoJSONSs
	timer = time.Now()
	fmt.Println("▶️  Loading GeoJSONs")
	loadGeoJSONs(path.Join(*inputPtr, "geojson"), &collections )
	fmt.Println("✔️  Loaded layers from geojsons in", time.Now().Sub(timer).String())

	// print loaded layers
	fmt.Printf("ℹ️  Loaded the following layers (%d): ", len(collections))
	layerNames := make([]string, 0, len(collections))
	for layerName := range collections {
		layerNames = append(layerNames, layerName)
	}
	sort.Strings(layerNames)
	fmt.Printf("%s\n", strings.Join(layerNames, ", "))

	// convert layers
	timer = time.Now()
	fmt.Println("▶️  Converting coordinates of all layers to Web Mercator (EPSG:3857)")
	waitGrp := sync.WaitGroup{}
	for _, ptr := range collections {
		waitGrp.Add(1)
		go func(ptr *geojson.FeatureCollection) {
			convertLayer(ptr, meta.WorldSize)
			defer waitGrp.Done()
		}(ptr)
	}
	waitGrp.Wait()
	fmt.Println("✔️  Converted coordinates of all layers in", time.Now().Sub(timer).String())

	// build mvts
	timer = time.Now()
	fmt.Println("▶️  Building mapbox vector tiles")
	buildVectorTiles(*outputPtr, &collections, meta.WorldSize, &layerSettings)
	fmt.Println("✔️  Built mapbox vector tiles in", time.Now().Sub(timer).String())

	fmt.Printf("\n    🎉  Finished in %s\n", time.Now().Sub(start).String())
}
