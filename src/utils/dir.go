package utils

import (
	"os"
)

// IsFile tests wether given path exists and is a file
func IsFile(filePath string) bool {
	file, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		return false
	}

	return !file.IsDir()
}

// IsDirectory tests wether given path exists and is a directory
func IsDirectory(dirPath string) bool {
	dir, err := os.Stat(dirPath)

	if os.IsNotExist(err) {
		return false
	}

	return dir.IsDir()
}
