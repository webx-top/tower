package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"net/http"

	"github.com/admpub/log"
	"github.com/webx-top/reverseproxy"
)

const ProxyPort = "8080"

var errAppQuit = errors.New("== App quit unexpetedly")

type Proxy struct {
	App                 *App
	appOldPort          string
	ReserveProxy        reverseproxy.ReverseProxy
	Watcher             *Watcher
	FirstRequest        *sync.Once
	upgraded            int64
	Port                string
	AdminPwd            string
	AdminIPs            []string
	Engine              string
	AutoRestartMaxTimes int
	autoRestartTimes    int
	waiting             *sync.Once
}

func NewProxy(app *App, watcher *Watcher) (proxy Proxy) {
	proxy.App = app
	proxy.Watcher = watcher
	proxy.Port = ProxyPort
	proxy.AdminIPs = []string{`127.0.0.1`, `::1`}
	proxy.AutoRestartMaxTimes = 3
	proxy.waiting = &sync.Once{}
	return
}

func (this *Proxy) authAdmin(ctx reverseproxy.Context) bool {
	pwd := ctx.QueryValue(`pwd`)
	valid := false
	if len(pwd) > 0 && pwd == this.AdminPwd {
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
	if this.App.DisabledVisitPort() || len(this.Port) == 0 {
		<-make(chan int)
		return nil
	}
	this.FirstRequest = &sync.Once{}
	router := &ProxyRouter{Proxy: this}
	router.dst = "http://localhost:" + app.Port
	var engine string
	if strings.ToLower(this.Engine) == `fast` {
		this.ReserveProxy = &reverseproxy.FastReverseProxy{PassingBrowsingURL: true}
		engine = `FastHTTP`
	} else {
		this.ReserveProxy = &reverseproxy.NativeReverseProxy{PassingBrowsingURL: true}
		engine = `Standard`
	}

	config := reverseproxy.ReverseProxyConfig{
		Listen:          `:` + this.Port,
		Router:          router,
		RequestIDHeader: "X-Request-ID",
		ResponseBefore: func(ctx reverseproxy.Context) bool {
			switch ctx.RequestPath() {
			case "/tower-proxy/watch/restart":
				status := `done`
				code := 200
				if !this.authAdmin(ctx) {
					code = http.StatusUnauthorized
					status = `Authentication failed`
				} else {
					err := this.App.Restart()
					if err != nil {
						code = http.StatusInternalServerError
						status = err.Error()
					}
				}
				ctx.SetStatusCode(code)
				ctx.SetBody([]byte(status))
				return true

			case "/tower-proxy/watch/pause":
				status := `done`
				code := 200
				if !this.authAdmin(ctx) {
					code = http.StatusUnauthorized
					status = `Authentication failed`
				} else {
					this.Watcher.Paused = true
				}
				ctx.SetStatusCode(code)
				ctx.SetBody([]byte(status))
				return true

			case "/tower-proxy/watch/begin":
				status := `done`
				code := 200
				if !this.authAdmin(ctx) {
					code = http.StatusUnauthorized
					status = `Authentication failed`
				} else {
					this.Watcher.Paused = false
				}
				ctx.SetStatusCode(code)
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
				err := errAppQuit
				this.waiting.Do(func() {
					if !this.App.IsQuit() {
						this.waiting = &sync.Once{}
						return
					}
					for ; this.autoRestartTimes < this.AutoRestartMaxTimes; this.autoRestartTimes++ {
						this.App.Stop(this.App.Port)
						this.App.Clean()
						var port string
						port, err = getPort()
						if err == nil {
							err = this.App.Start(true, port)
						}
						if err == nil {
							this.autoRestartTimes = 0
							break
						}
						log.Error(err)
					}
					this.waiting = &sync.Once{}
				})
				if err != nil {
					if this.App.buildErr != nil {
						RenderBuildError(ctx, this.App, this.App.buildErr.Error())
						return true
					}
					log.Warn(errAppQuit)
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
			if ctx.IsDead() {
				RenderError(ctx, this.App, "App quit unexpetedly.")
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
	log.Info(`== Server(`+engine+`) Address:`, addr)
	this.ReserveProxy.Listen()
	this.ReserveProxy.Stop()
	return nil
}
