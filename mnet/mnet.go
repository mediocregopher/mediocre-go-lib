// Package mnet extends the standard package with extra functionality which is
// commonly useful
package mnet

import (
	"context"
	"net"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mcmp"
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

	cmp *mcmp.Component
}

type listenerOpts struct {
	proto           string
	defaultAddr     string
	closeOnShutdown bool
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

// ListenerCloseOnShutdown sets the Listener's behavior when mrun's Shutdown
// event is triggered on its Component. If true the Listener will call Close on
// itself, if false it will do nothing.
//
// Defaults to true.
func ListenerCloseOnShutdown(closeOnShutdown bool) ListenerOpt {
	return func(opts *listenerOpts) {
		opts.closeOnShutdown = closeOnShutdown
	}
}

// ListenerDefaultAddr adjusts the defaultAddr which the Listener will use. The
// addr will still be configurable via mcfg regardless of what this is set to.
// The default is ":0".
func ListenerDefaultAddr(defaultAddr string) ListenerOpt {
	return func(opts *listenerOpts) {
		opts.defaultAddr = defaultAddr
	}
}

// InstListener instantiates a Listener which will be initialized when the Init
// event is triggered on the given Component, and closed when the Shutdown event
// is triggered on the returned Component.
func InstListener(cmp *mcmp.Component, opts ...ListenerOpt) *Listener {
	lOpts := listenerOpts{
		proto:           "tcp",
		defaultAddr:     ":0",
		closeOnShutdown: true,
	}
	for _, opt := range opts {
		opt(&lOpts)
	}

	cmp = cmp.Child("net")
	l := &Listener{cmp: cmp}

	addr := mcfg.String(cmp, "listen-addr",
		mcfg.ParamDefault(lOpts.defaultAddr),
		mcfg.ParamUsage(
			strings.ToUpper(lOpts.proto)+" address to listen on in format "+
				"[host]:port. If port is 0 then a random one will be chosen",
		),
	)

	mrun.InitHook(cmp, func(context.Context) error {
		var err error

		cmp.Annotate("proto", lOpts.proto, "addr", *addr)

		if lOpts.isPacketConn() {
			l.PacketConn, err = net.ListenPacket(lOpts.proto, *addr)
			cmp.Annotate("addr", l.PacketConn.LocalAddr().String())
		} else {
			l.Listener, err = net.Listen(lOpts.proto, *addr)
			cmp.Annotate("addr", l.Listener.Addr().String())
		}
		if err != nil {
			return merr.Wrap(err, cmp.Context())
		}

		mlog.From(cmp).Info("listening")
		return nil
	})

	// TODO track connections and wait for them to complete before shutting
	// down?
	mrun.ShutdownHook(cmp, func(context.Context) error {
		if !lOpts.closeOnShutdown {
			return nil
		}
		mlog.From(cmp).Info("shutting down listener")
		return l.Close()
	})

	return l
}

// Accept wraps a call to Accept on the underlying net.Listener, providing debug
// logging.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return conn, err
	}
	mlog.From(l.cmp).Debug("connection accepted",
		mctx.Annotated("remoteAddr", conn.RemoteAddr().String()))
	return conn, nil
}

// Close wraps a call to Close on the underlying net.Listener, providing debug
// logging.
func (l *Listener) Close() error {
	mlog.From(l.cmp).Info("listener closing")
	if l.Listener != nil {
		return l.Listener.Close()
	}
	return l.PacketConn.Close()
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
