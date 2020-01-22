package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"

	"github.com/pegnet/pegnet/modules/grader"

	"github.com/pegnet/pegnet/modules/factoidaddress"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/FactomWyomingEntity/prosper-pool/accounting"
	"github.com/FactomWyomingEntity/prosper-pool/authentication"
	"github.com/FactomWyomingEntity/prosper-pool/config"
	"github.com/FactomWyomingEntity/prosper-pool/database"
	"github.com/FactomWyomingEntity/prosper-pool/exit"
	"github.com/FactomWyomingEntity/prosper-pool/factomclient"
	"github.com/FactomWyomingEntity/prosper-pool/minutekeeper"
	"github.com/FactomWyomingEntity/prosper-pool/pegnet"
	"github.com/FactomWyomingEntity/prosper-pool/polling"
	"github.com/FactomWyomingEntity/prosper-pool/sharesubmit"
	"github.com/FactomWyomingEntity/prosper-pool/stratum"
	"github.com/FactomWyomingEntity/prosper-pool/web"
	"github.com/pegnet/pegnet/modules/opr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	engLog = log.WithField("mod", "eng")
)

type PoolEngine struct {
	conf *viper.Viper

	StratumServer *stratum.Server
	Database      *database.SqlDatabase
	PegnetNode    *pegnet.Node
	Poller        *polling.DataSources
	Accountant    *accounting.Accountant
	Submitter     *sharesubmit.Submitter
	Authenticator *authentication.Authenticator
	Web           *web.HttpServices
	MinuteKeeper  *minutekeeper.MinuteKeeper

	Identity IdentityInformation

	// Engine hooks
	// nodeHook listens for new pegnet blocks
	nodeHook <-chan pegnet.PegnetdHook
}

// IdentityInformation contains all the info needed to make OPRs
type IdentityInformation struct {
	// Identity and CoinbaseAddress is used for OPR creation
	Identity        string
	CoinbaseAddress string
	// ESAddress can be used for monitoring funds
	ESAddress factom.EsAddress
}

// Sets up all the module connections and serves as an an overview
// with access to all modules.
func Setup(conf *viper.Viper) (*PoolEngine, error) {
	e := new(PoolEngine)
	e.conf = conf

	// Init modules
	err := e.init()
	if err != nil {
		return nil, err
	}

	// Link modules
	err = e.link()
	if err != nil {
		return nil, err
	}

	return e, nil
}

// init calls the 'New' on all the modules to initialize them with their
// configurations
func (e *PoolEngine) init() error {
	log.Infof("Build Version: %s", config.CompiledInVersion)
	log.Infof("Build commit %s", config.CompiledInBuild)

	db, err := database.New(e.conf)
	if err != nil {
		return err
	}

	stratumServer, err := stratum.NewServer(e.conf)
	if err != nil {
		return err
	}

	p, err := pegnet.NewPegnetNode(e.conf, db)
	if err != nil {
		return err
	}

	// This one will fatal log if an error is encountered
	// TODO: We should probably make this like the rest
	pol := polling.NewDataSources(e.conf, true)

	acc, err := accounting.NewAccountant(e.conf, db.DB)
	if err != nil {
		return err
	}

	sub, err := sharesubmit.NewSubmitter(e.conf, db.DB)
	if err != nil {
		return err
	}

	auth, err := authentication.NewAuthenticator(e.conf, db.DB)
	if err != nil {
		return err
	}

	srv := web.NewHttpServices(e.conf, db.DB)

	mk := minutekeeper.NewMinuteKeeper(factomclient.FactomClientFromConfig(e.conf))

	// Load our identity info for oprs
	if id := e.conf.GetString(config.ConfigPoolIdentity); id == "" {
		return fmt.Errorf("opr identity must be set")
	} else {
		e.Identity.Identity = id
	}

	if ec := e.conf.GetString(config.ConfigPoolESAddress); ec == "" {
		return fmt.Errorf("private entry credit address must be set")
	} else {
		adr, err := factom.NewEsAddress(ec)
		if err != nil {
			return fmt.Errorf("config entry credit address failed: %s", err.Error())
		}
		e.Identity.ESAddress = adr
	}

	if fa := e.conf.GetString(config.ConfigPoolCoinbase); fa == "" {
		return fmt.Errorf("public factoid coinbase address must be set")
	} else {
		_, err := factom.NewFAAddress(fa)
		if err != nil {
			return fmt.Errorf("config coinbase address failed: %s", err.Error())
		}
		e.Identity.CoinbaseAddress = fa
	}

	// Set all the fields so we can access them from whoever has the engine
	e.StratumServer = stratumServer
	e.Database = db
	e.PegnetNode = p
	e.Poller = pol
	e.Accountant = acc
	e.Submitter = sub
	e.Authenticator = auth
	e.Web = srv
	e.MinuteKeeper = mk

	// Add all closes
	exit.GlobalExitHandler.AddExit(e.Database.Close)

	return nil
}

