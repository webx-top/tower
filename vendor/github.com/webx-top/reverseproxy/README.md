#reverseproxy

Golang写的HTTP和websocket反向代理。

支持net/http和fasthttp。

本项目是从 https://github.com/tsuru/planb 将反向代理功能单独剥离出来的一个独立的包。
```
router := &recoderRouter{dst: ts.URL}
var rp ReverseProxy = &FastReverseProxy{}
addr, err := rp.Initialize(ReverseProxyConfig{Listen: "127.0.0.1:0", Router: router, RequestIDHeader: "X-RID"})
go rp.Listen()
defer rp.Stop()
```