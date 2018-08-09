// Package mnet extends the standard package with extra functionality which is
// commonly useful
package mnet

import (
	"net"
)

func mustGetCIDRNetwork(cidr string) *net.IPNet {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return n
}

// https://en.wikipedia.org/wiki/Reserved_IP_addresses

var reservedCIDRs4 = []*net.IPNet{
	mustGetCIDRNetwork("0.0.0.0/8"),          // current network
	mustGetCIDRNetwork("10.0.0.0/8"),         // private network
	mustGetCIDRNetwork("100.64.0.0/10"),      // private network
	mustGetCIDRNetwork("127.0.0.0/8"),        // localhost
	mustGetCIDRNetwork("169.254.0.0/16"),     // link-local
	mustGetCIDRNetwork("172.16.0.0/12"),      // private network
	mustGetCIDRNetwork("192.0.0.0/24"),       // IETF protocol assignments
	mustGetCIDRNetwork("192.0.2.0/24"),       // documentation and examples
	mustGetCIDRNetwork("192.88.99.0/24"),     // 6to4 Relay
	mustGetCIDRNetwork("192.168.0.0/16"),     // private network
	mustGetCIDRNetwork("198.18.0.0/15"),      // private network
	mustGetCIDRNetwork("198.51.100.0/24"),    // documentation and examples
	mustGetCIDRNetwork("203.0.113.0/24"),     // documentation and examples
	mustGetCIDRNetwork("224.0.0.0/4"),        // IP multicast
	mustGetCIDRNetwork("240.0.0.0/4"),        // reserved
	mustGetCIDRNetwork("255.255.255.255/32"), // limited broadcast address
}

var reservedCIDRs6 = []*net.IPNet{
	mustGetCIDRNetwork("::/128"),        // unspecified address
	mustGetCIDRNetwork("::1/128"),       // loopback address
	mustGetCIDRNetwork("100::/64"),      // discard prefix
	mustGetCIDRNetwork("2001::/32"),     // Teredo tunneling
	mustGetCIDRNetwork("2001:20::/28"),  // ORCHID v2
	mustGetCIDRNetwork("2001:db8::/32"), // documentation and examples
	mustGetCIDRNetwork("2002::/16"),     // 6to4 addressing
	mustGetCIDRNetwork("fc00::/7"),      // unique local
	mustGetCIDRNetwork("fe80::/10"),     // link local
	mustGetCIDRNetwork("ff00::/8"),      // multicast
}

// IsReservedIP returns true if the given valid IP is part of a reserved IP
// range.
func IsReservedIP(ip net.IP) bool {
	containedBy := func(cidrs []*net.IPNet) bool {
		for _, cidr := range cidrs {
			if cidr.Contains(ip) {
				return true
			}
		}
		return false
	}

	if ip.To4() != nil {
		return containedBy(reservedCIDRs4)
	}
	return containedBy(reservedCIDRs6)
}
