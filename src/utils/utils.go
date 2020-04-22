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

// Sum calculates the sum of the given uint array
func Sum(array []uint) uint {
	if len(array) == 0 {
		return 0
	}

	var sum uint = 0

	for i := 0; i < len(array); i++ {
		sum += array[i]
	}

	return sum
}
