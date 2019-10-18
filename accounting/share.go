package accounting

type Payouts struct {
	Reward // All the reward info

	// PoolFeeRate denoted with 10000 being 100% and 1 being 0.01%
	PoolFeeRate int64
	PoolFee     int64 // In PEG

	PoolDifficuty float64

	UserPayouts []Payout
}

// TakePoolCut will take the amount owed the pool, and return the
// remaining rewards to be distributed
func (p *Payouts) TakePoolCut(remaining int64) int64 {
	if p.PoolFeeRate == 0 {
		return remaining
	}
	// Divide by 100*100 since our fee is in 100*100 (1 == 0.01% == 0.0001)
	p.PoolFee = (remaining * p.PoolFeeRate) / (100 * 100)
	return remaining - p.PoolFee
}

type Payout struct {
	UserID        string
	UserDifficuty float64

	// Proportion denoted with 10000 being 100% and 1 being 0.01%
	Proportion int64
	Payout     int64 // In PEG
}

type Reward struct {
	JobID  string `gorm:"primary_key"` // Block height of reward payout
	Reward int64  // PEG reward for block

	Winning int // Number of oprs in the winning set
	Graded  int // Number of oprs in the graded set
}

// Share is an accepted piece of work done by a miner.
type Share struct {
	JobID      string // JobID's are always a block height
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

// ShareSum is the sum of shares for a given job
type ShareSum struct {
	TotalDifficulty float64
	TotalShares     int
}

func (sum *ShareSum) AddShare(s Share) {
	sum.TotalDifficulty += s.Difficulty
	sum.TotalShares++
}
