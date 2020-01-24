package accounting

import (
	"fmt"
	"sort"
	"time"

	"github.com/FactomWyomingEntity/prosper-pool/difficulty"
	"github.com/shopspring/decimal"
)

type OwedPayouts struct {
	Reward // All the reward info

	// PoolFeeRate is the pool cut
	PoolFeeRate decimal.Decimal `sql:"type:decimal(20,8);" json:"poolfeerate"`
	PoolFee     int64           `json:"poolfee"` // In PEG
	// Dust should always be 0, but it is any rewards that are not accounted
	// to a user or to the pool. We should account for it if it happens.
	Dust int64 `json:"dust"`

	PoolDifficuty float64 `json:"pooldifficulty"`
	PDiff         string  `gorm:"default:'ffff000000000000'" json:"pdiff"` // String to avoid sql uint64 errors
	TotalHashrate float64 `gorm:"default:0" json:"totalhashrate"`

	UserPayouts []UserOwedPayouts `gorm:"foreignkey:JobID" json:"userpayouts,omitempty"`
}

func NewPayout(r Reward, poolFeeRate decimal.Decimal, work ShareMap) *OwedPayouts {
	p := new(OwedPayouts)
	p.PoolFeeRate = poolFeeRate
	p.Reward = r
	p.PDiff = fmt.Sprintf("%x", difficulty.PDiff)
	remaining := p.TakePoolCut(p.Reward.PoolReward)
	p.Payouts(work, remaining)

	return p
}

func (p *OwedPayouts) Payouts(work ShareMap, remaining int64) {
	p.PoolDifficuty = work.TotalDiff
	var totalPayout int64
	for user, work := range work.Sums {
		prop := decimal.NewFromFloat(work.TotalDifficulty).Div(decimal.NewFromFloat(p.PoolDifficuty))
		prop = prop.Truncate(AccountingPrecision)

		// Last hashrate is the best guess
		hashrate := work.LastHashrate()
		if work.TotalShares < 5 {
			// If there is too few shares, don't bother trying to calc a hashrate
			hashrate = 0
		}

		pay := UserOwedPayouts{
			UserID:           user,
			UserDifficuty:    work.TotalDifficulty,
			TotalSubmissions: work.TotalShares,
			Proportion:       prop,
			Payout:           cut(remaining, prop),
			HashRate:         hashrate,
		}
		p.UserPayouts = append(p.UserPayouts, pay)
		totalPayout += pay.Payout

		// Only if a miner mines for at least 20s
		if work.LastShare.Sub(work.FirstShare) > time.Second*20 {
			p.TotalHashrate += pay.HashRate
		}
	}
	p.Dust = remaining - totalPayout
}

// TakePoolCut will take the amount owed the pool, and return the
// remaining rewards to be distributed
func (p *OwedPayouts) TakePoolCut(remaining int64) int64 {
	if p.PoolFeeRate.IsZero() {
		return remaining
	}

	p.PoolFee = cut(remaining, p.PoolFeeRate)
	return remaining - p.PoolFee
}

// cut returns the proportional amount in the total
func cut(total int64, prop decimal.Decimal) int64 {
	amt := decimal.New(total, 0)
	cut := amt.Mul(prop)
	return cut.IntPart()
}

type UserOwedPayouts struct {
	JobID            int32  `gorm:"primary_key" json:"jobid"`
	UserID           string `gorm:"primary_key"`
	UserDifficuty    float64
	TotalSubmissions int

	// Proportion denoted with 10000 being 100% and 1 being 0.01%
	Proportion decimal.Decimal `sql:"type:decimal(20,8);"`
	Payout     int64           // In PEG

	HashRate float64 `gorm:"default:0"` // Hashrate in h/s
}

type Reward struct {
	JobID      int32 `gorm:"primary_key" json:"jobid"` // Block height of reward payout
	PoolReward int64 `json:"poolreward"`               // PEG reward for block

	Winning int `json:"winningoprs"` // Number of oprs in the winning set
	Graded  int `json:"gradedoprs"`  // Number of oprs in the graded set
}

