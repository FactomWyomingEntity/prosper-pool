package config

import (
	"time"

	"github.com/spf13/viper"
)

// All config locations
const (
	LoggingLevel = "app.loglevel"

	ConfigSQLHost     = "database.host"
	ConfigSQLPort     = "database.port"
	ConfigSQLDBName   = "database.dbname"
	ConfigSQLUsername = "database.username"
	ConfigSQLPassword = "database.password"

	ConfigFactomdLocation = "factom.factomdlocation"

	ConfigPegnetPollingPeriod = "pegnet.pollingperiod"
	ConfigPegnetRetryPeriod   = "pegnet.retryperiod"

	Config1ForgeKey = "oracle.1ForgeKey"
)

func SetDefaults(conf *viper.Viper) {
	// All config defaults
	conf.SetDefault(ConfigSQLHost, "localhost")
	conf.SetDefault(ConfigSQLPort, 5432)
	conf.SetDefault(ConfigSQLDBName, "postgres")
	conf.SetDefault(ConfigSQLUsername, "postgres")
	conf.SetDefault(ConfigSQLPassword, "password")

	conf.SetDefault(ConfigFactomdLocation, "http://localhost:8088/v2")

	viper.SetDefault(ConfigPegnetPollingPeriod, time.Second*2)
	viper.SetDefault(ConfigPegnetRetryPeriod, time.Second*5)
}
