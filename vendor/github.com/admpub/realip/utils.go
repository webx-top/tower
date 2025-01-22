package realip

import (
	"errors"
	"net"
	"strings"
)

func IsPrivateAddress(address string) (bool, error) {
	ipAddress := net.ParseIP(address)
	return IsPrivateIP(ipAddress)
}

// isPrivateIP works by checking if the address is under private CIDR blocks.
// List of private CIDR blocks can be seen on :
//
// https://en.wikipedia.org/wiki/Private_network
//
// https://en.wikipedia.org/wiki/Link-local_address
func IsPrivateIP(ipAddress net.IP) (bool, error) {
	if ipAddress == nil {
		return false, errors.New("address is not valid")
	}
	for i := range defaultPrivateCIDRs {
		if defaultPrivateCIDRs[i].Contains(ipAddress) {
			return true, nil
		}
	}

	return false, nil
}

// parseIP parse a string representation of an IP and returns a net.IP with the
// minimum byte representation or nil if input is invalid.
func ParseIP(ip string) net.IP {
	parsedIP := net.ParseIP(ip)

	if ipv4 := parsedIP.To4(); ipv4 != nil {
		// return ip in a 4-byte representation
		return ipv4
	}

	// return ip in a 16-byte representation or nil
	return parsedIP
}

func ParseCIDR(ipStr string) (*net.IPNet, error) {
	if !strings.Contains(ipStr, "/") {
		ip := ParseIP(ipStr)
		if ip == nil {
			return nil, &net.ParseError{Type: "IP address", Text: ipStr}
		}

		switch len(ip) {
		case net.IPv4len:
			ipStr += "/32"
		case net.IPv6len:
			ipStr += "/128"
		}
	}
	_, cidrNet, err := net.ParseCIDR(ipStr)
	return cidrNet, err
}

func ParseHeaderForwarded(headerValue string) []string {
	values := strings.Split(headerValue, ";")
	items := make([]string, 0, len(values))
	for _, item := range values {
		item = strings.TrimSpace(item)
		if len(item) == 0 {
			continue
		}
		if !strings.HasPrefix(item, `for=`) {
			continue
		}
		for _, vfor := range strings.Split(item, ",") {
			vfor = strings.TrimSpace(vfor)
			if len(vfor) == 0 {
				continue
			}
			if !strings.HasPrefix(vfor, `for=`) {
				continue
			}
			vfor = strings.TrimPrefix(vfor, `for=`)
			vfor = strings.Trim(vfor, `"`)
			if len(vfor) == 0 {
				continue
			}
			items = append(items, vfor)
		}
	}
	return items
}
