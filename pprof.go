package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/admpub/log"
)

func startPprof(port int) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Infof("== Debug server URL: http://%s/debug/pprof/", addr)
	log.Error(http.ListenAndServe(addr, nil))
}
