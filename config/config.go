package config

import (
	"time"

	"github.com/spf13/viper"
)

// All config locations
const (
	LoggingLevel = "app.loglevel"

	ConfigPoolCut = "pool.PoolFeeRate"

	ConfigSQLHost     = "Database.host"
	ConfigSQLPort     = "Database.port"
	ConfigSQLDBName   = "Database.dbname"
	ConfigSQLUsername = "Database.username"
	ConfigSQLPassword = "Database.password"

	ConfigFactomdLocation = "Factom.FactomdLocation"

	ConfigPegnetPollingPeriod = "Pegnet.PollingPeriod"
	ConfigPegnetRetryPeriod   = "Pegnet.RetryPeriod"

	Config1ForgeKey            = "Oracle.1ForgeKey"
	ConfigApiLayerKey          = "Oracle.ApiLayerKey"
	ConfigCoinMarketCapKey     = "Oracle.CoinMarketCapKey"
	ConfigOpenExchangeRatesKey = "Oracle.OpenExchangeRatesKey"

	Config1ForgePriority            = "OracleDataSources.1Forge"
	ConfigAPILayerPriority          = "OracleDataSources.APILayer"
	ConfigCoinCapPriority           = "OracleDataSources.CoinCap"
	ConfigExchangeRatesPriority     = "OracleDataSources.ExchangeRates"
	ConfigKitcoPriority             = "OracleDataSources.Kitco"
	ConfigOpenExchangeRatesPriority = "OracleDataSources.OpenExchangeRates"
	ConfigCoinMarketCapPriority     = "OracleDataSources.CoinMarketCap"
	ConfigFreeForexAPIpPriority     = "OracleDataSources.FreeForexAPI"
	ConfigFixedUSDPriority          = "OracleDataSources.FixedUSD"
	ConfigAlternativeMePriority     = "OracleDataSources.AlternativeMe"
)

func SetDefaults(conf *viper.Viper) {
	// All config defaults
	conf.SetDefault(ConfigSQLHost, "localhost")
	conf.SetDefault(ConfigSQLPort, 5432)
	conf.SetDefault(ConfigSQLDBName, "postgres")
	conf.SetDefault(ConfigSQLUsername, "postgres")
	conf.SetDefault(ConfigSQLPassword, "password")

	conf.SetDefault(ConfigFactomdLocation, "http://localhost:8088/v2")

	conf.SetDefault(ConfigPegnetPollingPeriod, time.Second*2)
	conf.SetDefault(ConfigPegnetRetryPeriod, time.Second*5)

	conf.SetDefault(Config1ForgeKey, "CHANGME")
	conf.SetDefault(ConfigApiLayerKey, "CHANGME")
	conf.SetDefault(ConfigCoinMarketCapKey, "CHANGME")
	conf.SetDefault(ConfigOpenExchangeRatesKey, "CHANGME")

	conf.SetDefault(Config1ForgePriority, -1)
	conf.SetDefault(ConfigAPILayerPriority, -1)
	conf.SetDefault(ConfigCoinCapPriority, -1)
	conf.SetDefault(ConfigExchangeRatesPriority, -1)
	conf.SetDefault(ConfigKitcoPriority, -1)
	conf.SetDefault(ConfigOpenExchangeRatesPriority, -1)
	conf.SetDefault(ConfigCoinMarketCapPriority, -1)
	conf.SetDefault(ConfigFreeForexAPIpPriority, -1)
	conf.SetDefault(ConfigFixedUSDPriority, -1)
	conf.SetDefault(ConfigAlternativeMePriority, -1)

	conf.SetDefault(ConfigPoolCut, "0.05")
}
