package pegnet

import (
	"github.com/jinzhu/gorm"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/FactomWyomingEntity/prosper-pool/config"
	"github.com/FactomWyomingEntity/prosper-pool/database"
	"github.com/FactomWyomingEntity/prosper-pool/factomclient"
	"github.com/pegnet/pegnet/modules/grader"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	pegdLog = log.WithField("mod", "pegd")
)

type Node struct {
	FactomClient *factom.Client
	config       *viper.Viper

	db   *database.SqlDatabase
	Sync *database.BlockSync

	hooks []chan<- PegnetdHook

	// Indicate a fresh boot
	justBooted bool
}

func NewPegnetNode(conf *viper.Viper, db *database.SqlDatabase) (*Node, error) {
	n := new(Node)
	n.FactomClient = factomclient.FactomClientFromConfig(conf)
	n.config = conf
	n.db = db

	if sync, err := n.SelectSynced(); err != nil {
		if err == gorm.ErrRecordNotFound {
			n.Sync = new(database.BlockSync)
			n.Sync.Synced = int32(config.PegnetActivation)
			pegdLog.Debug("connected to a fresh database")
		} else {
			return nil, err
		}
	} else {
		n.Sync = sync
	}

	grader.InitLX()
	return n, nil
}

// PegnetdHook contains all the info (aside from assets) needed to make
// and opr for mining
type PegnetdHook struct {
	Height int32
	// Top means the block is the latest block
	Top         bool
	GradedBlock grader.GradedBlock
}

func (n *Node) GetHook() <-chan PegnetdHook {
	hook := make(chan PegnetdHook, 10)
	n.AddHook(hook)
	return hook
}

// AddHook does not need to be thread safe, as it is called before
// the node is running
func (n *Node) AddHook(hook chan<- PegnetdHook) {
	n.hooks = append(n.hooks, hook)
}

func (n Node) SelectSynced() (*database.BlockSync, error) {
	var s database.BlockSync
	// TODO: Ensure this is max() equivalent
	dbErr := n.db.Order("synced desc").First(&s)
	return &s, dbErr.Error
}
