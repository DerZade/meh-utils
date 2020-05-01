package mvt

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

const template = `
{
	"minzoom": 0,
	"maxzoom": %d,
    "vector_layers": [
		%s
	]
}
`

const prefix = `{ "id": "`
const postfix = `", "fields": {} }`

// writeTileJSON writes sat.json containing the maxLod to the sat.json into the outputDirectory
func writeTileJSON(outputDirectory string, maxLod uint16, layerNames []string) {
	var err error

	var items []string

	for i := 0; i < len(layerNames); i++ {
		item := strings.Join([]string{prefix, layerNames[i], postfix}, "")
		items = append(items, item)
	}

	vectorLayerStr := strings.Join(items, ",\n        ")

	f, err := os.Create(path.Join(outputDirectory, "tile.json"))
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteString(fmt.Sprintf(template, maxLod, vectorLayerStr))
	if err != nil {
		fmt.Println(err)
		f.Close()
		return
	}
	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
}
