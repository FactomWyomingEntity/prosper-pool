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

	// TODO: Enable ssl
	dbUri := fmt.Sprintf("host=%s user=%s dbname=%s password=%s port=%d sslmode=disable",
		viper.GetString(config.ConfigSQLHost),
		viper.GetString(config.ConfigSQLUsername),
		viper.GetString(config.ConfigSQLDBName),
		viper.GetString(config.ConfigSQLPassword),
		viper.GetInt(config.ConfigSQLPort))

	db, err := gorm.Open("postgres", dbUri)
	if err != nil {
		return nil, err
	}

	s.DB = db
	s.FullAutoMigrate()
	return s, nil
}

func (s SqlDatabase) FullAutoMigrate() {
	s.DB.AutoMigrate(&PegnetGrade{})
	s.DB.AutoMigrate(&PegnetPayout{})
	s.DB.AutoMigrate(&BlockSync{})

	// Add unique constraint for height and position for payouts
	s.Model(&PegnetPayout{}).AddUniqueIndex("uidx_payouts", "height", "position")

}
