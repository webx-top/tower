package main

import (
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
	} else if !app.IsRunning() || this.Watcher.Changed.Load() {
		this.FirstRequest.Do(func() {
			this.Watcher.Reset()
			err = app.Restart(this.ctx)
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
		log.Infof("== Request: %7s %s => Completed %d in %vs", r.logEntry.Method, r.logEntry.Path, r.logEntry.StatusCode, r.logEntry.TotalDuration.Seconds())
	}
	return nil
}
