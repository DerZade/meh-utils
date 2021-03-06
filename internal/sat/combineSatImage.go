package sat

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"os"
	"path"
	"sync"

	"github.com/gruppe-adler/meh-utils/internal/utils"
)

// combineSatImage combines the 4x4 tiles form the inputDir to a new image.RGBA
func combineSatImage(inputDir string) *image.RGBA {
	// holds heights of all rows
	heights := []uint{0, 0, 0, 0}
	widths := []uint{0, 0, 0, 0}

	var images [4][4]image.Image

	// load all images
	waitGrp := sync.WaitGroup{}
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			waitGrp.Add(1)

			go func(col int, row int) {
				defer waitGrp.Done()

				// open image
				filePath := path.Join(inputDir, fmt.Sprintf("%d", col), fmt.Sprintf("%d.png", row))
				file, err := os.Open(filePath)
				if err != nil {
					log.Fatal(err)
				}
				img, _, err := image.Decode(file)
				if err != nil {
					log.Fatal(err)
				}

				file.Close()
				if err != nil {
					log.Fatal(err)
				}

				// save in structure
				images[col][row] = img
			}(col, row)
		}
	}
	waitGrp.Wait()

	// update heights / widths
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			img := images[col][row]

			imgWidth := uint(img.Bounds().Dx())
			if imgWidth > widths[col] {
				widths[col] = imgWidth
			}
			imgHeight := uint(img.Bounds().Dy())
			if imgHeight > heights[row] {
				heights[row] = imgHeight
			}
		}
	}

	width := int(utils.Sum(widths))
	height := int(utils.Sum(heights))

	// create new img
	combinedImage := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{width, height}})

	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			img := &images[col][row]

			x := int(utils.Sum(widths[0:col]))
			y := int(utils.Sum(heights[0:row]))
			upperLeftPoint := image.Point{x, y}
			r := image.Rectangle{upperLeftPoint, upperLeftPoint.Add((*img).Bounds().Size())}

			draw.Draw(combinedImage, r, *img, image.Point{0, 0}, draw.Src)
		}
	}

	return combinedImage
}
