package utils

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"math"
	"runtime"
	"sync"

	"github.com/nfnt/resize"
	"golang.org/x/sync/semaphore"

	"../mbtiles"
)

// BuildTileSet builds tiles for given LOD from given image into outputDirectory
func BuildTileSet(lod uint8, combinedSatImage *image.RGBA, mbt *mbtiles.MBTiles) {
	// outputDirectory = path.Join(outputDirectory, fmt.Sprintf("%d", lod))

	tilesPerRowCol := int(math.Pow(2, float64(lod)))

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

				sem.Acquire(context.Background(), 1)

				subImg := (*combinedSatImage).SubImage(rect)

				img := resize.Resize(256, 256, subImg, resize.MitchellNetravali)

				buf := new(bytes.Buffer)
				png.Encode(buf, img)

				mbt.InsertTile(uint(lod), uint(col), uint(row), buf.Bytes())

				sem.Release(1)
			}(col, row)
		}
	}

	wg2.Wait()
}

var sem = semaphore.NewWeighted(int64(runtime.NumCPU()))
