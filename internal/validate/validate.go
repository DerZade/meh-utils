package validate

import (
	"fmt"
	"path"

	"github.com/gruppe-adler/meh-utils/internal/utils"
)

// MehDirectory validates that given directory is valid grad_meh directory
func MehDirectory(mehDirPath string) error {
	if !utils.IsDirectory(mehDirPath) {
		return fmt.Errorf("%s does not exists or is no directory", mehDirPath)
	}

	// check DEM
	if !utils.IsFile(path.Join(mehDirPath, "dem.asc.gz")) {
		return fmt.Errorf("%s is missing", path.Join(mehDirPath, "dem.asc.gz"))
	}

	// check preview.png
	if !utils.IsFile(path.Join(mehDirPath, "preview.png")) {
		return fmt.Errorf("%s is missing", path.Join(mehDirPath, "preview.png"))
	}

	// check meta.json
	if !utils.IsFile(path.Join(mehDirPath, "meta.json")) {
		return fmt.Errorf("%s is missing", path.Join(mehDirPath, "meta.json"))
	}

	// check geojson directory
	if !utils.IsDirectory(path.Join(mehDirPath, "geojson")) {
		return fmt.Errorf("%s is missing", path.Join(mehDirPath, "geojson"))
	}

	return SatDirectory(path.Join(mehDirPath, "sat"))
}

// SatDirectory validates that given directory is valid grad_meh sat directory
func SatDirectory(satDirPath string) error {

	// check if directory exists
	if !utils.IsDirectory(satDirPath) {
		return fmt.Errorf("%s does not exists or is no directory", satDirPath)
	}

	// check if sat tiles exist
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			filePath := path.Join(satDirPath, fmt.Sprintf("%d", col), fmt.Sprintf("%d.png", row))
			if !utils.IsFile(filePath) {
				return fmt.Errorf("%s is missing", filePath)
			}
		}
	}

	return nil
}
