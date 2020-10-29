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

	"../utils"
	"github.com/nfnt/resize"
	"golang.org/x/sync/semaphore"
)

// BuildTileSet builds tiles for given LOD from given combinedSatImage into outputDirectory
func buildTileSet(lod uint8, combinedSatImage *image.RGBA, outputDirectory string) {
	outputDirectory = path.Join(outputDirectory, fmt.Sprintf("%d", lod))

	tilesPerRowCol := uint(math.Pow(2, float64(lod)))

	// make col directories
	wg := sync.WaitGroup{}
	for col := uint(0); col < tilesPerRowCol; col++ {
		wg.Add(1)
		go func(col uint) {
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

	resizedImg := resize.Resize(256*tilesPerRowCol, 256*tilesPerRowCol, combinedSatImage, resize.MitchellNetravali).(*image.RGBA)

	wg2 := sync.WaitGroup{}

	for col := uint(0); col < tilesPerRowCol; col++ {
		for row := uint(0); row < tilesPerRowCol; row++ {
			wg2.Add(1)
			go func(col, row uint) {
				defer wg2.Done()
				tilePath := path.Join(outputDirectory, fmt.Sprintf("%d", col), fmt.Sprintf("%d.png", row))

				createTile(resizedImg, col, row, tilePath)
			}(col, row)
		}
	}

	wg2.Wait()
}

var sem = semaphore.NewWeighted(int64(runtime.NumCPU() * 2))

func createTile(combinedSatImage *image.RGBA, col, row uint, tilePath string) {
	sem.Acquire(context.Background(), 1)

	x := int(256 * col)
	y := int(256 * row)
	p := image.Point{x, y}

	rect := image.Rectangle{p, p.Add(image.Point{256, 256})}

	subImg := (*combinedSatImage).SubImage(rect)

	out, err := os.Create(tilePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	png.Encode(out, subImg)

	sem.Release(1)
}
