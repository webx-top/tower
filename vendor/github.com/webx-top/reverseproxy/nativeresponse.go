package reverseproxy

import (
	"io"
	"net/http"
)

var _ Context = &NativeResponse{}

type NativeResponse struct {
	RespWriter http.ResponseWriter
	*http.Request
	isDead bool
}

func (n *NativeResponse) SetBody(body []byte) {
	n.RespWriter.Write(body)
}

func (n *NativeResponse) SetStatusCode(code int) {
	n.RespWriter.WriteHeader(code)
}

func (n *NativeResponse) Redirect(url string, code int) {
	http.Redirect(n.RespWriter, n.Request, url, code)
}

func (n *NativeResponse) SetHeader(key string, value string) {
	n.RespWriter.Header().Set(key, value)
}

func (n *NativeResponse) GetHeader(key string) string {
	return n.RespWriter.Header().Get(key)
}

func (n *NativeResponse) RequestURI() string {
	return n.Request.URL.RequestURI()
}

func (n *NativeResponse) RequestPath() string {
	return n.Request.URL.Path
}

func (n *NativeResponse) RequestMethod() string {
	return n.Request.Method
}

func (n *NativeResponse) RemoteAddr() string {
	return n.Request.RemoteAddr
}

func (n *NativeResponse) QueryValue(key string) string {
	return n.Request.URL.Query().Get(key)
}

func (n *NativeResponse) QueryValues(key string) []string {
	q := n.Request.URL.Query()
	if v, ok := q[key]; ok {
		return v
	}
	return []string{}
}

func (n *NativeResponse) ResponseWriter() io.Writer {
	return n.RespWriter
}

func (n *NativeResponse) RequestHost() string {
	return n.Request.Host
}

func (n *NativeResponse) IsDead() bool {
	return n.isDead
}

func (n *NativeResponse) SetDead(on bool) {
	n.isDead = on
}
