package preview

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path"
	"time"

	"github.com/gruppe-adler/meh-utils/internal/utils"
	"github.com/gruppe-adler/meh-utils/internal/validate"
	"github.com/nfnt/resize"
)

var sizes = []uint{128, 256, 512, 1024}

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

	// validate input directory structure
	err := validate.MehDirectory(*inputPtr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✔️  Validated input directory structure")

	timer = time.Now()
	fmt.Println("▶️  Loading preview image")

	file, err := os.Open(path.Join(*inputPtr, "preview.png"))
	if err != nil {
		log.Fatal(err)
	}
	previewImage, _, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	file.Close()
	if err != nil {
		log.Fatal(err)
	}

	previewHeight := previewImage.Bounds().Dy()
	previewWidth := previewImage.Bounds().Dx()

	fmt.Println("✔️  Loaded preview image in", time.Now().Sub(timer).String())

	timer = time.Now()
	fmt.Println("▶️  Writing original preview image to output")
	saveImage(path.Join(*outputPtr, "preview.png"), previewImage)

	fmt.Println("✔️  Wrote original preview image in", time.Now().Sub(timer).String())

	for _, size := range sizes {
		timer = time.Now()
		fmt.Printf("▶️  Building x%d image\n", size)

		factor := float64(size) / float64(previewHeight)
		w := uint(float64(previewWidth) * factor)

		img := resize.Resize(size, w, previewImage, resize.MitchellNetravali)
		saveImage(path.Join(*outputPtr, fmt.Sprintf("preview_%d.png", size)), img)

		fmt.Printf("✔️  Built x%d in %s\n", size, time.Now().Sub(timer).String())
	}

	fmt.Printf("\n    🎉  Finished in %s\n", time.Now().Sub(start).String())
}

func saveImage(path string, img image.Image) {
	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	png.Encode(out, img)

	err = out.Close()
	if err != nil {
		log.Fatal(err)
	}
}
