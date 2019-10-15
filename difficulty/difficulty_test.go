package difficulty_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/FactomWyomingEntity/private-pool/difficulty"
)

const (
	K uint64 = 1000

	Minute uint64 = 60
)

func TestExpectedMinimumTarget(t *testing.T) {
	fmt.Printf("%x\n", Target(100*K))
	fmt.Printf("%x\n", ExpectedTarget(100*K, time.Minute/30))

	//fmt.Println(BlocksPerTime(100*K, time.Minute*9))

	fmt.Printf("%x\n", ExpectedMinimumTarget(100*K*MiningPeriod, 1))
	fmt.Printf("%x\n", ExpectedMinimumTarget(5*K*2*Minute, 1))
	fmt.Printf("%x\n", ExpectedMinimumTarget(5*K*Minute, 1))
	fmt.Printf("%x\n", ExpectedMinimumTarget(5*K*MiningPeriod, 1))
	fmt.Printf("%x\n", ExpectedMinimumTarget(100*K*MiningPeriod, 1))
}
