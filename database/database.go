package database

import (
	"database/sql"

	"github.com/spf13/viper"
)

type SqlDatabase struct {
	*sql.DB
}

func New(conf *viper.Viper) *SqlDatabase {
	s := new(SqlDatabase)

	return s
}
