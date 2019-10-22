package web

import (
	"fmt"
	"net/http"

	"github.com/jinzhu/gorm"

	"github.com/FactomWyomingEntity/private-pool/authentication"
	"github.com/FactomWyomingEntity/private-pool/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	wLog = log.WithFields(log.Fields{"mod": "web"})
)

type HttpServices struct {
	Auth    *authentication.Authenticator
	Primary *http.Server
	conf    *viper.Viper
	db      *gorm.DB
}

func NewHttpServices(conf *viper.Viper, db *gorm.DB) *HttpServices {
	s := new(HttpServices)
	s.conf = conf
	s.db = db
	return s
}

func (s *HttpServices) InitPrimary(auth *authentication.Authenticator) {
	primaryMux := http.NewServeMux()
	s.Auth = auth

	// Init a basic "whoami"
	primaryMux.HandleFunc("/whoami", s.WhoAmI)
	primaryMux.HandleFunc("/user/owed", s.OwedPayouts)
	primaryMux.HandleFunc("/pool/rewards", s.PoolRewards)

	auth.AddHandler(primaryMux)

	s.Primary = &http.Server{
		Handler: auth.GetSessionManager(primaryMux),
		Addr:    fmt.Sprintf("0.0.0.0:%d", s.conf.GetInt(config.ConfigWebPort)),
	}
}

func (s *HttpServices) Listen() {
	wLog.Infof("Serving primary web on %s", s.Primary.Addr)
	go s.Primary.ListenAndServe()
}

func (s *HttpServices) Close() error {
	_ = s.Primary.Close()
	return nil
}
