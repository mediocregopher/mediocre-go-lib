// Package m is the glue which holds all the other packages in this project
// together. While other packages in this project are intended to be able to be
// used separately and largely independently, this package combines them in ways
// which I specifically like.
package m

import (
	"io"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
)

// TODO this isn't going to work. At some point there will be something like
// mhttp which will have some glue code for it in here, but then something
// within it which logs (like the http server), and then there'll be an import
// cycle.
//
// Ultimately it might be worthwhile to make Logger be a field on Cfg, that
// really seems like the only other place where code like this makes sense to
// go. But then that's greatly expanding the purview of Cfg, which is
// unfortunate....

// Log returns a Logger which will automatically include with the log extra
// contextual information based on the Cfg and the given KVs
//
// If the cfg is nil then mlog.DefaultLogger is returned.
func Log(cfg *mcfg.Cfg, kvs ...mlog.KVer) *mlog.Logger {
	fn := cfg.FullName()
	l := mlog.DefaultLogger.WithWriteFn(func(w io.Writer, msg mlog.Message) error {
		msg.Msg = "(" + fn + ") " + msg.Msg
		return mlog.DefaultWriteFn(w, msg)
	})
	if len(kvs) > 0 {
		l = l.WithKV(kvs...)
	}
	return l
}
