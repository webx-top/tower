package realip

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"
)

var defaultConfig = New().SetIgnorePrivateIP(true).StartWatchEnv(context.Background(), time.Minute*5)

func Default() *Config {
	return defaultConfig
}

var (
	defaultTrustedProxies   = []string{"0.0.0.0/0", "::/0"}
	defaultUnsafeTrustedIPs = []net.IP{net.ParseIP("0.0.0.0"), net.ParseIP("::")}
	defaultTrustedCIDRs     = []*net.IPNet{
		{ // 0.0.0.0/0 (IPv4)
			IP:   net.IP{0x0, 0x0, 0x0, 0x0},
			Mask: net.IPMask{0x0, 0x0, 0x0, 0x0},
		},
		{ // ::/0 (IPv6)
			IP:   net.IP{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			Mask: net.IPMask{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
	}
	defaultPrivateCIDRs []*net.IPNet
)

// Should use canonical format of the header key s
// https://golang.org/pkg/net/http/#CanonicalHeaderKey
const (
	// - Default

	HeaderXForwardedFor = "X-Forwarded-For"
	HeaderXRealIP       = "X-Real-Ip"

	// RFC7239 defines a new "Forwarded: " header designed to replace the
	// existing use of X-Forwarded-* headers.
	// e.g. Forwarded: for=192.0.2.60;proto=https;by=203.0.113.43
	HeaderForwarded = "Forwarded"

	// - Cloudflare -

	HeaderCFConnectingIP = "Cf-Connecting-Ip"
	HeaderTrueClientIP   = "True-Client-Ip"
)

func HeaderIsXForwardedFor(headerName string) bool {
	return HeaderXForwardedFor == http.CanonicalHeaderKey(headerName)
}

func HeaderIsXRealIP(headerName string) bool {
	return HeaderXRealIP == http.CanonicalHeaderKey(headerName)
}

func HeaderIsForwarded(headerName string) bool {
	return HeaderForwarded == http.CanonicalHeaderKey(headerName)
}

func HeaderEquals(headerNameA string, headerNameB string) bool {
	return strings.EqualFold(headerNameA, headerNameB)
}

func SetPrivateCIDRs(maxCidrBlocks ...string) {
	defaultPrivateCIDRs = make([]*net.IPNet, len(maxCidrBlocks))
	for i, maxCidrBlock := range maxCidrBlocks {
		cidr, err := ParseCIDR(maxCidrBlock)
		if err != nil {
			panic(err)
		}
		defaultPrivateCIDRs[i] = cidr
	}
}

func init() {
	maxCidrBlocks := []string{
		"127.0.0.1/8",    // localhost
		"10.0.0.0/8",     // 24-bit block
		"172.16.0.0/12",  // 20-bit block
		"192.168.0.0/16", // 16-bit block
		"169.254.0.0/16", // link local address
		"::1/128",        // localhost IPv6
		"fc00::/7",       // unique local address IPv6
		"fe80::/10",      // link local address IPv6
	}
	SetPrivateCIDRs(maxCidrBlocks...)
}
