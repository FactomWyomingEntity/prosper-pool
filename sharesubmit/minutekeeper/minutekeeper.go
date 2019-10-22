package minutekeeper

import (
	"context"
	"time"

	"go.uber.org/atomic"

	"github.com/Factom-Asset-Tokens/factom"
	log "github.com/sirupsen/logrus"
)

var (
	mLog = log.WithField("mod", "minkeep")
)

const (
	PollInterval = time.Second * 2
)

// MinuteKeeper has the job of watching the minutes and deciding if it is
// a good time to submit or not. If we are between minute 0 and minute 1, we
// should not submit.
type MinuteKeeper struct {
	FactomClient *factom.Client

	submit atomic.Bool

	lastNoneZeroHeight int32
	lastZero           time.Time
	syncing            bool
}

func NewMinuteKeeper(cl *factom.Client) *MinuteKeeper {
	k := new(MinuteKeeper)
	k.FactomClient = cl
	k.setSubmit(true)
	k.lastZero = time.Now()

	return k
}

// Run is supposed to detect minute 0-1. This is hard as if we sync
// by dbstates, we only get minute 0
func (k *MinuteKeeper) Run(ctx context.Context) {
	// Start watching minutes
	for {
		var cr CurrentMinute
		err := k.FactomClient.FactomdRequest(ctx, "current-minute", nil, &cr)
		if err != nil {
			// Any error? We use rolling submits, and just eat the 1min problem
			k.setSubmit(true)
			mLog.WithError(err).Error("failed to get minute")
			time.Sleep(PollInterval)
			continue
		}
		// This little logic's goal is to detect min -> min 1.
		//	First: If the minute is non-zero, then we are:
		//		- Syncing & Submitting
		// 	Second: If the minute is 0, and the last non zero block was not the
		//			current, then we are not syncing minutes.
		//		- Not syncing & submitting
		//	Third: If we are syncing and we are on minute 0, then
		//			we are betweem min 0 and 1.
		//		- Syncing and not submitting

		// Record the current syncing block
		if cr.Minute != 0 {
			k.setSubmit(true)
			k.syncing = true
			k.lastNoneZeroHeight = cr.Directoryblockheight
		} else if cr.Minute == 0 && cr.Directoryblockheight != k.lastNoneZeroHeight {
			// New minute 0. If we did not sync the prev block by mins,
			// we are dbstate syncing
			k.syncing = false
			k.setSubmit(true)
		} else if k.syncing && cr.Minute == 0 && cr.Directoryblockheight == cr.Leaderheight-1 {
			// On minute 0
			k.setSubmit(false)
		}

		mLog.WithFields(log.Fields{
			"sub":  k.submit.Load(),
			"min":  cr.Minute,
			"sync": k.syncing,
			"sin":  time.Since(k.lastZero),
			"dht":  cr.Directoryblockheight,
			"lht":  cr.Leaderheight,
			"lnz":  k.lastNoneZeroHeight,
		}).
			Tracef("new min")
		time.Sleep(PollInterval)
	}
}

func (k *MinuteKeeper) setSubmit(b bool) {
	k.submit.Store(b)
}

type CurrentMinute struct {
	Leaderheight            int32 `json:"leaderheight"`
	Directoryblockheight    int32 `json:"directoryblockheight"`
	Minute                  int32 `json:"minute"`
	Currentblockstarttime   int64 `json:"currentblockstarttime"`
	Currentminutestarttime  int64 `json:"currentminutestarttime"`
	Currenttime             int64 `json:"currenttime"`
	Directoryblockinseconds int64 `json:"directoryblockinseconds"`
	Stalldetected           bool  `json:"stalldetected"`
	Faulttimeout            int32 `json:"faulttimeout"`
	Roundtimeout            int32 `json:"roundtimeout"`
}
