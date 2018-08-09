// Package mhttp extends the standard package with extra functionality which is
// commonly useful
package mhttp

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/m"
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mnet"
)

// CfgServer initializes and returns an *http.Server which will initialize on
// the Start hook. This also sets up config params for configuring the address
// being listened on.
func CfgServer(cfg *mcfg.Cfg, h http.Handler) *http.Server {
	cfg = cfg.Child("http")
	addr := cfg.ParamString("addr", ":0", "Address to listen on. Default is any open port")

	srv := http.Server{Handler: h}
	cfg.Start.Then(func(ctx context.Context) error {
		srv.Addr = *addr
		kv := mlog.KV{"addr": *addr}
		log := m.Log(cfg, kv)
		log.Info("listening")
		go func() {
			err := srv.ListenAndServe()
			// TODO the listening log should happen here, somehow, now that we
			// know the actual address being listened on
			log.Fatal("http server fataled", mlog.ErrKV(err))
		}()
		return nil
	})

	// TODO shutdown logic
	return &srv
}

// AddXForwardedFor populates the X-Forwarded-For header on the Request to
// convey that the request is being proxied for IP.
//
// If the IP is invalid, loopback, or otherwise part of a reserved range, this
// does nothing.
func AddXForwardedFor(r *http.Request, ipStr string) {
	const xff = "X-Forwarded-For"
	ip := net.ParseIP(ipStr)
	if ip == nil || mnet.IsReservedIP(ip) { // IsReservedIP includes loopback
		return
	}
	prev, _ := r.Header[xff]
	r.Header.Set(xff, strings.Join(append(prev, ip.String()), ", "))
}

// ReverseProxy returns an httputil.ReverseProxy which will send requests to the
// given URL and copy their responses back without modification.
//
// Only the Scheme and Host of the given URL are used.
//
// Any http.ResponseWriters passed into the ServeHTTP call of the returned
// instance should not be modified afterwards.
func ReverseProxy(u *url.URL) *httputil.ReverseProxy {
	rp := new(httputil.ReverseProxy)
	rp.Director = func(req *http.Request) {
		if ipStr, _, err := net.SplitHostPort(req.RemoteAddr); err != nil {
			AddXForwardedFor(req, ipStr)
		}

		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
	}

	// TODO when this package has a function for creating a Client use that for
	// the default here

	return rp
}
