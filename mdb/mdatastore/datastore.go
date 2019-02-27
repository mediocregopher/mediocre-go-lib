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

// WithDatastore returns a Datastore instance which will be initialized and
// configured when the start event is triggered on the returned Context (see
// mrun.Start). The Datastore instance will have Close called on it when the
// stop event is triggered on the returned Context (see mrun.Stop).
//
// gce is optional and can be passed in if there's an existing gce object which
// should be used, otherwise a new one will be created with mdb.MGCE.
func WithDatastore(parent context.Context, gce *mdb.GCE) (context.Context, *Datastore) {
	ctx := mctx.NewChild(parent, "datastore")
	if gce == nil {
		ctx, gce = mdb.WithGCE(ctx, "")
	}

	ds := &Datastore{
		gce: gce,
	}

	ctx = mrun.WithStartHook(ctx, func(innerCtx context.Context) error {
		ds.ctx = mctx.MergeAnnotations(ds.ctx, ds.gce.Context())
		mlog.Info("connecting to datastore", ds.ctx)
		var err error
		ds.Client, err = datastore.NewClient(innerCtx, ds.gce.Project, ds.gce.ClientOptions()...)
		return merr.Wrap(err, ds.ctx)
	})
	ctx = mrun.WithStopHook(ctx, func(context.Context) error {
		return ds.Client.Close()
	})
	ds.ctx = ctx
	return mctx.WithChild(parent, ctx), ds
}
