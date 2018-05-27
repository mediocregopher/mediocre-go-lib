// Package mhttp extends the standard package with extra functionality which is
// commonly useful
package mhttp

import (
	"context"
	"net/http"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
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
		mlog.Info("HTTP server listening", kv)
		go func() {
			err := srv.ListenAndServe()
			// TODO the listening log should happen here, somehow, now that we
			// know the actual address being listened on
			mlog.Fatal("http server fataled", kv, mlog.ErrKV(err))
		}()
		return nil
	})

	// TODO shutdown logic
	return &srv
}
