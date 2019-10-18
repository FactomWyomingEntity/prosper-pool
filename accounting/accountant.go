package accounting

import (
	"context"
	"fmt"
	"sync"

	"github.com/FactomWyomingEntity/private-pool/config"
	"github.com/spf13/viper"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

type Accountant struct {
	DB *gorm.DB

	// Jobs are indexed by job id
	jobLock     sync.RWMutex
	JobsByMiner map[string]*ShareMap
	JobsByUser  map[string]*ShareMap

	newJobs chan string
	shares  chan *Share
	rewards chan *Reward

	// Pool Configuration
	PoolFeeRate int64 // Denoted with 1 being 0.01%
}

func NewAccountant(conf *viper.Viper, db *gorm.DB) (*Accountant, error) {
	a := new(Accountant)
	a.DB = db
	a.shares = make(chan *Share, 1000)

	cut := conf.GetInt64(config.ConfigPoolCut)
	if cut == 0 {
		return nil, fmt.Errorf("you set a pool fee of 0. If this was intentional, set it to -1 to have no fee")
	}

	if cut > 100*10 {
		return nil, fmt.Errorf("pool fee is set to over 100%%")
	} else if cut == -1 {
		a.PoolFeeRate = 0
	} else {
		a.PoolFeeRate = cut
	}

	return a, nil
}

// Listen accepts new shares and shares for handling the payout accounting.
func (a *Accountant) Listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case share := <-a.shares:
			// A new share from a miner that we need to account for
			if !a.JobExists(share.JobID) {
				log.WithFields(log.Fields{
					"job":     share.JobID,
					"minerid": share.MinerID,
					"userid":  share.UserID,
				}).Debugf("share submitted, but no job exits")
				continue // Nothing to do if the job does not exist
			}

			a.jobLock.Lock()
			a.JobsByMiner[share.JobID].AddShare(share.MinerID, *share)
			a.JobsByUser[share.JobID].AddShare(share.UserID, *share)
			a.jobLock.Unlock()
		case newJob := <-a.newJobs:
			if a.JobExists(newJob) {
				log.WithFields(log.Fields{
					"job": newJob,
				}).Warnf("newjob, but already exists")
				continue
			}
			a.NewJob(newJob)
		case reward := <-a.rewards:
			rLog := log.WithFields(log.Fields{
				"job": reward.JobID,
				"peg": reward.Reward / 1e8,
			})
			// Indication of a block being completed and us earning rewards
			if !a.JobExists(reward.JobID) {
				rLog.Warnf("reward for job that does not exist")
				continue
			}
			a.jobLock.Lock()
			us := a.JobsByUser[reward.JobID]
			ms := a.JobsByMiner[reward.JobID]

			if us.TotalDiff != ms.TotalDiff {
				rLog.Error("miner job sum and user job sum differ")
			}
			us.Seal()
			ms.Seal()

			// Setup the payout struct
			var pays = &Payouts{
				PoolFeeRate:   a.PoolFeeRate,
				PoolDifficuty: us.TotalDiff,
				Reward:        *reward,
			}

			// First take the pool cut
			remaining := reward.Reward
			remaining = pays.TakePoolCut(remaining)

			rLog.WithFields(log.Fields{"pool": us.TotalDiff}).Infof("pool stats")
			a.jobLock.Unlock()
		}
	}
}

// NewJob adds a new job to the maps
func (a *Accountant) NewJob(jobid string) {
	a.jobLock.Lock()
	defer a.jobLock.Unlock()
	a.JobsByMiner[jobid] = NewShareMap()
	a.JobsByUser[jobid] = NewShareMap()
}

func (a Accountant) JobExists(jobid string) bool {
	a.jobLock.RLock()
	defer a.jobLock.RUnlock()
	_, ok := a.JobsByMiner[jobid]
	return ok
}

type BlockAccountant struct {
}
