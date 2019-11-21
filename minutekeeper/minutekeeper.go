package minutekeeper

import (
	"context"
	"time"

	"go.uber.org/atomic"

	"github.com/Factom-Asset-Tokens/factom"
	log "github.com/sirupsen/logrus"
)

const (
	PollInterval = time.Second * 2
)

// MinuteKeeper has the job of watching the minutes and deciding if it is
// a good time to submit or not. If we are between minute 0 and minute 1, we
// should not submit.
type MinuteKeeper struct {
	FactomClient *factom.Client

	submit       atomic.Bool
	submitHeight atomic.Int32

	lastNoneZeroHeight int32
	syncing            bool

	// Has it's own logger to not be noisy
	Logger *log.Logger
	logE   *log.Entry
}

type MinuteKeeperStatus struct {
	Submit             bool  `json:"submitting"`
	SubmitHeight       int32 `json:"submitheight"`
	Syncing            bool  `json:"syncing"`
	LastNoneZeroHeight int32 `json:"lastnonzero"`
}

func NewMinuteKeeper(cl *factom.Client) *MinuteKeeper {
	k := new(MinuteKeeper)
	k.FactomClient = cl
	k.setSubmit(true)
	k.Logger = log.New()
	k.Logger.SetLevel(log.FatalLevel)
	k.logE = k.Logger.WithField("mod", "minkeep")

	return k
}

func (k *MinuteKeeper) Status() MinuteKeeperStatus {
	return MinuteKeeperStatus{
		Submit:             k.submit.Load(),
		SubmitHeight:       k.submitHeight.Load(),
		Syncing:            k.syncing,
		LastNoneZeroHeight: k.lastNoneZeroHeight,
	}
}

func (k *MinuteKeeper) log() *log.Entry {
	return k.logE
}

// Run is supposed to detect minute 0-1. This is hard as if we sync
// by dbstates, we only get minute 0
func (k *MinuteKeeper) Run(ctx context.Context) {
	// Start watching minutes
	for {
		if ctx.Err() != nil {
			return // Cancelled
		}
		var cr CurrentMinute
		err := k.FactomClient.FactomdRequest(nil, "current-minute", nil, &cr)
		if err != nil {
			// Any error? We use rolling submits, and just eat the 1min problem
			k.setSubmit(true)
			k.log().WithError(err).Error("failed to get minute")
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
			k.setSubmitHeight(cr.Directoryblockheight + 1)
			k.syncing = true
			k.lastNoneZeroHeight = cr.Directoryblockheight
		} else if cr.Minute == 0 && cr.Directoryblockheight != k.lastNoneZeroHeight {
			// New minute 0. If we did not sync the prev block by mins,
			// we are dbstate syncing
			k.syncing = false
			k.setSubmit(true)
			k.setSubmitHeight(cr.Directoryblockheight + 1)
		} else if k.syncing && cr.Minute == 0 && cr.Directoryblockheight == cr.Leaderheight-1 {
			// On minute 0
			k.setSubmit(false)
		}

		k.log().WithFields(log.Fields{
			"sub":  k.submit.Load(),
			"min":  cr.Minute,
			"sync": k.syncing,
			"dht":  cr.Directoryblockheight,
			"lht":  cr.Leaderheight,
			"lnz":  k.lastNoneZeroHeight,
		}).Tracef("new min")
		time.Sleep(PollInterval)
	}
}

func (k *MinuteKeeper) setSubmitHeight(h int32) {
	k.submitHeight.Store(h)
}

func (k *MinuteKeeper) setSubmit(b bool) {
	k.submit.Store(b)
}

// CanSubmit will return if we are in a can submit mode. It does not indicate
// if the height you are asking about is the correct height to submit for.
func (k *MinuteKeeper) CanSubmit() bool {
	return k.submit.Load()
}

// CanSubmitHeight will ensure not only are we in a submit mode, but also
// if we are submitting for the latest height.
func (k *MinuteKeeper) CanSubmitHeight(h int32) bool {
	if h != k.submitHeight.Load() {
		return false
	}
	return k.CanSubmit()
}

// CurrentMinute is the factomd api struct
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
