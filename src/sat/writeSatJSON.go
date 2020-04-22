package sat

import (
	"fmt"
	"log"
	"os"
	"path"
)

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
