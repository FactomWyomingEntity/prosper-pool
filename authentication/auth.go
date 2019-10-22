package authentication

import (
	"net/http"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth_themes/clean"
	"github.com/qor/session/manager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	aLog = log.WithFields(log.Fields{"mod": "auth"})
)

// Authenticator is just a wrapper so we can add/extend functionality if we
// need too.
type Authenticator struct {
	*auth.Auth
	localMux *http.ServeMux
}

type User struct {
	gorm.Model
	UID  string `gorm:"column:uid"`
	Role string
}

type HotfixedAuthIdentity auth_identity.AuthIdentity

func (d *HotfixedAuthIdentity) BeforeCreate() (err error) {
	// Automatically email confirm
	t := time.Now()
	d.ConfirmedAt = &t
	return
}

func (HotfixedAuthIdentity) TableName() string { return "basics" }

func NewAuthenticator(conf *viper.Viper, db *gorm.DB) (*Authenticator, error) {
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

	return a, nil
}

func (a Authenticator) Confirm(userid string) {
	//a.DB.Model(&HotfixedAuthIdentity{}).Where()
}

func (a Authenticator) Exists(uid string) bool {
	dbErr := a.DB.Where("uid = ?", uid).First(&HotfixedAuthIdentity{})
	if dbErr.Error == nil {
		return true
	}
	return false
}

func (a Authenticator) GetSessionManager(mux *http.ServeMux) http.Handler {
	return manager.SessionManager.Middleware(mux)
}

func (a Authenticator) AddHandler(mux *http.ServeMux) {
	// Mount Auth to Router
	mux.Handle("/auth/", a.NewServeMux())
	a.localMux = mux // So we can make requests without http
}
