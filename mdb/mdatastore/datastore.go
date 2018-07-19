// Package mdatastore implements connecting to Google's Datastore service and
// simplifying a number of interactions with it.
package mdatastore

import (
	"context"

	"cloud.google.com/go/datastore"
	"github.com/mediocregopher/mediocre-go-lib/m"
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mdb"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
)

// Datastore is a wrapper around a datastore client providing more
// functionality.
type Datastore struct {
	*datastore.Client

	gce *mdb.GCE
	log *mlog.Logger
}

// Cfg configures and returns a Datastore instance which will be usable once
// StartRun is called on the passed in Cfg instance.
func Cfg(cfg *mcfg.Cfg) *Datastore {
	cfg = cfg.Child("datastore")
	var ds Datastore
	ds.gce = mdb.CfgGCE(cfg)
	ds.log = m.Log(cfg, &ds)
	cfg.Start.Then(func(ctx context.Context) error {
		ds.log.Info("connecting to datastore")
		var err error
		ds.Client, err = datastore.NewClient(ctx, ds.gce.Project, ds.gce.ClientOptions()...)
		return mlog.ErrWithKV(err, &ds)
	})
	return &ds
}

// KV implements the mlog.KVer interface.
func (ds *Datastore) KV() mlog.KV {
	return ds.gce.KV()
}
