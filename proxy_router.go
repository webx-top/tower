package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/admpub/log"
	"github.com/webx-top/reverseproxy"
	rlog "github.com/webx-top/reverseproxy/log"
)

type ProxyRouter struct {
	*Proxy
	dst           string //目标网址
	resultHost    string //最终操作的主机
	resultReqData *reverseproxy.RequestData
	resultIsDead  bool
	logEntry      *rlog.LogEntry
}

func (r *ProxyRouter) ChooseBackend(host string) (*reverseproxy.RequestData, error) {
	this := r.Proxy
	app := this.App
	var err error
	if app.SwitchToNewPort {
		this.FirstRequest.Do(func() {
			log.Info(`== Switch port: `, this.appOldPort, ` => `, app.Port)
			app.SwitchToNewPort = false
			this.upgraded = time.Now().Unix()
			go this.App.Clean()
			r.dst = "http://localhost:" + app.Port
			log.Info("== Listening to " + r.dst)
			this.FirstRequest = &sync.Once{}
		})
	} else if !app.IsRunning() || this.Watcher.Changed {
		this.FirstRequest.Do(func() {
			this.Watcher.Reset()
			err = app.Restart()
			this.FirstRequest = &sync.Once{}
		})
	}

	r.resultHost = host
	return &reverseproxy.RequestData{
		Backend:    r.dst,
		BackendIdx: 0,
		BackendKey: host,
		BackendLen: 1,
		Host:       host,
		StartTime:  time.Now(),
	}, err
}

func (r *ProxyRouter) EndRequest(reqData *reverseproxy.RequestData, isDead bool, fn func() *rlog.LogEntry) error {
	r.resultReqData = reqData
	r.logEntry = fn()
	r.resultIsDead = isDead
	if !r.Proxy.App.DisabledLogRequest {
		log.Infof("[%s]%s => Completed %d in %vs", r.logEntry.Method, r.logEntry.Path, r.logEntry.StatusCode, r.logEntry.TotalDuration.Seconds())
	}
	return nil
}

func (this *Proxy) Listen() error {
	this.FirstRequest = &sync.Once{}
	router := &ProxyRouter{Proxy: this}
	router.dst = "http://localhost:" + app.Port
	this.ReserveProxy = &reverseproxy.FastReverseProxy{}
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

			return false
		},
		ResponseAfter: func(ctx reverseproxy.Context) bool {
			if len(this.App.LastError) != 0 {
				RenderAppError(ctx, this.App, this.App.LastError)
				return true
			}
			if this.App.IsQuit() {
				log.Warn("== App quit unexpetedly")
				this.App.Start(false)
				RenderError(ctx, this.App, "App quit unexpetedly.")
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
