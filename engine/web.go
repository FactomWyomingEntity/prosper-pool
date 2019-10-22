package engine

import (
	"fmt"
	"net/http"

	"github.com/FactomWyomingEntity/private-pool/authentication"
	"github.com/FactomWyomingEntity/private-pool/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	wLog = log.WithFields(log.Fields{"mod": "web"})
)

type HttpServices struct {
	Primary *http.Server
	conf    *viper.Viper
}

func NewHttpServices(conf *viper.Viper) *HttpServices {
	s := new(HttpServices)
	s.conf = conf
	return s
}

func (s *HttpServices) InitPrimary(auth *authentication.Authenticator) {
	primaryMux := http.NewServeMux()

	// Init a basic "whoami"
	primaryMux.HandleFunc("/whoami", func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetCurrentUser(r)
		if user != nil {
			if uc, ok := user.(*authentication.User); ok {
				_, _ = fmt.Fprintf(w, "Hello, %v", uc.UID)
				return
			}
			_, _ = fmt.Fprintf(w, "Unknown")
			return
		}
		_, _ = fmt.Fprintf(w, "Not logged in")
	})

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
