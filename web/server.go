package web

import (
	"fmt"
	"net/http"

	"github.com/FactomWyomingEntity/prosper-pool/minutekeeper"

	"github.com/FactomWyomingEntity/prosper-pool/stratum"

	"github.com/jinzhu/gorm"

	"github.com/FactomWyomingEntity/prosper-pool/authentication"
	"github.com/FactomWyomingEntity/prosper-pool/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	wLog = log.WithFields(log.Fields{"mod": "web"})
)

type HttpServices struct {
	Auth          *authentication.Authenticator
	StratumServer *stratum.Server
	MinuteKeeper  *minutekeeper.MinuteKeeper
	Primary       *http.Server
	conf          *viper.Viper
	db            *gorm.DB
}

func NewHttpServices(conf *viper.Viper, db *gorm.DB) *HttpServices {
	s := new(HttpServices)
	s.conf = conf
	s.db = db
	return s
}

func (s *HttpServices) SetStratumServer(srv *stratum.Server) {
	s.StratumServer = srv
}

func (s *HttpServices) SetMinuteKeeper(mk *minutekeeper.MinuteKeeper) {
	s.MinuteKeeper = mk
}

// MiddleWare acts as a middleware for all requests to the web/api
func (s *HttpServices) MiddleWare() func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/auth/register" {
				_, _ = w.Write([]byte("You cannot register"))
				return
			}
			h.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}

	return f
}

func (s *HttpServices) InitPrimary(auth *authentication.Authenticator) {
	primaryMux := http.NewServeMux()
	s.Auth = auth

	primaryMux.HandleFunc("/", s.Index)

	// Init a basic "whoami"
	primaryMux.HandleFunc("/whoami", s.WhoAmI)
	primaryMux.HandleFunc("/user/owed", s.OwedPayouts)
	primaryMux.HandleFunc("/pool/rewards", s.PoolRewards)
	primaryMux.HandleFunc("/pool/submissions", s.PoolSubmissions)
	// primaryMux.HandleFunc("/api/v1/submitsync", s.MinuteKeeperInfo)

	// Links
	primaryMux.HandleFunc("/pool", s.PoolLinks)
	primaryMux.HandleFunc("/users", s.UserLinks)

	// Admin only
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/admin/links", s.AdminLinks)
	adminMux.HandleFunc("/admin/miners", s.PoolMiners)
	primaryMux.Handle("/admin/", s.Auth.Authority.Authorize("admin")(adminMux))

	// Add /auth to primary mux
	auth.AddHandler(primaryMux)

	apiBase := "/api/v1"
	primaryMux.Handle(apiBase, s.APIMux(apiBase))

	s.Primary = &http.Server{
		Handler: s.MiddleWare()(auth.GetSessionManager(primaryMux)),
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
