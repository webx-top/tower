package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func startPprof(port int) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("== Debug server URL: http://%s/debug/pprof/\n", addr)
	log.Println(http.ListenAndServe(addr, nil))
}
