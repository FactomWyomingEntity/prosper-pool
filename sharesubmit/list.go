package sharesubmit

import (
	"sort"
)

func InsertTarget(t uint64, a []uint64) int {
	index := sort.Search(len(a), func(i int) bool { return a[i] < t })
	if index == len(a) { // End of array, we don't want to grow it
		return -1
	}

	// Move things down
	copy(a[index+1:], a[index:])
	// Insert at index
	a[index] = t
	return index
}
