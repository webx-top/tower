package realip

import (
	"net/http"
)

// FromRequest returns client's real public IP address from http request headers.
func FromRequest(r *http.Request) string {
	return defaultConfig.ClientIP(r.RemoteAddr, r.Header.Get)
}

func XRemoteIP(remoteAddr string) string {
	return defaultConfig.RemoteIP(remoteAddr)
}

func XRealIP(xRealIP, xForwardedFor, remoteAddr string) string {
	// If both empty, return IP from remote address
	if len(xRealIP) == 0 && len(xForwardedFor) == 0 {
		return XRemoteIP(remoteAddr)
	}

	ip, valid := defaultConfig.ValidateIPHeader(xForwardedFor, HeaderXForwardedFor, defaultConfig.IgnorePrivateIP())
	if valid {
		return ip
	}

	// If nothing succeed, return X-Real-IP
	ip, valid = defaultConfig.ValidateIPHeader(xRealIP, HeaderXRealIP, defaultConfig.IgnorePrivateIP())
	if valid {
		return ip
	}
	return XRemoteIP(remoteAddr)
}

// RealIP is depreciated, use FromRequest instead
func RealIP(r *http.Request) string {
	return FromRequest(r)
}
