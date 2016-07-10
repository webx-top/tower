package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/admpub/log"
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
	Engine       string
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

func (this *Proxy) Listen() error {
	this.FirstRequest = &sync.Once{}
	router := &ProxyRouter{Proxy: this}
	router.dst = "http://localhost:" + app.Port

	if strings.ToLower(this.Engine) == `fast` {
		this.ReserveProxy = &reverseproxy.FastReverseProxy{PassingBrowsingURL: true}
	} else {
		this.ReserveProxy = &reverseproxy.NativeReverseProxy{PassingBrowsingURL: true}
	}

	config := reverseproxy.ReverseProxyConfig{
		Listen:          `:` + this.Port,
		Router:          router,
		RequestIDHeader: "X-Request-ID",
		ResponseBefore: func(ctx reverseproxy.Context) bool {
			switch ctx.RequestPath() {
			case "/tower-proxy/watch/pause":
				status := `done`
				if !this.authAdmin(ctx) {
					status = `Authentication failed`
				} else {
					this.Watcher.Paused = true
				}
				ctx.SetStatusCode(200)
				ctx.SetBody([]byte(status))
				return true

			case "/tower-proxy/watch/begin":
				status := `done`
				if !this.authAdmin(ctx) {
					status = `Authentication failed`
				} else {
					this.Watcher.Paused = false
				}
				ctx.SetStatusCode(200)
				ctx.SetBody([]byte(status))
				return true

			case "/tower-proxy/watch":
				status := `OK`
				if this.Watcher.Paused {
					status = `Pause`
				}
				ctx.SetStatusCode(200)
				ctx.SetBody([]byte(`watch status: ` + status))
				return true
			}

			this.App.LastError = ""
			if this.upgraded > 0 {
				timeout := time.Now().Unix() - this.upgraded
				if timeout > 3600 {
					this.upgraded = 0
				}
				ctx.SetHeader(`X-Server-Upgraded`, fmt.Sprintf("%v", timeout))
			}
			if this.App.IsQuit() {
				log.Warn("== App quit unexpetedly")
				if err := this.App.Start(false); err != nil {
					RenderError(ctx, this.App, "App quit unexpetedly.")
					return true
				}
			}
			return false
		},
		ResponseAfter: func(ctx reverseproxy.Context) bool {
			if len(this.App.LastError) != 0 {
				RenderAppError(ctx, this.App, this.App.LastError)
				return true
			}
			return false
		},
	}
	this.appOldPort = app.Port
	addr, err := this.ReserveProxy.Initialize(config)
	if err != nil {
		return err
	}
	log.Info("== Listening to " + router.dst)
	log.Info(`== Server Address:`, addr)
	this.ReserveProxy.Listen()
	this.ReserveProxy.Stop()
	return nil
}
