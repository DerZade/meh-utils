package sat

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"../utils"
	"../validate"
)

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
	combinedImg := combineSatImage(inputDir)
	fmt.Println("✔️  Combined satellite image in", time.Now().Sub(timer).String())

	maxLod := calcMaxLod(combinedImg)

	fmt.Println("ℹ️  Calculated max lod:", maxLod)

	timer = time.Now()
	fmt.Println("▶️  Building tiles")
	for lod := uint8(0); lod <= maxLod; lod++ {
		timer2 := time.Now()
		buildTileSet(lod, combinedImg, *outputPtr)
		fmt.Println("    ✔️  Finished tiles for LOD", lod, "in", time.Now().Sub(timer2).String())
	}
	fmt.Println("✔️  Built sat tiles in", time.Now().Sub(timer).String())

	timer = time.Now()
	fmt.Println("▶️  Creating sat.json")
	writeTileJSON(*outputPtr, maxLod)
	fmt.Println("✔️  Created tile.json in", time.Now().Sub(timer).String())

	fmt.Printf("\n    🎉  Finished in %s\n", time.Now().Sub(start).String())
}
