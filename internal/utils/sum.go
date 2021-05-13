package utils

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
