package main

import (
	"net/http"

	"github.com/webx-top/reverseproxy"
)

func (this *Proxy) handleWatchRestart(ctx reverseproxy.Context) error {
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
	return nil
}

func (this *Proxy) handleWatchPause(ctx reverseproxy.Context) error {
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
	return nil
}

func (this *Proxy) handleWatchBegin(ctx reverseproxy.Context) error {
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
	return nil
}

func (this *Proxy) handleWatchStatus(ctx reverseproxy.Context) error {
	status := `OK`
	if this.Watcher.Paused {
		status = `Pause`
	}
	ctx.SetStatusCode(200)
	ctx.SetBody([]byte(`Watcher Status: ` + status))
	return nil
}
