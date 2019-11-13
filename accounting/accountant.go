package accounting

import (
	"context"
	"fmt"
	"sync"

	"github.com/FactomWyomingEntity/prosper-pool/config"
	"github.com/FactomWyomingEntity/prosper-pool/difficulty"
	"github.com/FactomWyomingEntity/prosper-pool/stratum"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	acctLog = log.WithField("mod", "acct")
)

const AccountingPrecision = 8

type Accountant struct {
	DB *gorm.DB

	// Jobs are indexed by job id
	jobLock     sync.RWMutex
	JobsByMiner map[int32]*ShareMap
	JobsByUser  map[int32]*ShareMap

	newJobs     chan int32
	rewards     chan *Reward
	submissions <-chan *stratum.ShareSubmission

	// shares is mainly used for debug/testing. Most submissions come from
	// Stratum.
	shares chan *Share

	// Pool Configuration
	PoolFeeRate decimal.Decimal
}

func NewAccountant(conf *viper.Viper, db *gorm.DB) (*Accountant, error) {
	a := new(Accountant)
	a.DB = db
	a.shares = make(chan *Share, 100)
	a.rewards = make(chan *Reward, 1000)
	a.newJobs = make(chan int32, 100)
	a.JobsByMiner = make(map[int32]*ShareMap)
	a.JobsByUser = make(map[int32]*ShareMap)

	a.DB.AutoMigrate(&UserOwedPayouts{})
	a.DB.AutoMigrate(&OwedPayouts{})
	a.DB.AutoMigrate(&Paid{})

	cut := conf.GetString(config.ConfigPoolCut)

	if cut == "0" || cut == "" {
		return nil, fmt.Errorf("you set a pool fee of 0. If this was intentional, set it to -1 to have no fee")
	} else if cut == "-1" {
		a.PoolFeeRate = decimal.New(0, 0)
	} else {
		var err error
		a.PoolFeeRate, err = decimal.NewFromString(cut)
		if err != nil {
			return nil, err
		}
	}

	if a.PoolFeeRate.IntPart() > 1 || a.PoolFeeRate.IntPart() < 0 {
		return nil, fmt.Errorf("pool fee must be between 0 and 1")
	}

	a.PoolFeeRate = a.PoolFeeRate.Truncate(AccountingPrecision)

	return a, nil
}

func (a *Accountant) JobChannel() chan<- int32 {
	return a.newJobs
}

func (a *Accountant) RewardChannel() chan<- *Reward {
	return a.rewards
}

func (a *Accountant) ShareChannel() chan<- *Share {
	return a.shares
}

func (a *Accountant) SetSubmissions(subs <-chan *stratum.ShareSubmission) {
	a.submissions = subs
}

// Listen accepts new shares and shares for handling the payout accounting.
func (a *Accountant) Listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case submit := <-a.submissions:
			// A new share from a miner that we need to account for
			if !a.JobExists(submit.JobID) {
				acctLog.WithFields(log.Fields{
					"job":     submit.JobID,
					"minerid": submit.MinerID,
					"userid":  submit.Username,
				}).Debugf("share submitted, but no job exits")
				continue // Nothing to do if the job does not exist
			}

			share := Share{
				JobID:      submit.JobID,
				Nonce:      submit.Nonce,
				Difficulty: difficulty.DifficultyFromTarget(submit.Target, difficulty.PDiff),
				Target:     submit.Target,
				// The share will be rejected if sealed
				Accepted: true,
				MinerID:  submit.MinerID,
				UserID:   submit.Username,
			}

			a.AddShare(share)
		case share := <-a.shares:
			// A share from somewhere internal (probably a test)
			if !a.JobExists(share.JobID) {
				acctLog.WithFields(log.Fields{
					"job":     share.JobID,
					"minerid": share.MinerID,
					"userid":  share.UserID,
				}).Debugf("share submitted, but no job exits")
				continue // Nothing to do if the job does not exist
			}

			a.AddShare(*share)

		case newJob := <-a.newJobs:
			if a.JobExists(newJob) {
				acctLog.WithFields(log.Fields{
					"job": newJob,
				}).Warnf("newjob, but already exists")
				continue
			}
			a.NewJob(newJob)
		case reward := <-a.rewards:
			rLog := acctLog.WithFields(log.Fields{
				"job": reward.JobID,
				"peg": reward.PoolReward / 1e8,
			})
			// Indication of a block being completed and us earning rewards
			if !a.JobExists(reward.JobID) {
				// TODO: We will still do the accounting so our numbers add up.
				// 		But we should really see if we can do something to
				//		payout our users if this happens. Like if we reboot
				//		the pool, and didn't keep the user's pow. We could
				//		just use the last blocks proportions or something.
				rLog.Warnf("reward for job that does not exist")
				a.JobsByMiner[reward.JobID] = NewShareMap()
				a.JobsByUser[reward.JobID] = NewShareMap()
			}

			a.jobLock.Lock()
			us := a.JobsByUser[reward.JobID]
			ms := a.JobsByMiner[reward.JobID]

			if us.TotalDiff != ms.TotalDiff {
				rLog.Error("miner job sum and user job sum differ")
			}
			us.Seal()
			ms.Seal()

			// Setup the payout struct with all the proportional payouts.
			// This will also calculate the pool cut
			pays := NewPayout(*reward, a.PoolFeeRate, *us)

			dbErr := a.DB.FirstOrCreate(pays)
			if dbErr.Error != nil {
				// TODO: This is pretty bad. This means payments failed.
				// 		We don't want to just panic and kill the pool.
				//		Maybe, we can just write everything to a file,
				//		and try to notify someone?

				// TODO: Write to a file all the details so we can recover the payments
				rLog.WithError(dbErr.Error).Error("failed to write payouts to database")
			}

			rLog.WithFields(log.Fields{"pool-diff": us.TotalDiff}).Infof("pool stats")
			a.jobLock.Unlock()
		}
	}
}

func (a *Accountant) AddShare(share Share) {
	a.jobLock.Lock()
	a.JobsByMiner[share.JobID].AddShare(share.MinerID, share)
	a.JobsByUser[share.JobID].AddShare(share.UserID, share)
	a.jobLock.Unlock()
}

// NewJob adds a new job to the maps
func (a *Accountant) NewJob(jobid int32) {
	a.jobLock.Lock()
	defer a.jobLock.Unlock()
	a.JobsByMiner[jobid] = NewShareMap()
	a.JobsByUser[jobid] = NewShareMap()
}

func (a *Accountant) JobExists(jobid int32) bool {
	a.jobLock.RLock()
	defer a.jobLock.RUnlock()
	_, ok := a.JobsByMiner[jobid]
	return ok
}
