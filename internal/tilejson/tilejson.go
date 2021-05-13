package tilejson

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/gruppe-adler/meh-utils/internal/metajson"
)

// Write a tile.json
func Write(outputDirectory string, maxLod uint8, meta metajson.MetaJSON, layerName string, vectorLayerNames []string) error {
	var err error

	// build vector layers
	vectorLayers := make([]VectorLayer, len(vectorLayerNames))
	for i, layerName := range vectorLayerNames {
		fields, found := vectorLayerFields[layerName]

		if !found {
			fields = map[string]string{}
		}

		vectorLayers[i] = VectorLayer{
			ID:     layerName,
			Fields: fields,
		}
	}

	obj := TileJSON{
		TileJSON:     "2.2.0",
		Name:         fmt.Sprintf("%s %s Tiles", meta.DisplayName, layerName),
		Description:  fmt.Sprintf("%s Tiles of the Arma 3 Map '%s' from %s", layerName, meta.DisplayName, meta.Author),
		Scheme:       "xyz",
		Minzoom:      0,
		Maxzoom:      maxLod,
		VectorLayers: vectorLayers,
	}

	// create file
	f, err := os.Create(path.Join(outputDirectory, "tile.json"))
	if err != nil {
		return err
	}

	// marshal
	bytes, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return err
	}

	// write file
	_, err = f.Write(bytes)
	if err != nil {
		return err
	}

	// close file
	err = f.Close()
	if err != nil {
		return err
	}

	return err
}
