package sat

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"math"
	"os"
	"path"
	"sync"
	"time"

	"../utils"
	"github.com/nfnt/resize"
)

// BuildTileSet builds tiles for given LOD from given combinedSatImage into outputDirectory
func buildTileSet(lod uint8, combinedSatImage *image.RGBA, outputDirectory string, wg *sync.WaitGroup) {
	outputDirectory = path.Join(outputDirectory, fmt.Sprintf("%d", lod))

	start := time.Now()

	tilesPerRowCol := int(math.Pow(2, float64(lod)))

	// make col directories
	for col := 0; col < tilesPerRowCol; col++ {
		dirPath := path.Join(outputDirectory, fmt.Sprintf("%d", col))

		if !utils.IsDirectory(dirPath) {
			err := os.MkdirAll(dirPath, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	width := (*combinedSatImage).Bounds().Dy()
	height := (*combinedSatImage).Bounds().Dx()

	tileWidth := width / tilesPerRowCol
	tileHeight := height / tilesPerRowCol

	// remaining pixels
	widthRemainder := width % tilesPerRowCol
	heightRemainder := height % tilesPerRowCol

	wg2 := sync.WaitGroup{}

	for col := 0; col < tilesPerRowCol; col++ {
		for row := 0; row < tilesPerRowCol; row++ {
			tilePath := path.Join(outputDirectory, fmt.Sprintf("%d", col), fmt.Sprintf("%d.png", row))
			x := tileWidth * col
			y := tileHeight * row
			w := tileWidth
			h := tileHeight
			p := image.Point{x, y}

			// if we have any remaining pixels we'll distrubute them to the first rows / cols
			if widthRemainder > col+1 {
				w++
			}
			if heightRemainder > row+1 {
				h++
			}

			rect := image.Rectangle{p, p.Add(image.Point{w, h})}

			subImg := (*combinedSatImage).SubImage(rect)

			img := resize.Resize(256, 256, subImg, resize.MitchellNetravali)

			wg2.Add(1)

			go func(tilePath string) {
				defer wg2.Done()
				out, err := os.Create(tilePath)
				if err != nil {
					fmt.Println(err)
					return
				}
				png.Encode(out, img)
			}(tilePath)
		}
	}

	wg2.Wait()

	fmt.Println("✔️  Finished tiles for LOD", lod, "in", time.Now().Sub(start).String())
	defer wg.Done()
}
