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

// Log returns a Logger which will automatically include with the log extra
// contextual information based on the Cfg and the given KVs
//
// If the cfg is nil then mlog.DefaultLogger is returned.
func Log(cfg *mcfg.Cfg, kvs ...mlog.KVer) *mlog.Logger {
	fn := cfg.FullName()
	l := mlog.DefaultLogger.WithWriteFn(func(w io.Writer, msg mlog.Message) error {
		msg.Msg = fn + " " + msg.Msg
		return mlog.DefaultWriteFn(w, msg)
	})
	if len(kvs) > 0 {
		l = l.WithKV(kvs...)
	}
	return l
}
