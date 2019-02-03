// Package mdatastore implements connecting to Google's Datastore service and
// simplifying a number of interactions with it.
package mdatastore

import (
	"cloud.google.com/go/datastore"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mdb"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

// Datastore is a wrapper around a datastore client providing more
// functionality.
type Datastore struct {
	*datastore.Client

	gce *mdb.GCE
	log *mlog.Logger
}

// MNew returns a Datastore instance which will be initialized and configured
// when the start event is triggered on ctx (see mrun.Start). The Datastore
// instance will have Close called on it when the stop event is triggered on ctx
// (see mrun.Stop).
//
// gce is optional and can be passed in if there's an existing gce object which
// should be used, otherwise a new one will be created with mdb.MGCE.
func MNew(ctx mctx.Context, gce *mdb.GCE) *Datastore {
	if gce == nil {
		gce = mdb.MGCE(ctx, "")
	}

	ctx = mctx.ChildOf(ctx, "datastore")
	ds := &Datastore{
		gce: gce,
		log: mlog.From(ctx),
	}
	ds.log.SetKV(ds)

	mrun.OnStart(ctx, func(innerCtx mctx.Context) error {
		ds.log.Info("connecting to datastore")
		var err error
		ds.Client, err = datastore.NewClient(innerCtx, ds.gce.Project, ds.gce.ClientOptions()...)
		return merr.WithKV(err, ds.KV())
	})
	mrun.OnStop(ctx, func(mctx.Context) error {
		return ds.Client.Close()
	})
	return ds
}

// KV implements the mlog.KVer interface.
func (ds *Datastore) KV() map[string]interface{} {
	return ds.gce.KV()
}
