package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/FactomWyomingEntity/private-pool/exit"

	"github.com/Factom-Asset-Tokens/factom"

	"github.com/FactomWyomingEntity/private-pool/config"

	"github.com/FactomWyomingEntity/private-pool/database"
	"github.com/FactomWyomingEntity/private-pool/pegnet"
	"github.com/FactomWyomingEntity/private-pool/polling"
	"github.com/FactomWyomingEntity/private-pool/stratum"
	"github.com/pegnet/pegnet/modules/opr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type PoolEngine struct {
	conf *viper.Viper

	StratumServer *stratum.Server
	Database      *database.SqlDatabase
	PegnetNode    *pegnet.Node
	Poller        *polling.DataSources
	Identity      IdentityInformation

	// Engine hooks
	// nodeHook listens for new pegnet blocks
	nodeHook <-chan pegnet.HookStruct
}

// IdentityInformation contains all the info needed to make OPRs
type IdentityInformation struct {
	Identity        string
	CoinbaseAddress string
	ECAddress       factom.EsAddress
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
	pol := polling.NewDataSources(e.conf)

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
		e.Identity.ECAddress = adr
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

	// Add all closes
	exit.GlobalExitHandler.AddExit(e.Database.Close)

	return nil
}

func (e *PoolEngine) link() error {
	e.nodeHook = e.PegnetNode.GetHook()

	return nil
}

func (e *PoolEngine) Run(ctx context.Context) {
	// TODO: Spin off all threads

	go e.StratumServer.Listen(ctx)

	// Start syncing Blocks
	go e.PegnetNode.DBlockSync(ctx)

	// Listen for new jobs
	e.listenBlocks(ctx)
}

func (e *PoolEngine) listenBlocks(ctx context.Context) {
	for {
		select {
		case block := <-e.nodeHook:
			hLog := log.WithFields(log.Fields{"height": block.Height})
			// New block, let's construct the job
			assets, err := e.Poller.PullAllPEGAssets(2)
			if err != nil {
				hLog.WithError(err).Errorf("failed to poll asset pricing")
				continue
			}

			var _ = assets
			// Construct the OPR
			// TODO: Modules should have a constructor for us
			record := opr.V2Content{}
			record.Height = block.Height
			record.ID = e.Identity.Identity
			record.Address = e.Identity.CoinbaseAddress
			for _, winner := range block.GradedBlock.WinnersShortHashes() {
				data, err := hex.DecodeString(winner)
				if err != nil {
					hLog.WithError(err).Errorf("winner hex failed to parse")
					continue
				}
				record.Winners = append(record.Winners, data)
			}
			// Assets need to be set in a specific order
			record.Assets = make([]uint64, len(opr.V2Assets))
			for i, name := range opr.V2Assets {
				asset := assets[name]
				record.Assets[i] = uint64(math.Round(asset.Value * 1e8))
			}

			// Get OPRHash
			data, err := record.Marshal()
			if err != nil {
				hLog.WithError(err).Errorf("failed to get oprhash")
				continue
			}
			oprHash := sha256.Sum256(data)
			oprHashHex := fmt.Sprintf("%x", oprHash[:])
			hLog.WithFields(log.Fields{"oprhash": oprHashHex}).Debugf("new job")
			e.StratumServer.Notify(&stratum.Job{
				JobID:   fmt.Sprintf("%d", block.Height),
				OPRHash: oprHashHex,
				OPR:     record,
			})

		case <-ctx.Done():
			return
		}
	}
}
