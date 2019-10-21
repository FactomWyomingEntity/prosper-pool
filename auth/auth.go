package auth

import (
	"github.com/jinzhu/gorm"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth_themes/clean"
)

// Authenticator is just a wrapper so we can add/extend functionality if we
// need too.
type Authenticator struct {
	*auth.Auth
}

type User struct {
	gorm.Model
	UID  string `gorm:"column:uid"`
	Role string
}

type HotfixedAuthIdentity auth_identity.AuthIdentity

func (HotfixedAuthIdentity) TableName() string { return "basics" }

func NewAuthenticator(db *gorm.DB) *Authenticator {
	a := new(Authenticator)
	a.Auth = clean.New(&auth.Config{
		DB:                db,
		AuthIdentityModel: &HotfixedAuthIdentity{},
		UserModel:         User{},
	})

	db.AutoMigrate(&HotfixedAuthIdentity{})
	db.AutoMigrate(&User{})

	// Register Auth providers
	// Allow use username/password
	//a.RegisterProvider(password.New(&password.Config{}))

	return a
}

func (a Authenticator) Exists(uid string) bool {
	dbErr := a.DB.Where("uid = ?", uid).First(&HotfixedAuthIdentity{})
	if dbErr.Error == nil {
		return true
	}
	return false
}
