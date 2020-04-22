package metajson

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Grid represents the config of the different grids
type Grid struct {
	Format  string  `json:"format"`
	FormatX string  `json:"formatX"`
	FormatY string  `json:"formatY"`
	StepX   float64 `json:"stepX"`
	StepY   float64 `json:"stepY"`
	ZoomMax float64 `json:"zoomMax"`
}

// MetaJSON represents the structure of the meta.json outputted by grad_meh
type MetaJSON struct {
	Author          string  `json:"author"`
	DisplayName     string  `json:"displayName"`
	ElevationOffset float64 `json:"elevationOffset"`
	GridOffsetY     float64 `json:"gridOffsetX"`
	GridOffsetX     float64 `json:"gridOffsetY"`
	Grids           []Grid  `json:"grids"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Version         float64 `json:"version"`
	WorldName       string  `json:"worldName"`
	WorldSize       float64 `json:"worldSize"`
}

// Read meta.json from given path
func Read(metaJSONPath string) (MetaJSON, error) {

	var val MetaJSON

	// Open our jsonFile
	jsonFile, err := os.Open(metaJSONPath)
	// if we os.Open returns an error then handle it
	if err != nil {
		return val, err
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// read our opened jsonFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'val'
	json.Unmarshal(byteValue, &val)

	return val, nil
}
