package sat

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path"
	"sync"
	"time"

	"../utils"
	"../validate"
	"github.com/nfnt/resize"
)

const tileSizeInPx = 256

// CombineSatImage combines the 4x4 tiles form the inputDir to a new image.RGBA
func CombineSatImage(inputDir string) *image.RGBA {
	// holds heights of all rows
	heights := []uint{0, 0, 0, 0}
	widths := []uint{0, 0, 0, 0}

	var images [4][4]image.Image

	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
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

			// save in structure
			images[col][row] = img

			// update col width / row height
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

	// TODO: Trim img

	return combinedImage
}

// CalcMaxLod calculates maximum LOD based on the width of the combinedSatImage
func calcMaxLod(combinedSatImage *image.RGBA) uint8 {
	w := float64(combinedSatImage.Bounds().Dy())

	tilesPerRowCol := math.Ceil(w / tileSizeInPx)

	return uint8(math.Ceil(math.Log2(tilesPerRowCol)))
}

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

// WriteSatJSON writes sat.json containing the maxLod to the sat.json into the outputDirectory
func writeSatJSON(outputDirectory string, maxLod uint8) {
	var err error

	f, err := os.Create(path.Join(outputDirectory, "sat.json"))
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteString(fmt.Sprintf("{ \"maxLod\": %d }", maxLod))
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

// Run is the program's entrypoint
func Run(flagSet *flag.FlagSet) {

	var timer time.Time
	start := time.Now()

	outputPtr := flagSet.String("out", "", "Path to output directory")
	inputPtr := flagSet.String("in", "", "Path to grad_meh map directory")

	flagSet.Parse(os.Args[2:])

	// make sure both flags are present
	if *outputPtr == "" || *inputPtr == "" {
		flagSet.PrintDefaults()
		os.Exit(1)
	}

	// make sure given output directory is a valid directory
	if !utils.IsDirectory(*outputPtr) {
		log.Fatal(errors.New("Output directory doesn't exists"))
	}

	inputDir := path.Join(*inputPtr, "sat")

	// validate input directory structure
	err := validate.SatDirectory(inputDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✔️  Validated input directory structure")

	timer = time.Now()
	fmt.Println("▶️  Combining satellite image")
	combinedImg := CombineSatImage(inputDir)
	fmt.Println("✔️  Finished combining satellite image in", time.Now().Sub(timer).String())

	var maxLod uint8
	maxLod = calcMaxLod(combinedImg)

	fmt.Println("ℹ️  Calculated max lod:", maxLod)

	var wg sync.WaitGroup
	fmt.Println("▶️  Building tiles")
	for lod := uint8(0); lod <= maxLod; lod++ {
		wg.Add(1)
		go buildTileSet(lod, combinedImg, *outputPtr, &wg)

	}

	wg.Wait()

	timer = time.Now()
	fmt.Println("▶️  Creating sat.json")
	writeSatJSON(*outputPtr, maxLod)
	fmt.Println("✔️  Created sat.json in", time.Now().Sub(timer).String())

	fmt.Printf("\n    🎉  Finished in %s\n", time.Now().Sub(start).String())
}