func (e *PoolEngine) link() error {
	// NodeHook hooks all pegnet blocks
	e.nodeHook = e.PegnetNode.GetHook()

	// Submissions is all stratum miner submissions
	//	One for accounting
	acctSubmissions := e.StratumServer.GetSubmissionExport()
	e.Accountant.SetSubmissions(acctSubmissions)
	//	One for factom submit
	subSubmissions := e.StratumServer.GetSubmissionExport()
	e.Submitter.SetSubmissions(subSubmissions)

	e.Web.InitPrimary(e.Authenticator)
	e.Web.SetStratumServer(e.StratumServer)
	e.Web.SetMinuteKeeper(e.MinuteKeeper)

	e.StratumServer.SetAuthenticator(e.Authenticator)
	e.StratumServer.SetShareCheck(e.MinuteKeeper)

	return nil
}

func (e *PoolEngine) Run(ctx context.Context) {
	// MinuteKeeper watches for the min 0 to 1 problem
	//	- Used by the submitter and stratum server to reject shares
	go e.MinuteKeeper.Run(ctx)

	// Stratum server listens to new jobs - spits out new shares
	go e.StratumServer.Listen(ctx)

	// Accountant listens to new jobs, new rewards, and new shares
	go e.Accountant.Listen(ctx)

	// Start syncing Blocks - spits out new jobs, new rewards
	go e.PegnetNode.DBlockSync(ctx)

	// Submitter takes new blocks, new shares, and new jobs
	go e.Submitter.Run(ctx)

	// Start api/web
	go e.Web.Listen()

	// Listen for new jobs for forwarding
	e.listenBlocks(ctx)
}

func (e *PoolEngine) listenBlocks(ctx context.Context) {
	for {
		select {
		case hook := <-e.nodeHook:
			job := e.createJob(hook)
			if job == nil {
				// This is a problem. createJob() will log the error
				continue
			}

			// We update the job if it is the latest block
			// The hook.Top let's processes know if this is the latest block
			// or we are just syncing
			if hook.Top {
				// Update current job and notify the Miners
				e.StratumServer.UpdateCurrentJob(job)
				// Notify Accounting
				//	Notify of the new job
				e.Accountant.JobChannel() <- job.JobID
			}

			// Rewards are always processed, even if they are not new.
			//	Notify of the rewards
			e.Accountant.RewardChannel() <- e.findRewards(hook)

			// Notify Submissions
			//	Submissions needs the new job to know what shares are valid
			e.Submitter.GetBlocksChannel() <- sharesubmit.SubmissionJob{
				Block: hook,
				Job:   job,
			}
		case <-ctx.Done():
			return
		}
	}
}

// findRewards takes the graded block and tallies up the pool's rewards.
func (e *PoolEngine) findRewards(hook pegnet.PegnetdHook) *accounting.Reward {
	r := accounting.Reward{
		JobID: stratum.JobIDFromHeight(hook.Height),
	}

	for _, graded := range hook.GradedBlock.Graded() {
		// Match on either. If someone mines with a new identity, but for us
		// we will take it?
		if graded.OPR.GetID() == e.Identity.Identity ||
			graded.OPR.GetAddress() == e.Identity.CoinbaseAddress {
			r.Graded++
			if graded.Payout() > 0 {
				r.Winning++
				r.PoolReward += graded.Payout()
			}
		}
	}
	return &r
}