// Share is an accepted piece of work done by a miner.
type Share struct {
	JobID      int32  // JobID's are always a block height
	Nonce      []byte // Nonce is the work computed by the miner
	Difficulty float64
	Target     uint64
	Accepted   bool // Shares can be rejected

	// MinerID is the unique ID of the miner that submit the share
	MinerID string
	// All minerID's should be linked to a user via a userid. The userid is who earns the payouts
	UserID string
}

type ShareMap struct {
	// Sealed means no new shares are accepted and we can garbage collect
	Sealed bool

	TotalDiff float64
	Sums      map[string]*ShareSum
}

func NewShareMap() *ShareMap {
	s := new(ShareMap)
	s.Sums = make(map[string]*ShareSum)
	return s
}

func (m *ShareMap) Seal() {
	m.Sealed = true
}

func (m *ShareMap) AddShare(key string, s Share) {
	if m.Sealed {
		return // Do nothing, it's already sealed
	}

	m.TotalDiff += s.Difficulty
	if _, ok := m.Sums[key]; !ok {
		m.Sums[key] = new(ShareSum)
	}
	m.Sums[key].AddShare(s)
}

const (
	TargetsKept = 30
)

// ShareSum is the sum of shares for a given job
type ShareSum struct {
	TotalDifficulty float64
	TotalShares     int

	FirstShare time.Time
	LastShare  time.Time
	Targets    [TargetsKept]uint64
}

func (sum *ShareSum) AddShare(s Share) {
	if sum.FirstShare.IsZero() {
		sum.FirstShare = time.Now()
	}
	sum.LastShare = time.Now()

	sum.TotalDifficulty += s.Difficulty
	sum.TotalShares++
	InsertTarget(s.Target, &sum.Targets, sum.TotalShares)
}

func (sum ShareSum) LastHashrate() float64 {
	// Estimate user hashrate
	last := TargetsKept
	if sum.TotalShares < TargetsKept {
		last = sum.TotalShares
	}

	target := sum.Targets[last-1]
	hashrate := difficulty.EffectiveHashRate(target, last, sum.LastShare.Sub(sum.FirstShare).Seconds())
	return hashrate
}

// WeightedAverageHashrate weighs hashes according to their place
func (sum ShareSum) WeightedAverageHashrate() float64 {
	last := TargetsKept
	if sum.TotalShares < TargetsKept {
		last = sum.TotalShares
	}

	hashrate := float64(0)
	totalFactor := float64(0)
	for j := 0; j < last; j++ {
		totalFactor += float64(j) + 1
		hashrate += (float64(j) + 1) * difficulty.EffectiveHashRate(sum.Targets[j], j+1, sum.LastShare.Sub(sum.FirstShare).Seconds())
	}
	return hashrate / float64(totalFactor)
}

func (sum ShareSum) AverageHashrate() float64 {
	last := TargetsKept
	if sum.TotalShares < TargetsKept {
		last = sum.TotalShares
	}

	hashrate := float64(0)
	for j := 0; j < last; j++ {
		hashrate += difficulty.EffectiveHashRate(sum.Targets[j], j+1, sum.LastShare.Sub(sum.FirstShare).Seconds())
	}
	return hashrate / float64(last)
}

func InsertTarget(t uint64, a *[TargetsKept]uint64, total int) {
	if total > TargetsKept {
		total = TargetsKept
	}
	index := sort.Search(total, func(i int) bool { return a[i] < t })
	if index == TargetsKept {
		return
	}
	// Move things down
	copy(a[index+1:], a[index:])
	// Insert at index
	a[index] = t
}

func InsertSorted(s []int, e int) []int {
	s = append(s, 0)
	i := sort.Search(len(s), func(i int) bool { return s[i] > e })
	copy(s[i+1:], s[i:])
	s[i] = e
	return s
}

// some utils
func TruncateTo4(v float64) float64 {
	return float64(int64(v*1e4)) / 1e4
}
