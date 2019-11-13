package database

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/FactomWyomingEntity/prosper-pool/config"
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

type PaginationParams struct {
	Limit  int32  `json:"limit,omitempty"`
	Offset int32  `json:"offset,omitempty"`
	Order  string `json:"order,omitempty"`
	// OrderColumn needs sqlinjection protection
	// TODO: Verify the regex is enough...
	OrderColumn string `json:"column,omitempty"`
}

// Set defaults if not already set
func (p *PaginationParams) Default(limit int32, order, orderColumn string) *PaginationParams {
	if p.Limit == 0 {
		p.Limit = limit
	}
	if p.OrderColumn == "" {
		p.OrderColumn = orderColumn
	}
	if p.Order == "" {
		p.Order = order
	}
	return p
}

func (p *PaginationParams) Max(maxLimit int32) *PaginationParams {
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
	return p
}

type PaginationResponse struct {
	Records      int `json:"records"`
	TotalRecords int `json:"totalrecords"`
}

var columnRegex, _ = regexp.Compile("^[a-zA-Z_]+$")

// SimplePagination is the very basic pagination with no search terms.
func SimplePagination(tx *gorm.DB, params PaginationParams) (*gorm.DB, error) {
	if !(params.Order == "" || strings.ToUpper(params.Order) == "ASC" || strings.ToUpper(params.Order) == "DESC") {
		return nil, fmt.Errorf("order must be 'asc' or 'desc'")
	}

	if params.Limit != 0 {
		tx = tx.Limit(params.Limit)
	}
	if params.Offset != 0 {
		tx = tx.Offset(params.Offset)
	}
	if params.Order != "" {
		if params.OrderColumn == "" {
			return nil, fmt.Errorf("if providing an order, an order column must also be provided")
		}
		if !columnRegex.MatchString(params.OrderColumn) {
			return nil, fmt.Errorf("bad order column")
		}
		tx = tx.Order(fmt.Sprintf("%s %s", params.OrderColumn, params.Order))
	}

	return tx, nil
}

func TotalCount(tx *gorm.DB) int {
	var totalCount int
	err := tx.Count(&totalCount).Error
	if err != nil {
		return -1
	}
	return totalCount
}
