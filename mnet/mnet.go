// Package mnet extends the standard package with extra functionality which is
// commonly useful
package mnet

import (
	"context"
	"net"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

// Listener is returned by WithListen and simply wraps a net.Listener.
type Listener struct {
	// One of these will be populated during the start hook, depending on the
	// protocol configured.
	net.Listener
	net.PacketConn

	ctx context.Context

	// If set to true before mrun's stop event is run, the stop event will not
	// cause the MListener to be closed.
	NoCloseOnStop bool
}

type listenerOpts struct {
	proto       string
	defaultAddr string
}

func (lOpts listenerOpts) isPacketConn() bool {
	proto := strings.ToLower(lOpts.proto)
	return strings.HasPrefix(proto, "udp") ||
		proto == "unixgram" ||
		strings.HasPrefix(proto, "ip")
}

// ListenerOpt is a value which adjusts the behavior of WithListener.
type ListenerOpt func(*listenerOpts)

// ListenerProtocol adjusts the protocol which the Listener uses. The default is
// "tcp".
func ListenerProtocol(proto string) ListenerOpt {
	return func(opts *listenerOpts) {
		opts.proto = proto
	}
}

// ListenerDefaultAddr adjusts the defaultAddr which the Listener will use. The
// addr will still be configurable via mcfg regardless of what this is set to.
// The default is ":0".
func ListenerAddr(defaultAddr string) ListenerOpt {
	return func(opts *listenerOpts) {
		opts.defaultAddr = defaultAddr
	}
}

// WithListener returns a Listener which will be initialized when the start
// event is triggered on the returned Context (see mrun.Start), and closed when
// the stop event is triggered on the returned Context (see mrun.Stop).
func WithListener(ctx context.Context, opts ...ListenerOpt) (context.Context, *Listener) {
	lOpts := listenerOpts{
		proto:       "tcp",
		defaultAddr: ":0",
	}
	for _, opt := range opts {
		opt(&lOpts)
	}

	l := &Listener{
		ctx: mctx.NewChild(ctx, "net"),
	}

	var addr *string
	l.ctx, addr = mcfg.WithString(l.ctx, "listen-addr", lOpts.defaultAddr, strings.ToUpper(lOpts.proto)+" address to listen on in format [host]:port. If port is 0 then a random one will be chosen")

	l.ctx = mrun.WithStartHook(l.ctx, func(context.Context) error {
		var err error

		l.ctx = mctx.Annotate(l.ctx,
			"proto", lOpts.proto,
			"addr", *addr)

		if lOpts.isPacketConn() {
			l.PacketConn, err = net.ListenPacket(lOpts.proto, *addr)
			l.ctx = mctx.Annotate(l.ctx, "addr", l.PacketConn.LocalAddr().String())
		} else {
			l.Listener, err = net.Listen(lOpts.proto, *addr)
			l.ctx = mctx.Annotate(l.ctx, "addr", l.Listener.Addr().String())
		}
		if err != nil {
			return merr.Wrap(err, l.ctx)
		}

		mlog.Info("listening", l.ctx)
		return nil
	})

	// TODO track connections and wait for them to complete before shutting
	// down?
	l.ctx = mrun.WithStopHook(l.ctx, func(context.Context) error {
		if l.NoCloseOnStop {
			return nil
		}
		mlog.Info("stopping listener", l.ctx)
		return l.Close()
	})

	return mctx.WithChild(ctx, l.ctx), l
}

// Accept wraps a call to Accept on the underlying net.Listener, providing debug
// logging.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return conn, err
	}
	mlog.Debug("connection accepted",
		mctx.Annotate(l.ctx, "remoteAddr", conn.RemoteAddr().String()))
	return conn, nil
}

// Close wraps a call to Close on the underlying net.Listener, providing debug
// logging.
func (l *Listener) Close() error {
	mlog.Info("listener closing", l.ctx)
	return l.Listener.Close()
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
