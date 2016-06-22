package main

import (
	"strings"
	"sync"

	"github.com/webx-top/reverseproxy"
)

const ProxyPort = "8080"

type Proxy struct {
	App          *App
	appOldPort   string
	ReserveProxy reverseproxy.ReverseProxy
	Watcher      *Watcher
	FirstRequest *sync.Once
	upgraded     int64
	Port         string
	AdminPwd     string
	AdminIPs     []string
}

func NewProxy(app *App, watcher *Watcher) (proxy Proxy) {
	proxy.App = app
	proxy.Watcher = watcher
	proxy.Port = ProxyPort
	proxy.AdminIPs = []string{`127.0.0.1`, `::1`}
	return
}

func (this *Proxy) authAdmin(ctx reverseproxy.Context) bool {
	pwd := ctx.QueryValue(`pwd`)
	valid := false
	if pwd != `` || pwd == this.AdminPwd {
		valid = true
	} else {
		clientIP := ctx.RemoteAddr()
		if p := strings.LastIndex(clientIP, `]:`); p > -1 {
			clientIP = clientIP[0:p]
			clientIP = strings.TrimPrefix(clientIP, `[`)
		} else if p := strings.LastIndex(clientIP, `:`); p > -1 {
			clientIP = clientIP[0:p]
		}
		for _, ip := range this.AdminIPs {
			if ip == clientIP {
				valid = true
				break
			}
		}
	}
	return valid
}
