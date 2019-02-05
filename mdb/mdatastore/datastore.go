// Package mdatastore implements connecting to Google's Datastore service and
// simplifying a number of interactions with it.
package mdatastore

import (
	"context"

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
	ctx context.Context
}

// MNew returns a Datastore instance which will be initialized and configured
// when the start event is triggered on the returned Context (see mrun.Start).
// The Datastore instance will have Close called on it when the stop event is
// triggered on the returned Context (see mrun.Stop).
//
// gce is optional and can be passed in if there's an existing gce object which
// should be used, otherwise a new one will be created with mdb.MGCE.
func MNew(ctx context.Context, gce *mdb.GCE) (context.Context, *Datastore) {
	if gce == nil {
		ctx, gce = mdb.MGCE(ctx, "")
	}

	ds := &Datastore{
		gce: gce,
		ctx: mctx.NewChild(ctx, "datastore"),
	}

	// TODO the equivalent functionality as here will be added with annotations
	// ds.log.SetKV(ds)

	ds.ctx = mrun.OnStart(ds.ctx, func(innerCtx context.Context) error {
		mlog.Info(ds.ctx, "connecting to datastore")
		var err error
		ds.Client, err = datastore.NewClient(innerCtx, ds.gce.Project, ds.gce.ClientOptions()...)
		return merr.WithKV(err, ds.KV())
	})
	ds.ctx = mrun.OnStop(ds.ctx, func(context.Context) error {
		return ds.Client.Close()
	})
	return mctx.WithChild(ctx, ds.ctx), ds
}

// KV implements the mlog.KVer interface.
func (ds *Datastore) KV() map[string]interface{} {
	return ds.gce.KV()
}
