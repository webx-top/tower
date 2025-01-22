package realip

import (
	"context"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const EnvKey = `REALIP_TRUSTED_PROXIES`

func New() *Config {
	c := &Config{}
	c.Init()
	c.SetTrustedProxiesByEnv()
	return c
}

type Config struct {
	proxyType atomic.Value

	// forwardedByClientIP if enabled, client IP will be parsed from the request's headers that
	// match those stored at `Config.RemoteIPHeaders`. If no IP was
	// fetched, it falls back to the IP obtained from
	// `Context.Request().RemoteAddress()`.
	forwardedByClientIP atomic.Bool
	// remoteIPHeaders list of headers used to obtain the client IP when
	// `Config.ForwardedByClientIP` is `true` and
	// `Context.Request().RemoteAddress()` is matched by at least one of the
	// network origins of list defined by `Config.SetTrustedProxies()`.
	remoteIPHeaders []string
	ipHeaderMutex   sync.RWMutex

	ignorePrivateIP atomic.Bool
	trustedMutex    sync.RWMutex
	trustedProxies  []string
	trustedCIDRs    []*net.IPNet
	trustedDefault  bool

	envTrustedProxies string
	envMutex          sync.RWMutex
	envTrustedCIDRs   []*net.IPNet
}

func (c *Config) Init() *Config {
	c.forwardedByClientIP.Store(true)
	c.SetProxyType(ProxyDefault)
	c.ignorePrivateIP.Store(false)
	return c.TrustAll()
}

func (c *Config) SetProxyType(proxyType string) *Config {
	old, y := c.proxyType.Load().(string)
	if y && proxyType == old {
		return c
	}
	c.proxyType.Store(proxyType)
	hdrs, ok := headers[proxyType]
	if ok {
		c.SetRemoteIPHeaders(hdrs...)
	} else if proxyType != ProxyDefault {
		c.SetRemoteIPHeaders(headers[ProxyDefault]...)
	}
	return c
}

func (c *Config) SetIgnorePrivateIP(ignorePrivateIP bool) *Config {
	c.ignorePrivateIP.Store(ignorePrivateIP)
	return c
}

func (c *Config) SetForwardedByClientIP(forwardedByClientIP bool) *Config {
	c.forwardedByClientIP.Store(forwardedByClientIP)
	return c
}

func (c *Config) SetRemoteIPHeaders(remoteIPHeaders ...string) *Config {
	c.ipHeaderMutex.Lock()
	c.remoteIPHeaders = remoteIPHeaders
	c.ipHeaderMutex.Unlock()
	return c
}

func (c *Config) AddRemoteIPHeader(remoteIPHeaders ...string) *Config {
	c.ipHeaderMutex.Lock()
	c.remoteIPHeaders = append(c.remoteIPHeaders, remoteIPHeaders...)
	c.ipHeaderMutex.Unlock()
	return c
}

func (c *Config) IgnorePrivateIP() bool {
	return c.ignorePrivateIP.Load()
}

func PrepareTrustedCIDRs(trustedProxies []string) ([]*net.IPNet, error) {
	if trustedProxies == nil {
		return nil, nil
	}

	cidr := make([]*net.IPNet, 0, len(trustedProxies))
	for _, trustedProxy := range trustedProxies {
		cidrNet, err := ParseCIDR(trustedProxy)
		if err != nil {
			return cidr, err
		}
		cidr = append(cidr, cidrNet)
	}
	return cidr, nil
}

// SetTrustedProxies set a list of network origins (IPv4 addresses,
// IPv4 CIDRs, IPv6 addresses or IPv6 CIDRs) from which to trust
// request's headers that contain alternative client IP when
// `Config.ForwardedByClientIP` is `true`. `TrustedProxies`
// feature is enabled by default, and it also trusts all proxies
// by default. If you want to disable this feature, use
// Config.SetTrustedProxies(nil), then Context.ClientIP() will
// return the remote address directly.
func (c *Config) SetTrustedProxies(trustedProxies []string) error {
	if trustedProxies == nil {
		c.trustedMutex.RLock()
		isDefault := c.trustedDefault
		c.trustedMutex.RUnlock()
		if !isDefault {
			c.TrustAll()
		}
		return nil
	}

	c.trustedMutex.Lock()
	defer c.trustedMutex.Unlock()

	c.trustedProxies = trustedProxies
	trustedCIDRs, err := PrepareTrustedCIDRs(trustedProxies)
	if err != nil {
		return err
	}
	c.trustedCIDRs = trustedCIDRs
	c.trustedDefault = false
	return nil
}

func (c *Config) AddTrustedProxies(trustedProxies ...string) error {
	c.trustedMutex.Lock()
	defer c.trustedMutex.Unlock()

	c.trustedProxies = append(c.trustedProxies, c.trustedProxies...)
	trustedCIDRs, err := PrepareTrustedCIDRs(trustedProxies)
	if err != nil {
		return err
	}
	c.trustedCIDRs = append(c.trustedCIDRs, trustedCIDRs...)
	return nil
}

func (c *Config) TrustAll() *Config {
	c.trustedMutex.Lock()
	c.trustedProxies = make([]string, len(defaultTrustedProxies))
	copy(c.trustedProxies, defaultTrustedProxies)
	c.trustedCIDRs = defaultTrustedCIDRs
	c.trustedDefault = true
	c.trustedMutex.Unlock()
	return c
}

func (c *Config) SetTrustedProxiesByEnv() error {
	envValue := os.Getenv(EnvKey)
	c.envMutex.RLock()
	oldEnvValue := c.envTrustedProxies
	c.envMutex.RUnlock()
	if oldEnvValue == envValue {
		return nil
	}
	c.envMutex.Lock()
	c.envTrustedProxies = envValue
	if len(envValue) == 0 && len(oldEnvValue) > 0 {
		c.envTrustedCIDRs = nil
	}
	c.envMutex.Unlock()
	if len(envValue) == 0 {
		return nil
	}
	items := strings.Split(envValue, `,`)
	trustedProxies := make([]string, 0, len(items))
	for _, tp := range items {
		tp = strings.TrimSpace(tp)
		if len(tp) > 0 {
			trustedProxies = append(trustedProxies, tp)
		}
	}
	if len(trustedProxies) > 0 {
		trustedCIDRs, err := PrepareTrustedCIDRs(trustedProxies)
		if err != nil {
			return err
		}
		c.envMutex.Lock()
		c.envTrustedCIDRs = trustedCIDRs
		c.envMutex.Unlock()
	}
	return nil
}

func (c *Config) WatchEnvValue(ctx context.Context, dur time.Duration) error {
	t := time.NewTicker(dur)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			err := c.SetTrustedProxiesByEnv()
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return context.Canceled
		}
	}
}

