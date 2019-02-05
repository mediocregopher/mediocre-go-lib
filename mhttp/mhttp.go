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

// MServer is returned by MListenAndServe and simply wraps an *http.Server.
type MServer struct {
	*http.Server
	ctx context.Context
}

// MListenAndServe returns an http.Server which will be initialized and have
// ListenAndServe called on it (asynchronously) when the start event is
// triggered on the returned Context (see mrun.Start). The Server will have
// Shutdown called on it when the stop event is triggered on the returned
// Context (see mrun.Stop).
//
// This function automatically handles setting up configuration parameters via
// mcfg. The default listen address is ":0".
func MListenAndServe(ctx context.Context, h http.Handler) (context.Context, *MServer) {
	srv := &MServer{
		Server: &http.Server{Handler: h},
		ctx:    mctx.NewChild(ctx, "http"),
	}

	var listener *mnet.MListener
	srv.ctx, listener = mnet.MListen(srv.ctx, "tcp", "")
	listener.NoCloseOnStop = true // http.Server.Shutdown will do this

	// TODO the equivalent functionality as here will be added with annotations
	//logger := mlog.From(ctx)
	//logger.SetKV(listener)

	srv.ctx = mrun.OnStart(srv.ctx, func(context.Context) error {
		srv.Addr = listener.Addr().String()
		srv.ctx = mrun.Thread(srv.ctx, func() error {
			if err := srv.Serve(listener); err != http.ErrServerClosed {
				mlog.Error(srv.ctx, "error serving listener", merr.KV(err))
				return err
			}
			return nil
		})
		return nil
	})

	srv.ctx = mrun.OnStop(srv.ctx, func(innerCtx context.Context) error {
		mlog.Info(srv.ctx, "shutting down server")
		if err := srv.Shutdown(innerCtx); err != nil {
			return err
		}
		return mrun.Wait(srv.ctx, innerCtx.Done())
	})

	return mctx.WithChild(ctx, srv.ctx), srv
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
