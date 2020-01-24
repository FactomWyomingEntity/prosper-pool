package mining_test

import (
	"bytes"
	"encoding/hex"
	"math"
	"testing"

	. "github.com/FactomWyomingEntity/prosper-pool/mining"
)

// BenchmarkNonceRotate/simple_nonce_increment-8         	200000000	         7.94 ns/op
func BenchmarkNonceRotate(b *testing.B) {
	b.Run("simple Nonce increment", testIncrement)
}

func testIncrement(b *testing.B) {
	ni := NewNonceIncrementer(math.MaxUint32, 1)
	ni.Nonce = append(ni.Nonce, []byte{0, 0, 0, 0, 0, 0}...)
	for i := 0; i < b.N; i++ {
		ni.NextNonce()
	}
}

// Takes 14.95s
// Single byte prefix is 12.74s
func TestNonceIncrementer(t *testing.T) {
	incrs := make([]*NonceIncrementer, 0)
	for i := 0; i < 256; i++ {
		incrs = append(incrs, NewNonceIncrementer(math.MaxUint32, uint32(i)))
	}
	used := make(map[string]bool)

	// convert []byte to int
	c := func(b []byte) int {
		var r int
		for i := 0; i < len(b); i++ {
			r <<= 8
			r += int(b[i])
		}
		return r
	}

	var a int
	for i := 0; i < 0x10000; i++ {
		a = c(incrs[0].Nonce[5:])

		if a != i {
			t.Fatalf("n1 mismatched i. want = %d, got = %d, raw = %s", i, a, hex.EncodeToString(incrs[0].Nonce))
		}

		for _, inc := range incrs {
			if bytes.Compare(incrs[0].Nonce[5:], inc.Nonce[5:]) != 0 {
				t.Fatalf("mismatch at %d. n0 = %s, n%d = %s", i, hex.EncodeToString(incrs[0].Nonce[1:]), inc.Nonce[0], hex.EncodeToString(inc.Nonce[1:]))
			}
		}

		for _, inc := range incrs {
			if used[string(inc.Nonce)] {
				t.Fatalf("nonce id%d %d already seen before", inc.Nonce, i)
			}
			used[string(inc.Nonce)] = true
		}

		for _, inc := range incrs {
			inc.NextNonce()
		}
	}
}

func TestNonceIncrementer_Prefix(t *testing.T) {
	n := NewNonceIncrementer(100, 10)
	if !bytes.Equal(n.Prefix(), []byte{0, 0, 0, 100, 10}) {
		t.Error("Not the right prefix")
	}
}

//
//func TestLXR(t *testing.T) {
//	InitLX()
//	// 4c36d71fa95bffe0 f9060442285e00a2c43d7669d6ed794d4371256d8fbfa9f4a84696a6d8a845c5 4cd818b501017bd4 8 fffedc2acef6b383
//	o, _ := hex.DecodeString("f9060442285e00a2c43d7669d6ed794d4371256d8fbfa9f4a84696a6d8a845c5")
//	n, _ := hex.DecodeString("4cd818b501017bd4")
//	h := LX.Hash(append(o, n...))
//	fmt.Printf(hex.EncodeToString(h))
//}
