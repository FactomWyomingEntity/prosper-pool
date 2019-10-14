package database

import (
	"fmt"

	"github.com/FactomWyomingEntity/private-pool/config"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/spf13/viper"
)

type SqlDatabase struct {
	*gorm.DB
}

func New(conf *viper.Viper) (*SqlDatabase, error) {
	s := new(SqlDatabase)

	dbUri := fmt.Sprintf("host=%s user=%s dbname=%s password=%s",
		viper.GetString(config.ConfigSQLHost),
		viper.GetString(config.ConfigSQLUsername),
		viper.GetString(config.ConfigSQLDBName),
		viper.GetString(config.ConfigSQLPassword))

	db, err := gorm.Open("postgres", dbUri)
	if err != nil {
		return nil, err
	}

	s.DB = db
	return s, nil
}

func (s SqlDatabase) AutoMigrate() {

}
