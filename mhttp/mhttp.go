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

// Server is returned by WithListeningServer and simply wraps an *http.Server.
type Server struct {
	*http.Server
	ctx context.Context
}

// WithListeningServer returns a *Server which will be initialized and have
// ListenAndServe called on it (asynchronously) when the start event is
// triggered on the returned Context (see mrun.Start). The Server will have
// Shutdown called on it when the stop event is triggered on the returned
// Context (see mrun.Stop).
//
// This function automatically handles setting up configuration parameters via
// mcfg. The default listen address is ":0".
func WithListeningServer(ctx context.Context, h http.Handler) (context.Context, *Server) {
	srv := &Server{
		Server: &http.Server{Handler: h},
		ctx:    mctx.NewChild(ctx, "http"),
	}

	var listener *mnet.Listener
	srv.ctx, listener = mnet.WithListener(srv.ctx, "tcp", "")
	listener.NoCloseOnStop = true // http.Server.Shutdown will do this

	srv.ctx = mrun.WithStartHook(srv.ctx, func(context.Context) error {
		srv.Addr = listener.Addr().String()
		srv.ctx = mrun.WithThreads(srv.ctx, 1, func() error {
			mlog.Info("serving requests", srv.ctx)
			if err := srv.Serve(listener); !merr.Equal(err, http.ErrServerClosed) {
				mlog.Error("error serving listener", srv.ctx, merr.Context(err))
				return err
			}
			return nil
		})
		return nil
	})

	srv.ctx = mrun.WithStopHook(srv.ctx, func(innerCtx context.Context) error {
		mlog.Info("shutting down server", srv.ctx)
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
