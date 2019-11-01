package database

import (
	"time"
)

// Base Gorm model with omitemptys
type Model struct {
	ID        uint       `gorm:"primary_key" json:"id,omitempty"`
	CreatedAt *time.Time `json:"created,omitempty"`
	UpdatedAt *time.Time `json:"updated,omitempty"`
	DeletedAt *time.Time `sql:"index" json:"deleted,omitempty"`
}
