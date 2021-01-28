// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reverseproxy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/admpub/fasthttp"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/webx-top/echo/engine"
	"github.com/webx-top/reverseproxy/log"
)

type FastReverseProxy struct {
	ReverseProxyConfig
	listener           net.Listener
	server             *fasthttp.Server
	dialFunc           func(addr string) (net.Conn, error)
	mu                 sync.Mutex
	clientMap          map[string]*fasthttp.HostClient
	PassingBrowsingURL bool
}

func dialWithTimeout(t time.Duration) func(string) (net.Conn, error) {
	if t > 0 {
		return func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, t)
		}
	}
	return func(addr string) (net.Conn, error) {
		return fasthttp.Dial(addr)
	}
}

func (rp *FastReverseProxy) Initialize(rpConfig ReverseProxyConfig) error {
	rp.ReverseProxyConfig = rpConfig
	rp.dialFunc = dialWithTimeout(rp.DialTimeout)
	rp.clientMap = make(map[string]*fasthttp.HostClient)
	rp.server = &fasthttp.Server{
		Handler: rp.Handler,
	}
	return nil
}

func (rp *FastReverseProxy) Listen(listener ...net.Listener) error {
	if rp.ReverseProxyConfig.DisabledAloneService {
		return nil
	}
	if len(listener) > 0 {
		rp.listener = listener[0]
	} else if rp.listener == nil {
		var err error
		rp.listener, err = net.Listen("tcp", rp.ReverseProxyConfig.Listen)
		if err != nil {
			return err
		}
	}
	return rp.server.Serve(rp.listener)
}

func (rp *FastReverseProxy) Listener() net.Listener {
	return rp.listener
}

func (rp *FastReverseProxy) Stop() error {
	if rp.ReverseProxyConfig.DisabledAloneService {
		return nil
	}
	return rp.listener.Close()
}

func (rp *FastReverseProxy) HandlerForEcho(resp engine.Response, req engine.Request) {
	rp.Handler(req.Object().(*fasthttp.RequestCtx))
}

func (rp *FastReverseProxy) getClient(addr string, tls bool) *fasthttp.HostClient {
	addr = addMissingPort(addr, tls)
	rp.mu.Lock()
	defer rp.mu.Unlock()
	client, ok := rp.clientMap[addr]
	if ok {
		return client
	}
	client = &fasthttp.HostClient{
		Addr:         addr,
		IsTLS:        tls,
		Dial:         rp.dialFunc,
		ReadTimeout:  rp.RequestTimeout,
		WriteTimeout: rp.RequestTimeout,
	}
	rp.clientMap[addr] = client
	return client
}

func addMissingPort(addr string, isTLS bool) string {
	n := strings.Index(addr, ":")
	if n >= 0 {
		return addr
	}
	port := 80
	if isTLS {
		port = 443
	}
	return fmt.Sprintf("%s:%d", addr, port)
}

func (rp *FastReverseProxy) debugHeaders(rsp *fasthttp.Response, reqData *RequestData, isDebug bool) {
	if !isDebug {
		return
	}
	rsp.Header.Set("X-Debug-Backend-Url", reqData.Backend)
	rsp.Header.Set("X-Debug-Backend-Id", strconv.FormatUint(uint64(reqData.BackendIdx), 10))
	rsp.Header.Set("X-Debug-Frontend-Key", reqData.Host)
}

func (rp *FastReverseProxy) serveWebsocket(dstHost string, reqData *RequestData, ctx *fasthttp.RequestCtx) {
	req := &ctx.Request
	uri := req.URI()
	if !rp.PassingBrowsingURL {
		uri.SetHost(dstHost)
	}
	dstConn, err := rp.dialFunc(dstHost)
	if err != nil {
		log.LogError(reqData.String(), string(uri.Path()), err)
		return
	}
	var clientIP string
	if clientIP, _, err = net.SplitHostPort(ctx.RemoteAddr().String()); err == nil {
		if prior := string(req.Header.Peek("X-Forwarded-For")); len(prior) > 0 {
			clientIP = prior + ", " + clientIP
		}
		req.Header.Set("X-Forwarded-For", clientIP)
	}
	_, err = req.WriteTo(dstConn)
	if err != nil {
		log.LogError(reqData.String(), string(uri.Path()), err)
		return
	}
	ctx.Hijack(func(conn net.Conn) {
		defer dstConn.Close()
		defer conn.Close()
		errc := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			_, err := io.Copy(dst, src)
			errc <- err
		}
		go cp(dstConn, conn)
		go cp(conn, dstConn)
		<-errc
	})
}

func (rp *FastReverseProxy) ridString(req *fasthttp.Request) string {
	return rp.RequestIDHeader + ":" + string(req.Header.Peek(rp.RequestIDHeader))
}

