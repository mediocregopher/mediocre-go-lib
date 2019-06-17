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

	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mnet"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

// Server is returned by WithListeningServer and simply wraps an *http.Server.
type Server struct {
	*http.Server
	cmp *mcmp.Component
}

// InstListeningServer returns a *Server which will be initialized and have
// ListenAndServe called on it (asynchronously) when the Init event is triggered
// on the Component. The Server will have Shutdown called on it when the
// Shutdown event is triggered on the Component.
//
// This function automatically handles setting up configuration parameters via
// mcfg. The default listen address is ":0".
func InstListeningServer(cmp *mcmp.Component, h http.Handler) *Server {
	srv := &Server{
		Server: &http.Server{Handler: h},
		cmp:    cmp.Child("http"),
	}

	listener := mnet.InstListener(srv.cmp,
		// http.Server.Shutdown will handle this
		mnet.ListenerCloseOnShutdown(false),
	)

	threadCtx := context.Background()
	mrun.InitHook(srv.cmp, func(ctx context.Context) error {
		srv.Addr = listener.Addr().String()
		threadCtx = mrun.WithThreads(threadCtx, 1, func() error {
			mlog.From(srv.cmp).Info("serving requests", ctx)
			if err := srv.Serve(listener); !merr.Equal(err, http.ErrServerClosed) {
				mlog.From(srv.cmp).Error("error serving listener", ctx, merr.Context(err))
				return merr.Wrap(err, srv.cmp.Context(), ctx)
			}
			return nil
		})
		return nil
	})

	mrun.ShutdownHook(srv.cmp, func(ctx context.Context) error {
		mlog.From(srv.cmp).Info("shutting down server", ctx)
		if err := srv.Shutdown(ctx); err != nil {
			return merr.Wrap(err, srv.cmp.Context(), ctx)
		}
		err := mrun.Wait(threadCtx, ctx.Done())
		return merr.Wrap(err, srv.cmp.Context(), ctx)
	})

	return srv
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
