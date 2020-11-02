package terrainrgb

import (
	"compress/gzip"
	"log"
	"os"

	dem "../dem"
)

func loadDEM(path string) dem.EsriASCIIRaster {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	gz, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}

	raster, err := dem.ParseEsriASCIIRaster(gz)
	if err != nil {
		log.Fatal(err)
	}

	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = gz.Close()
	if err != nil {
		log.Fatal(err)
	}

	return raster
}
