package mnet

import (
	"net"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest/massert"
)

func TestIsReservedIP(t *T) {
	assertReserved := func(ipStr string) massert.Assertion {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			panic("ip:" + ipStr + " not valid")
		}
		return massert.Comment(massert.Equal(true, IsReservedIP(ip)),
			"ip:%q", ipStr)
	}

	massert.Fatal(t, massert.All(
		assertReserved("127.0.0.1"),
		assertReserved("::ffff:127.0.0.1"),
		assertReserved("192.168.40.50"),
		assertReserved("::1"),
		assertReserved("100::1"),
	))

	massert.Fatal(t, massert.None(
		assertReserved("8.8.8.8"),
		assertReserved("::ffff:8.8.8.8"),
		assertReserved("2600:1700:7580:6e80:21c:25ff:fe97:44df"),
	))
}
