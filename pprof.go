package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/admpub/log"
)

func startPprof(port int) *http.Server {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Infof("== Debug server URL: http://%s/debug/pprof/", addr)
	server := &http.Server{Addr: addr, Handler: nil}
	go log.Error(server.ListenAndServe())
	return server
}
