package stratum

import (
	"os"
	"strconv"
	"sync"

	lxr "github.com/pegnet/LXRHash"
)

// LX holds an instance of lxrhash
var LX *lxr.LXRHash
var lxInitializer sync.Once

// The init function for LX is expensive. So we should explicitly call the init if we intend
// to use it. Make the init call idempotent
func InitLX() {
	lxInitializer.Do(func() {
		// This code will only be executed ONCE, no matter how often you call it
		if size, err := strconv.Atoi(os.Getenv("LXRBITSIZE")); err == nil && size >= 8 && size <= 30 {
			LX = lxr.Init(lxr.Seed, uint64(size), lxr.HashSize, lxr.Passes)
		} else {
			LX = lxr.Init(lxr.Seed, lxr.MapSizeBits, lxr.HashSize, lxr.Passes)
		}
	})
}

// Validator validates an oprhash + nonce combo
func Validate(oprhash, nonce []byte, target uint64) bool {
	return ComputeTarget(oprhash, nonce) == target
}

func ComputeTarget(oprhash, nonce []byte) (difficulty uint64) {
	no := make([]byte, len(oprhash)+len(nonce))
	i := copy(no, oprhash)
	copy(no[i:], nonce)
	b := LX.Hash(no)

	// The high eight bytes of the hash(hash(entry.Content) + nonce) is the difficulty.
	// Because we don't have a difficulty bar, we can define difficulty as the greatest
	// value, rather than the minimum value.  Our bar is the greatest difficulty found
	// within a 10 minute period.  We compute difficulty as Big Endian.
	return uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
		uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
}
