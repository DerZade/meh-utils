package sat

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"log"
	"math"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"../utils"
	"github.com/nfnt/resize"
	"golang.org/x/sync/semaphore"
)

// BuildTileSet builds tiles for given LOD from given combinedSatImage into outputDirectory
func buildTileSet(lod uint8, combinedSatImage *image.RGBA, outputDirectory string) {
	outputDirectory = path.Join(outputDirectory, fmt.Sprintf("%d", lod))

	start := time.Now()

	tilesPerRowCol := int(math.Pow(2, float64(lod)))

	// make col directories
	wg := sync.WaitGroup{}
	for col := 0; col < tilesPerRowCol; col++ {
		wg.Add(1)
		go func(col int) {
			defer wg.Done()
			dirPath := path.Join(outputDirectory, fmt.Sprintf("%d", col))
			if !utils.IsDirectory(dirPath) {
				err := os.MkdirAll(dirPath, os.ModePerm)
				if err != nil {
					log.Fatal(err)
				}
			}
		}(col)
	}

	wg.Wait()

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
			wg2.Add(1)
			go func(col int, row int) {
				defer wg2.Done()
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
				createTile(combinedSatImage, rect, tilePath)
			}(col, row)
		}
	}

	wg2.Wait()

	fmt.Println("✔️  Finished tiles for LOD", lod, "in", time.Now().Sub(start).String())
}

var sem = semaphore.NewWeighted(int64(runtime.NumCPU()))

func createTile(combinedSatImage *image.RGBA, rect image.Rectangle, tilePath string) {
	sem.Acquire(context.Background(), 1)

	subImg := (*combinedSatImage).SubImage(rect)

	img := resize.Resize(256, 256, subImg, resize.MitchellNetravali)

	out, err := os.Create(tilePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	png.Encode(out, img)

	sem.Release(1)
}
