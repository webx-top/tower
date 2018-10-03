// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reverseproxy

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/webx-top/echo/engine"
	"github.com/webx-top/reverseproxy/log"
)

var (
	noRouteResponseContent = []byte("no such route")
	okResponse             = []byte("OK")
	websocketUpgrade       = []byte("websocket")

	ErrAllBackendsDead      = errors.New("all backends are dead")
	ErrNoRegisteredBackends = errors.New("no backends registered for host")
	ErrNoBackends           = errors.New("no backends")
)

type Router interface {
	ChooseBackend(host string) (*RequestData, error)
	EndRequest(reqData *RequestData, isDead bool, fn func() *log.LogEntry) error
}

type ReverseProxy interface {
	Initialize(rpConfig ReverseProxyConfig) error
	Listen(...net.Listener) error
	Stop() error
	HandlerForEcho(engine.Response, engine.Request)
}

type ReverseProxyConfig struct {
	Listen               string
	Router               Router
	FlushInterval        time.Duration
	DialTimeout          time.Duration
	RequestTimeout       time.Duration
	ReadTimeout          time.Duration
	ReadHeaderTimeout    time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	RequestIDHeader      string
	ResponseBefore       func(Context) bool
	ResponseAfter        func(Context) bool
	DisabledAloneService bool
}

type RequestData struct {
	BackendLen int
	Backend    string
	BackendIdx int
	BackendKey string
	Host       string
	StartTime  time.Time
}

func (r *RequestData) String() string {
	back := r.Backend
	if back == "" {
		back = "?"
	}
	return r.Host + " -> " + back
}

func (r *RequestData) logError(path string, rid string, err error) {
	log.ErrorLogger.Print("ERROR in ", r.String(), " - ", path, " - ", rid, " - ", err.Error())
}

type Context interface {
	SetBody([]byte)
	SetStatusCode(int)
	Redirect(string, int)
	SetHeader(string, string)
	GetHeader(string) string
	RequestURI() string
	RequestPath() string
	RequestMethod() string
	RemoteAddr() string
	QueryValue(string) string
	QueryValues(string) []string
	ResponseWriter() io.Writer
	RequestHost() string
	IsDead() bool
	SetDead(bool)
}
