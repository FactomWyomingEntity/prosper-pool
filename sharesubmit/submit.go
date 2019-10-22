package sharesubmit

import (
	"context"
	"math/big"

	"github.com/FactomWyomingEntity/private-pool/config"
	"github.com/FactomWyomingEntity/private-pool/difficulty"
	"github.com/FactomWyomingEntity/private-pool/pegnet"
	"github.com/FactomWyomingEntity/private-pool/stratum"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	sLog = log.WithFields(log.Fields{"mod": "submit"})
)

// Submitter handles submitting shares to factomd. If the share is too old,
// or too low, it will not submit. If we are submitting too many, then it
// will switch from rolling submissions to minute 9 submissions
type Submitter struct {
	db *gorm.DB

	// shares channel is made elsewhere
	shares <-chan *stratum.ShareSubmission
	blocks chan SubmissionJob

	currentJob    *stratum.Job
	currentEMA    EMA
	configuration struct {
		Cutoff       int
		EMANumPoints int
	}
}

// SubmissionJob contains all the info a submitter will need.
// It needs the details for the last block submitted to maintain a
// min target. It needs the job information to submit the incoming shares
type SubmissionJob struct {
	Block pegnet.PegnetdHook
	Job   *stratum.Job
}

func NewSubmitter(conf *viper.Viper, db *gorm.DB) (*Submitter, error) {
	s := new(Submitter)
	s.blocks = make(chan SubmissionJob, 10)
	s.db = db
	s.db.AutoMigrate(&EMA{})

	// Load the latest ema
	dbErr := s.db.Order("block_height desc").First(&s.currentEMA)
	if dbErr.Error != nil && dbErr.Error != gorm.ErrRecordNotFound {
		return nil, dbErr.Error
	}

	s.configuration.Cutoff = conf.GetInt(config.ConfigSubmitterCutoff)
	s.configuration.EMANumPoints = conf.GetInt(config.ConfigSubmitterEMAN)

	return s, nil
}

func (s *Submitter) SetShareChannel(shares <-chan *stratum.ShareSubmission) {
	s.shares = shares
}

func (s Submitter) GetBlocksChannel() chan<- SubmissionJob {
	return s.blocks
}

func (s *Submitter) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case block := <-s.blocks:
			// A new block indicates a new job
			s.currentJob = block.Job

			set := block.Block.GradedBlock.Graded()
			last, lastIndex := uint64(0), 0
			if len(set) > 1 {
				last, lastIndex = set[len(set)-1].SelfReportedDifficulty, len(set)-1
			}
			minTarget := difficulty.CalculateMinimumDifficultyFromOPRs(set, 200)
			ema := EMA{
				BlockHeight:     block.Block.Height,
				JobID:           stratum.JobIDFromHeight(block.Block.Height),
				Cutoff:          s.configuration.Cutoff,
				MinimumTarget:   minTarget,
				EMAValue:        ComputeEMA(minTarget, s.currentEMA.EMAValue, s.configuration.EMANumPoints),
				LastGraded:      last,
				LastGradedIndex: lastIndex,
				N:               s.configuration.EMANumPoints,
			}

			err := s.saveEMA(ema)
			if err != nil {
				sLog.WithError(err).WithField("height", block.Block.Height).Errorf("failed to write ema")
			}

			// This means there is also a new job
			if block.Block.Top {
				lastGradedDifficulty.Set(float64(ema.LastGraded))
				lastGradedIndex.Set(float64(ema.LastGradedIndex))
				cutoffMinimumDifficulty.Set(float64(ema.MinimumTarget))
				cutoffMinimumIndex.Set(200)
				emaDifficulty.Set(float64(ema.EMAValue))
			}

			var _ = ema

		case share := <-s.shares:
			if share.JobID != s.currentJob.JobID {
				continue // Invalid share
			}
		}
	}
}

// saveEMA will save a copy of the EMA to the database. It's a copy because
// uint64s are not always safe to sql
func (s Submitter) saveEMA(ema EMA) error {
	return s.db.Create(&ema).Error
}

// EMA = [Latest Value  - Previous EMA Value] * (2 / N+1) + Previous EMA
// N is the number of points in the Exponential Moving Average
type EMA struct {
	BlockHeight     int32 `gorm:"primary_key"`
	JobID           string
	Cutoff          int    // 200
	MinimumTarget   uint64 // 200 Cutoff target
	EMAValue        uint64 // EMA value
	LastGraded      uint64 // Last graded diff
	LastGradedIndex int
	N               int
}

func ComputeEMA(latest uint64, previous uint64, nPoints int) uint64 {
	if previous == 0 {
		return latest
	}

	l := new(big.Int).SetUint64(latest)
	p := new(big.Int).SetUint64(previous)
	n := big.NewInt(int64(nPoints) + 1)

	s := new(big.Int).Sub(l, p)
	s.Mul(s, big.NewInt(2))
	s.Div(s, n)
	s.Add(s, p)
	return s.Uint64()
}

// BeforeCreate
// uint64's cannot have their highest bit set. The lowest bit doesn't matter
// so we can shift right, then shift left when reading.
func (d *EMA) BeforeCreate() (err error) {
	d.MinimumTarget = d.MinimumTarget >> 1
	d.EMAValue = d.EMAValue >> 1
	d.LastGraded = d.LastGraded >> 1

	return
}

func (d *EMA) AfterFind() (err error) {
	// Add back the top bit
	d.MinimumTarget = d.MinimumTarget << 1
	d.EMAValue = d.EMAValue << 1
	d.LastGraded = d.LastGraded << 1
	return
}
