package factomclient

import (
	"github.com/Factom-Asset-Tokens/factom"
	"github.com/FactomWyomingEntity/prosper-pool/config"
	"github.com/spf13/viper"
)

// TODO: Add TLS support
func FactomClientFromConfig(conf *viper.Viper) *factom.Client {
	cl := factom.NewClient()
	cl.FactomdServer = conf.GetString(config.ConfigFactomdLocation)
	// We don't use walletd
	cl.WalletdServer = conf.GetString("http://localhost:8089")

	return cl
}
