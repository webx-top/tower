package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func startPprof(port int) {
	log.Println(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), nil))
}
