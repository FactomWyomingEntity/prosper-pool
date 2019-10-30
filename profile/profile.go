package profile

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"

	log "github.com/sirupsen/logrus"
)

// StartProfiler runs the go pprof tool
// `go tool pprof http://localhost:6060/debug/pprof/profile`
// https://golang.org/pkg/net/http/pprof/
func StartProfiler(expose bool, port int) {
	pre := "localhost"
	if expose {
		pre = ""
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	addr := fmt.Sprintf("%s:%d", pre, port)
	log.Infof("Profiling on %s", addr)
	runtime.SetBlockProfileRate(100000)
	log.Println(http.ListenAndServe(addr, mux))
}

// TODO: Add prometheus option
/*func launchPrometheus(port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", prometheus.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}*/
