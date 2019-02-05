package mnet

import (
	"fmt"
	"io/ioutil"
	"net"
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest"
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

func TestMListen(t *T) {
	ctx := mtest.NewCtx()
	ctx, l := MListen(ctx, "", "")
	mtest.Run(ctx, t, func() {
		go func() {
			conn, err := net.Dial("tcp", l.Addr().String())
			if err != nil {
				t.Fatal(err)
			} else if _, err = fmt.Fprint(conn, "hello world"); err != nil {
				t.Fatal(err)
			}
			conn.Close()
		}()

		conn, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		} else if b, err := ioutil.ReadAll(conn); err != nil {
			t.Fatal(err)
		} else if string(b) != "hello world" {
			t.Fatalf("read %q from conn", b)
		}
	})
}
