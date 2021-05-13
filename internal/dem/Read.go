package dem

import (
	"compress/gzip"
	"log"
	"os"
)

// Read digital elevation model from given path
func Read(path string) EsriASCIIRaster {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	gz, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}

	raster, err := ParseEsriASCIIRaster(gz)
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
