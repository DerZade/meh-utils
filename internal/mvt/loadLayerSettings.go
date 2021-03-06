package mvt

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

const defaultLayerSettings = `
[
	{ "layer": "debug", "minzoom": 6 },
    { "layer": "locations/hill", "minzoom": 0 },
    { "layer": "locations/vegetationbroadleaf", "minzoom": 0 },
    { "layer": "locations/vegetationvineyard", "minzoom": 0 },
    { "layer": "locations/viewpoint", "minzoom": 0 },
    { "layer": "locations/namecity", "minzoom": 0 },
    { "layer": "locations/namecitycapital", "minzoom": 0 },
    { "layer": "locations/namevillage", "minzoom": 0 },
    { "layer": "locations/namelocal", "minzoom": 0 },
    { "layer": "locations/namemarine", "minzoom": 0 },
    { "layer": "locations/airport", "minzoom": 0 },
    { "layer": "bunker", "minzoom": 0 },
    { "layer": "chapel", "minzoom": 4 },
    { "layer": "church", "minzoom": 4 },
    { "layer": "cross", "minzoom": 4 },
    { "layer": "fuelstation", "minzoom": 4 },
    { "layer": "lighthouse", "minzoom": 4 },
    { "layer": "rock", "minzoom": 5 },
    { "layer": "shipwreck", "minzoom": 4 },
    { "layer": "transmitter", "minzoom": 4 },
    { "layer": "tree", "minzoom": 6 },
    { "layer": "bush", "minzoom": 8 },
    { "layer": "watertower", "minzoom": 4 },
    { "layer": "fortress", "minzoom": 4 },
    { "layer": "fountain", "minzoom": 4 },
    { "layer": "quay", "minzoom": 4 },
    { "layer": "hospital", "minzoom": 4 },
    { "layer": "busstop", "minzoom": 4 },
    { "layer": "stack", "minzoom": 4 },
    { "layer": "ruin", "minzoom": 4 },
    { "layer": "tourism", "minzoom": 4 },
    { "layer": "powersolar", "minzoom": 4 },
    { "layer": "powerwave", "minzoom": 4 },
    { "layer": "powerwind", "minzoom": 4 },
    { "layer": "view-tower", "minzoom": 4 },
    { "layer": "runway", "minzoom": 0 },
    { "layer": "powerline", "minzoom": 4 },
    { "layer": "railway", "minzoom": 4 },
    { "layer": "house", "minzoom": 2 },
    { "layer": "roads/main_road", "minzoom": 3 },
    { "layer": "roads/main_road-bridge", "minzoom": 3 },
    { "layer": "roads/road", "minzoom": 3 },
    { "layer": "roads/road-bridge", "minzoom": 3 },
    { "layer": "roads/track", "minzoom": 3 },
    { "layer": "roads/track-bridge", "minzoom": 3 },
    { "layer": "roads/trail", "minzoom": 4 },
    { "layer": "roads/trail-bridge", "minzoom": 4 },
    { "layer": "water", "minzoom": 0 },
    { "layer": "forest", "minzoom": 3 },
    { "layer": "rocks", "minzoom": 3 },
    { "layer": "mount", "minzoom": 2 },
    { "layer": "contours/01", "minzoom": 8 },
    { "layer": "contours/05", "minzoom": 7, "maxzoom": 7 },
    { "layer": "contours/10", "minzoom": 5, "maxzoom": 6 },
    { "layer": "contours/50", "minzoom": 3, "maxzoom": 4 },
    { "layer": "contours/100", "minzoom": 0, "maxzoom": 2 }
]`

type layerSetting struct {
	Layer   string `json:"layer"`
	MinZoom *uint8 `json:"minzoom,omitempty"`
	MaxZoom *uint8 `json:"maxzoom,omitempty"`
}

func loadLayerSettings(filePath string) []layerSetting {

	var val []layerSetting
	var byteValue []byte

	if filePath == "" {
		byteValue = []byte(defaultLayerSettings)
	} else {
		// Open our jsonFile
		jsonFile, err := os.Open(filePath)
		// if we os.Open returns an error then handle it
		if err != nil {
			log.Fatal(err)
		}

		// defer the closing of our jsonFile so that we can parse it later on
		defer jsonFile.Close()

		// read our opened jsonFile as a byte array.
		byteValue, _ = ioutil.ReadAll(jsonFile)
	}

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'val'
	json.Unmarshal(byteValue, &val)

	return val
}