func (rp *FastReverseProxy) Handler(ctx *fasthttp.RequestCtx) {
	req := &ctx.Request
	resp := &ctx.Response
	host := string(req.Header.Host())
	uri := req.URI()
	if host == "__ping__" && len(uri.Path()) == 1 && uri.Path()[0] == byte('/') {
		resp.SetBody(okResponse)
		return
	}
	r := &FastResponse{RequestCtx: ctx}
	if rp.ReverseProxyConfig.ResponseBefore != nil {
		if rp.ReverseProxyConfig.ResponseBefore(r) {
			return
		}
	}
	if rp.RequestIDHeader != "" && len(req.Header.Peek(rp.RequestIDHeader)) == 0 {
		var unparsedID *uuid.UUID
		unparsedID, err := uuid.NewV4()
		if err == nil {
			req.Header.Set(rp.RequestIDHeader, unparsedID.String())
		}
	}
	reqData, err := rp.Router.ChooseBackend(host)
	if err != nil {
		reqData.logError(string(uri.Path()), rp.ridString(req), err)
	}
	dstScheme := ""
	dstHost := ""
	u, err := url.Parse(reqData.Backend)
	if err == nil {
		dstScheme = u.Scheme
		dstHost = u.Host
	} else {
		reqData.logError(string(uri.Path()), rp.ridString(req), err)
	}
	if dstHost == "" {
		dstHost = reqData.Backend
	}
	upgrade := req.Header.Peek("Upgrade")
	if len(upgrade) > 0 && bytes.Equal(bytes.ToLower(upgrade), websocketUpgrade) {
		resp.SkipResponse = true
		rp.serveWebsocket(dstHost, reqData, ctx)
		return
	}
	var backendDuration time.Duration
	logEntry := func() *log.LogEntry {
		proto := "HTTP/1.0"
		if req.Header.IsHTTP11() {
			proto = "HTTP/1.1"
		}
		return &log.LogEntry{
			Now:             time.Now(),
			BackendDuration: backendDuration,
			TotalDuration:   time.Since(reqData.StartTime),
			BackendKey:      reqData.BackendKey,
			RemoteAddr:      ctx.RemoteAddr().String(),
			Method:          string(ctx.Method()),
			Path:            string(uri.Path()),
			Proto:           proto,
			Referer:         string(ctx.Referer()),
			UserAgent:       string(ctx.UserAgent()),
			RequestIDHeader: rp.RequestIDHeader,
			RequestID:       string(req.Header.Peek(rp.RequestIDHeader)),
			StatusCode:      resp.StatusCode(),
			ContentLength:   int64(resp.Header.ContentLength()),
		}
	}
	isDebug := len(req.Header.Peek("X-Debug-Router")) > 0
	req.Header.Del("X-Debug-Router")
	if err != nil || dstHost == "" {
		if err != nil {
			reqData.logError(string(uri.Path()), rp.ridString(req), err)
		}
		var status int
		var body []byte
		switch err {
		case nil, ErrNoRegisteredBackends:
			status = http.StatusBadRequest
			body = noRouteResponseContent
		default:
			status = http.StatusServiceUnavailable
		}
		resp.SetStatusCode(status)
		resp.SetBody(body)
		rp.debugHeaders(resp, reqData, isDebug)
		endErr := rp.Router.EndRequest(reqData, false, logEntry)
		if endErr != nil {
			reqData.logError(string(uri.Path()), rp.ridString(req), endErr)
		}
		return
	}
	hostOnly, _, _ := net.SplitHostPort(dstHost)
	if hostOnly == "" {
		hostOnly = dstHost
	}
	isIP := net.ParseIP(hostOnly) != nil
	if !isIP {
		req.Header.SetBytesV("X-Host", uri.Host())
		req.Header.SetBytesV("X-Forwarded-Host", uri.Host())

		if !rp.PassingBrowsingURL {
			uri.SetHost(hostOnly)
		}
	}
	client := rp.getClient(dstHost, dstScheme == "https")
	t0 := time.Now().UTC()
	err = client.Do(req, resp)
	backendDuration = time.Since(t0)
	markAsDead := false
	if err != nil {
		var isTimeout bool
		if netErr, ok := err.(net.Error); ok {
			markAsDead = !netErr.Temporary()
			isTimeout = netErr.Timeout()
		}
		if isTimeout {
			markAsDead = false
			err = fmt.Errorf("request timed out after %v: %s", time.Since(reqData.StartTime), err)
		} else {
			err = fmt.Errorf("error in backend request: %s", err)
		}
		if markAsDead {
			err = fmt.Errorf("%s *DEAD*", err)
			r.SetDead(true)
		}
		resp.SetStatusCode(http.StatusServiceUnavailable)
		reqData.logError(string(uri.Path()), rp.ridString(req), err)
	}
	rp.debugHeaders(resp, reqData, isDebug)
	endErr := rp.Router.EndRequest(reqData, markAsDead, logEntry)
	if endErr != nil {
		reqData.logError(string(uri.Path()), rp.ridString(req), endErr)
	}
	if rp.ReverseProxyConfig.ResponseAfter != nil {
		if rp.ReverseProxyConfig.ResponseAfter(r) {
			return
		}
	}
}
