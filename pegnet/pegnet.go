package pegnet

import (
	"github.com/jinzhu/gorm"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/FactomWyomingEntity/private-pool/config"
	"github.com/FactomWyomingEntity/private-pool/database"
	"github.com/pegnet/pegnet/modules/grader"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var OPRChain = *factom.NewBytes32FromString("a642a8674f46696cc47fdb6b65f9c87b2a19c5ea8123b3d2f0c13b6f33a9d5ef")
var TransactionChain = *factom.NewBytes32FromString("cffce0f409ebba4ed236d49d89c70e4bd1f1367d86402a3363366683265a242d")
var PegnetActivation uint32 = 206421
var GradingV2Activation uint32 = 210330

// TransactionConversionActivation indicates when tx/conversions go live on mainnet.
// Target Activation Height is Oct 7, 2019 15 UTC
var TransactionConversionActivation uint32 = 213237

// Estimated to be Oct 14 2019, 15:00:00 UTC
var PEGPricingActivation uint32 = 214287

type Node struct {
	FactomClient *factom.Client
	config       *viper.Viper

	db   *database.SqlDatabase
	Sync *database.BlockSync

	hooks []chan<- grader.GradedBlock
}

func NewPegnetNode(conf *viper.Viper, db *database.SqlDatabase) (*Node, error) {
	n := new(Node)
	n.FactomClient = FactomClientFromConfig(conf)
	n.config = conf
	n.db = db

	if sync, err := n.SelectSynced(); err != nil {
		if err == gorm.ErrRecordNotFound {
			n.Sync = new(database.BlockSync)
			n.Sync.Synced = int32(PegnetActivation)
			log.Debug("connected to a fresh database")
		} else {
			return nil, err
		}
	} else {
		n.Sync = sync
	}

	grader.InitLX()
	return n, nil
}

func (n Node) AddHook(hook chan<- grader.GradedBlock) {
	n.hooks = append(n.hooks, hook)
}

func (n Node) SelectSynced() (*database.BlockSync, error) {
	var s database.BlockSync
	// TODO: Ensure this is max() equivalent
	dbErr := n.db.Order("synced desc").First(&s)
	return &s, dbErr.Error
}

func FactomClientFromConfig(conf *viper.Viper) *factom.Client {
	cl := factom.NewClient()
	cl.FactomdServer = conf.GetString(config.ConfigFactomdLocation)
	// We don't use walletd
	cl.WalletdServer = conf.GetString("http://localhost:8089/v2")

	return cl
}
