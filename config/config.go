package config

import (
	"github.com/spf13/viper"
)

// All config locations
const (
	ConfigSQLHost     = "database.host"
	ConfigSQLDBName   = "database.dbname"
	ConfigSQLUsername = "database.username"
	ConfigSQLPassword = "database.password"
)

func SetDefaults(conf *viper.Viper) {
	// All config defaults
	conf.SetDefault(ConfigSQLHost, "localhost:5432")
	conf.SetDefault(ConfigSQLDBName, "public")
	conf.SetDefault(ConfigSQLUsername, "postgres")
	conf.SetDefault(ConfigSQLPassword, "password")
}