func (c *Config) StartWatchEnv(ctx context.Context, dur time.Duration) *Config {
	err := c.SetTrustedProxiesByEnv()
	if err != nil {
		log.Println(err)
	}
	go func() {
		err := c.WatchEnvValue(ctx, dur)
		if err != nil {
			log.Println(err)
		}
	}()
	return c
}

// IsUnsafeTrustedProxies checks if Engine.trustedCIDRs contains all IPs, it's not safe if it has (returns true)
func (c *Config) IsUnsafeTrustedProxies() bool {
	for _, ip := range defaultUnsafeTrustedIPs {
		if c.isTrustedProxy(ip) {
			return true
		}
	}
	return false
}

// isTrustedProxy will check whether the IP address is included in the trusted list according to Engine.trustedCIDRs
func (c *Config) isTrustedProxy(ip net.IP) bool {
	c.trustedMutex.RLock()
	trustedCIDRs := c.trustedCIDRs
	c.trustedMutex.RUnlock()

	for _, cidr := range trustedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}

	c.envMutex.RLock()
	envTrustedCIDRs := c.envTrustedCIDRs
	c.envMutex.RUnlock()

	for _, cidr := range envTrustedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// ValidateIPHeader will parse X-Forwarded-For header and return the trusted client IP address
func (c *Config) ValidateIPHeader(headerValue string, headerName string, ignorePrivateIP bool) (clientIP string, valid bool) {
	if len(headerValue) == 0 {
		return
	}
	var items []string
	if headerName == HeaderForwarded {
		items = ParseHeaderForwarded(headerValue)
	} else {
		items = strings.Split(headerValue, ",")
	}
	for i := len(items) - 1; i >= 0; i-- {
		clientIP = strings.TrimSpace(items[i])
		if len(clientIP) == 0 {
			continue
		}
		ip := net.ParseIP(clientIP)
		if ip == nil {
			break
		}
		if ignorePrivateIP {
			isPrivate, err := IsPrivateIP(ip)
			if err != nil || isPrivate {
				continue
			}
		}
		// X-Forwarded-For is appended by proxy
		// Check IPs in reverse order and stop when find untrusted proxy
		// 如果客户端伪造 IP 地址，格式为：X-Forwarded-For: 伪造的 IP 地址 1, [伪造的 IP 地址 2...], IP0(client), IP1(proxy), IP2(proxy)。
		if i == 0 || !c.isTrustedProxy(ip) {
			valid = true
			return
		}
	}
	return
}

// ClientIP implements one best effort algorithm to return the real client IP.
// It calls c.RemoteIP() under the hood, to check if the remote IP is a trusted proxy or not.
// If it is it will then try to parse the headers defined in Engine.RemoteIPHeaders (defaulting to [X-Forwarded-For, X-Real-Ip]).
// If the headers are not syntactically valid OR the remote IP does not correspond to a trusted proxy,
// the remote IP (coming from Request.RemoteAddr) is returned.
func (c *Config) ClientIP(remoteAddress string, header func(string) string) string {
	// It also checks if the remoteIP is a trusted proxy or not.
	// In order to perform this validation, it will see if the IP is contained within at least one of the CIDR blocks
	// defined by Config.SetTrustedProxies()
	remoteAddress = c.RemoteIP(remoteAddress)
	if len(remoteAddress) == 0 {
		return ""
	}
	remoteIP := net.ParseIP(remoteAddress)
	if remoteIP == nil {
		return ""
	}
	trusted := c.isTrustedProxy(remoteIP)
	if trusted && c.forwardedByClientIP.Load() {
		c.ipHeaderMutex.RLock()
		remoteIPHeaders := c.remoteIPHeaders
		c.ipHeaderMutex.RUnlock()
		if remoteIPHeaders != nil {
			ignorePrivateIP := c.ignorePrivateIP.Load()
			for _, headerName := range remoteIPHeaders {
				ip, valid := c.ValidateIPHeader(header(headerName), headerName, ignorePrivateIP)
				if valid {
					return ip
				}
			}
		}
	}
	return remoteIP.String()
}

// RemoteIP parses the IP from Request.RemoteAddr, normalizes and returns the IP (without the port).
func (c *Config) RemoteIP(remoteAddress string) string {
	remoteAddress = strings.TrimSpace(remoteAddress)
	if len(remoteAddress) == 0 {
		return ""
	}
	var cutset string
	if remoteAddress[0] == '[' {
		cutset = `]:`
	} else {
		cutset = `:`
	}
	if !strings.Contains(remoteAddress, cutset) {
		return remoteAddress
	}
	ip, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		log.Printf(`[realip] failed to net.SplitHostPort(%q): %v`, remoteAddress, err.Error())
		return ""
	}
	return ip
}
