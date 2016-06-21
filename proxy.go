package main

import (
	"net/http"
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

func (this *Proxy) authAdmin(r *http.Request) bool {
	query := r.URL.Query()
	pwd := query.Get(`pwd`)
	valid := false
	if pwd != `` || pwd == this.AdminPwd {
		valid = true
	} else {
		clientIP := r.RemoteAddr
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

/*
func (this *Proxy) SetBody(code int, body []byte, w http.ResponseWriter) {
	if code == 0 {
		code = http.StatusOK
	}
	w.WriteHeader(code)
	w.Write(body)
}

func (this *Proxy) Listen() (err error) {
	log.Info("== Listening to http://localhost:" + this.Port)
	this.SetReserveProxy()
	this.FirstRequest = &sync.Once{}

	http.HandleFunc("/tower-proxy/watch/pause", func(w http.ResponseWriter, r *http.Request) {
		status := `done`
		if !this.authAdmin(r) {
			status = `Authentication failed`
		} else {
			this.Watcher.Paused = true
		}
		this.SetBody(0, []byte(status), w)
	})

	http.HandleFunc("/tower-proxy/watch/begin", func(w http.ResponseWriter, r *http.Request) {
		status := `done`
		if !this.authAdmin(r) {
			status = `Authentication failed`
		} else {
			this.Watcher.Paused = false
		}
		this.SetBody(0, []byte(status), w)
	})

	http.HandleFunc("/tower-proxy/watch", func(w http.ResponseWriter, r *http.Request) {
		status := `OK`
		if this.Watcher.Paused {
			status = `Pause`
		}
		this.SetBody(0, []byte(`watch status: `+status), w)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		this.ServeRequest(w, r)
	})
	return http.ListenAndServe(":"+this.Port, nil)
}

func (this *Proxy) SetReserveProxy() {
	log.Info("== Proxy to http://localhost:" + this.App.Port)
	this.appOldPort = this.App.Port
	url, _ := url.ParseRequestURI("http://localhost:" + this.App.Port)
	this.ReserveProxy = httputil.NewSingleHostReverseProxy(url)
}

func (this *Proxy) ServeRequest(w http.ResponseWriter, r *http.Request) {
	mw := ResponseWriterWrapper{ResponseWriter: w}
	if !this.App.DisabledLogRequest {
		this.logStartRequest(r)
		defer this.logEndRequest(&mw, r, time.Now())
	}

	if this.App.SwitchToNewPort {
		log.Info(`== Switch port:`, this.appOldPort, `=>`, this.App.Port)
		this.App.SwitchToNewPort = false
		this.SetReserveProxy()
		this.FirstRequest.Do(func() {
			this.ReserveProxy.ServeHTTP(&mw, r)
			this.upgraded = time.Now().Unix()
			go this.App.Clean()
			this.FirstRequest = &sync.Once{}
		})
	} else if !this.App.IsRunning() || this.Watcher.Changed {
		this.Watcher.Reset()
		err := this.App.Restart()
		if err != nil {
			RenderBuildError(&mw, this.App, err.Error())
			return
		}

		this.FirstRequest.Do(func() {
			this.ReserveProxy.ServeHTTP(&mw, r)
			this.FirstRequest = &sync.Once{}
		})
	}

	this.App.LastError = ""
	if this.upgraded > 0 {
		timeout := time.Now().Unix() - this.upgraded
		if timeout > 3600 {
			this.upgraded = 0
		}
		mw.Header().Set(`X-Server-Upgraded`, fmt.Sprintf("%v", timeout))
	}

	if !mw.Processed {
		this.ReserveProxy.ServeHTTP(&mw, r)
	}

	if len(this.App.LastError) != 0 {
		RenderAppError(&mw, this.App, this.App.LastError)
	} else if this.App.IsQuit() {
		log.Warn("== App quit unexpetedly")
		this.App.Start(false)
		RenderError(&mw, this.App, "App quit unexpetedly.")
	}
}
*/
