package realip

const (
	ProxyCloudflare = `cloudflare`
	ProxyDefault    = `default`
)

var headers = map[string][]string{
	ProxyCloudflare: {HeaderTrueClientIP, HeaderCFConnectingIP, HeaderXForwardedFor, HeaderXRealIP},
	ProxyDefault:    {HeaderForwarded, HeaderXForwardedFor, HeaderXRealIP},
}