// createJob returns the job to send to the stratum miners.
func (e *PoolEngine) createJob(hook pegnet.PegnetdHook) *stratum.Job {
	hLog := engLog.WithFields(log.Fields{"height": hook.Height})
	if !hook.Top {
		hLog.WithFields(log.Fields{"top": hook.Top}).Debug("no new job for height")
		// Don't bother populating the fields
		return &stratum.Job{
			JobID:   stratum.JobIDFromHeight(hook.Height + 1),
			OPRHash: hex.EncodeToString(make([]byte, 32)),
			OPR:     opr.V2Content{},
			OPRv4:   opr.V4Content{},
		}
	}

	assetList := opr.V2Assets
	version := uint8(2)
	if uint32(hook.Height+1) >= config.FreeFloatingPEGPriceActivation {
		version = 3
	}
	if uint32(hook.Height+1) >= config.V4OPRActivation {
		version = 4
		assetList = opr.V4Assets
	}

	// New block, let's construct the job
	assets, err := e.Poller.PullAllPEGAssets(version)
	if err != nil {
		hLog.WithError(err).Errorf("failed to poll asset pricing")
		return nil
	}

	var _ = assets
	// Construct the OPR
	// TODO: Modules should have a constructor for us
	record := opr.V2Content{}
	record.Height = hook.Height + 1
	record.ID = e.Identity.Identity
	record.Address = e.Identity.CoinbaseAddress
	if hook.GradedBlock != nil {
		for _, winner := range hook.GradedBlock.WinnersShortHashes() {
			data, err := hex.DecodeString(winner)
			if err != nil {
				hLog.WithError(err).Errorf("winner hex failed to parse")
				return nil
			}
			record.Winners = append(record.Winners, data)
		}
	}

	// Assets need to be set in a specific order
	record.Assets = make([]uint64, len(assetList))
	for i, name := range assetList {
		if name == "PEG" && version == 2 {
			record.Assets[i] = uint64(0) // PEG Price is 0 until activation
			continue
		}
		asset := assets[name]
		record.Assets[i] = uint64(math.Round(asset.Value * 1e8))
	}

	// Get OPRHash
	data, err := record.Marshal()
	if version == 4 {
		// V4 is just a wrapper for v2 with more assets
		v4Record := opr.V4Content{record}
		data, err = v4Record.Marshal()
	}

	if err != nil {
		hLog.WithError(err).Errorf("failed to get oprhash")
		return nil
	}
	oprHash := sha256.Sum256(data)
	oprHashHex := fmt.Sprintf("%x", oprHash[:])
	hLog.WithFields(log.Fields{"oprhash": oprHashHex, "top": hook.Top}).Debugf("new job")

	switch version {
	case 2:
		err = ValidateV2Content(data)
	case 3:
		err = ValidateV3Content(data)
	case 4:
		err = ValidateV4Content(data)
	}
	if err != nil {
		hLog.WithError(err).Errorf("OPR Data is Invalid! All submitted records by the pool will be rejected by PegNet!")
		hLog.Errorf("Please check your config is correct. Like the correct data sources")
	}

	// The job is for the height + 1. The synced block is wrapping up the last
	// job
	return &stratum.Job{
		JobID:   stratum.JobIDFromHeight(hook.Height + 1),
		OPRHash: oprHashHex,
		OPR:     record,
		OPRv4:   opr.V4Content{record},
	}
}

func ValidateV3Content(content []byte) error {
	err := ValidateV2Content(content)
	if err != nil {
		return err
	}

	o, err := opr.ParseV2Content(content)
	if err != nil {
		return grader.NewDecodeError(err.Error())
	}

	// Also check the peg price is non-zero
	for i, val := range o.Assets {
		if val == 0 {
			return grader.NewValidateError(fmt.Sprintf("asset quote must be greater than 0, %s is 0", opr.V2Assets[i]))
		}
	}

	return nil
}

// The module does not validate opr content, only the full entry...
// We need to fix that there.
// TODO: Move this into pegnet modules
func ValidateV2Content(content []byte) error {
	o, err := opr.ParseV2Content(content)
	if err != nil {
		return grader.NewDecodeError(err.Error())
	}

	// verify assets
	if len(o.Assets) != len(opr.V2Assets) {
		return grader.NewValidateError("invalid assets")
	}
	for i, val := range o.Assets {
		if i > 0 && val == 0 {
			return grader.NewValidateError(fmt.Sprintf("asset quote must be greater than 0, %s is 0", opr.V2Assets[i]))
		}
	}

	if len(o.Winners) != 10 && len(o.Winners) != 25 {
		return grader.NewValidateError("must have exactly 10 or 25 previous winning shorthashes")
	}

	if err := factoidaddress.Valid(o.Address); err != nil {
		return grader.NewValidateError(fmt.Sprintf("factoidaddress is invalid : %s", err.Error()))
	}

	if valid, _ := regexp.MatchString("^[a-zA-Z0-9,]+$", o.ID); !valid {
		return grader.NewValidateError("only alphanumeric characters and commas are allowed in the identity")
	}

	return nil
}

// The module does not validate opr content, only the full entry...
// We need to fix that there.
// TODO: Move this into pegnet modules
func ValidateV4Content(content []byte) error {
	o, err := opr.ParseV2Content(content)
	if err != nil {
		return grader.NewDecodeError(err.Error())
	}

	// verify assets
	if len(o.Assets) != len(opr.V4Assets) {
		return grader.NewValidateError("invalid assets")
	}
	for i, val := range o.Assets {
		if val == 0 {
			return grader.NewValidateError(fmt.Sprintf("asset quote must be greater than 0, %s is 0", opr.V2Assets[i]))
		}
	}

	if len(o.Winners) != 25 {
		return grader.NewValidateError("must have exactly 10 or 25 previous winning shorthashes")
	}

	if err := factoidaddress.Valid(o.Address); err != nil {
		return grader.NewValidateError(fmt.Sprintf("factoidaddress is invalid : %s", err.Error()))
	}

	if valid, _ := regexp.MatchString("^[a-zA-Z0-9,]+$", o.ID); !valid {
		return grader.NewValidateError("only alphanumeric characters and commas are allowed in the identity")
	}

	return nil
}
