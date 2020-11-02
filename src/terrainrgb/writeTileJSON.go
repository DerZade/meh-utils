package terrainrgb

import (
	"fmt"
	"log"
	"os"
	"path"
)

const template = `
{
	"minzoom": 0,
	"maxzoom": %d
}
`

// writeTileJSON writes sat.json containing the maxLod to the sat.json into the outputDirectory
func writeTileJSON(outputDirectory string, maxLod uint8) {
	var err error

	f, err := os.Create(path.Join(outputDirectory, "tile.json"))
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteString(fmt.Sprintf(template, maxLod))
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
