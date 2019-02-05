// Package mnet extends the standard package with extra functionality which is
// commonly useful
package mnet

import (
	"context"
	"net"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

// MListener is returned by MListen and simply wraps a net.Listener.
type MListener struct {
	net.Listener
	ctx context.Context

	// If set to true before mrun's stop event is run, the stop event will not
	// cause the MListener to be closed.
	NoCloseOnStop bool
}

// MListen returns an MListener which will be initialized when the start event
// is triggered on the returned Context (see mrun.Start), and closed when the
// stop event is triggered on the returned Context (see mrun.Stop).
//
// network defaults to "tcp" if empty. defaultAddr defaults to ":0" if empty,
// and will be configurable via mcfg.
func MListen(ctx context.Context, network, defaultAddr string) (context.Context, *MListener) {
	if network == "" {
		network = "tcp"
	}
	if defaultAddr == "" {
		defaultAddr = ":0"
	}

	l := new(MListener)

	// TODO the equivalent functionality as here will be added with annotations
	//l.log = mlog.From(ctx)
	//l.log.SetKV(l)

	ctx, addr := mcfg.String(ctx, "listen-addr", defaultAddr, strings.ToUpper(network)+" address to listen on in format [host]:port. If port is 0 then a random one will be chosen")
	ctx = mrun.OnStart(ctx, func(context.Context) error {
		var err error
		if l.Listener, err = net.Listen(network, *addr); err != nil {
			return err
		}
		mlog.Info(l.ctx, "listening")
		return nil
	})

	// TODO track connections and wait for them to complete before shutting
	// down?
	ctx = mrun.OnStop(ctx, func(context.Context) error {
		if l.NoCloseOnStop {
			return nil
		}
		mlog.Info(l.ctx, "stopping listener")
		return l.Close()
	})

	l.ctx = ctx
	return ctx, l
}

// Accept wraps a call to Accept on the underlying net.Listener, providing debug
// logging.
func (l *MListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return conn, err
	}
	mlog.Debug(l.ctx, "connection accepted", mlog.KV{"remoteAddr": conn.RemoteAddr()})
	return conn, nil
}

// Close wraps a call to Close on the underlying net.Listener, providing debug
// logging.
func (l *MListener) Close() error {
	mlog.Debug(l.ctx, "listener closing")
	err := l.Listener.Close()
	mlog.Debug(l.ctx, "listener closed", merr.KV(err))
	return err
}

// KV implements the mlog.KVer interface.
func (l *MListener) KV() map[string]interface{} {
	return map[string]interface{}{"addr": l.Addr().String()}
}

////////////////////////////////////////////////////////////////////////////////

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
