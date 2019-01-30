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

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mnet"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

// MListenAndServe returns an http.Server which will be initialized and have
// ListenAndServe called on it (asynchronously) when the start event is
// triggered on ctx (see mrun.Start). The Server will have Shutdown called on it
// when the stop event is triggered on ctx (see mrun.Stop).
//
// This function automatically handles setting up configuration parameters via
// mcfg. The default listen address is ":0".
func MListenAndServe(ctx mctx.Context, h http.Handler) *http.Server {
	ctx = mctx.ChildOf(ctx, "http")
	listener := mnet.MListen(ctx, "tcp", "")
	listener.NoCloseOnStop = true // http.Server.Shutdown will do this

	logger := mlog.From(ctx)
	logger.SetKV(listener)

	srv := http.Server{Handler: h}
	mrun.OnStart(ctx, func(mctx.Context) error {
		srv.Addr = listener.Addr().String()
		mrun.Thread(ctx, func() error {
			if err := srv.Serve(listener); err != http.ErrServerClosed {
				logger.Error("error serving listener", merr.KV(err))
				return err
			}
			return nil
		})
		return nil
	})

	mrun.OnStop(ctx, func(innerCtx mctx.Context) error {
		logger.Info("shutting down server")
		if err := srv.Shutdown(context.Context(innerCtx)); err != nil {
			return err
		}
		return mrun.Wait(ctx, innerCtx.Done())
	})

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
