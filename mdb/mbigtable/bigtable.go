// Package mbigtable implements connecting to Google's Bigtable service and
// simplifying a number of interactions with it.
package mbigtable

import (
	"context"
	"strings"

	"cloud.google.com/go/bigtable"
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mdb"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

func isErrAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasSuffix(err.Error(), " already exists")
}

// Bigtable is a wrapper around a bigtable client providing more functionality.
type Bigtable struct {
	*bigtable.Client
	Instance string

	gce *mdb.GCE
	ctx context.Context
}

// WithBigTable returns a Bigtable instance which will be initialized and
// configured when the start event is triggered on the returned Context (see
// mrun.Start). The Bigtable instance will have Close called on it when the
// stop event is triggered on the returned Context (see mrun.Stop).
//
// gce is optional and can be passed in if there's an existing gce object which
// should be used, otherwise a new one will be created with mdb.MGCE.
//
// defaultInstance can be given as the instance name to use as the default
// parameter value. If empty the parameter will be required to be set.
func WithBigTable(parent context.Context, gce *mdb.GCE, defaultInstance string) (context.Context, *Bigtable) {
	ctx := mctx.NewChild(parent, "bigtable")
	if gce == nil {
		ctx, gce = mdb.WithGCE(ctx, "")
	}

	bt := &Bigtable{
		gce: gce,
	}

	var inst *string
	{
		const name, descr = "instance", "name of the bigtable instance in the project to connect to"
		if defaultInstance != "" {
			ctx, inst = mcfg.WithString(ctx, name, defaultInstance, descr)
		} else {
			ctx, inst = mcfg.WithRequiredString(ctx, name, descr)
		}
	}

	ctx = mrun.WithStartHook(ctx, func(innerCtx context.Context) error {
		bt.Instance = *inst

		bt.ctx = mctx.MergeAnnotations(bt.ctx, bt.gce.Context())
		bt.ctx = mctx.Annotate(bt.ctx, "instance", bt.Instance)

		mlog.Info("connecting to bigtable", bt.ctx)
		var err error
		bt.Client, err = bigtable.NewClient(
			innerCtx,
			bt.gce.Project, bt.Instance,
			bt.gce.ClientOptions()...,
		)
		return merr.Wrap(err, bt.ctx)
	})
	ctx = mrun.WithStopHook(ctx, func(context.Context) error {
		return bt.Client.Close()
	})
	bt.ctx = ctx
	return mctx.WithChild(parent, ctx), bt
}

// EnsureTable ensures that the given table exists and has (at least) the given
// column families.
//
// This method requires admin privileges on the bigtable instance.
func (bt *Bigtable) EnsureTable(ctx context.Context, name string, colFams ...string) error {
	ctx = mctx.MergeAnnotations(ctx, bt.ctx)
	ctx = mctx.Annotate(ctx, "table", name)
	mlog.Info("ensuring table", ctx)

	mlog.Debug("creating admin client", ctx)
	adminClient, err := bigtable.NewAdminClient(ctx, bt.gce.Project, bt.Instance)
	if err != nil {
		return merr.Wrap(err, ctx)
	}
	defer adminClient.Close()

	mlog.Debug("creating bigtable table (if needed)", ctx)
	err = adminClient.CreateTable(ctx, name)
	if err != nil && !isErrAlreadyExists(err) {
		return merr.Wrap(err, ctx)
	}

	for _, colFam := range colFams {
		ctx := mctx.Annotate(ctx, "family", colFam)
		mlog.Debug("creating bigtable column family (if needed)", ctx)
		err := adminClient.CreateColumnFamily(ctx, name, colFam)
		if err != nil && !isErrAlreadyExists(err) {
			return merr.Wrap(err, ctx)
		}
	}

	return nil
}

// Table returns the bigtable.Table instance which can be used to write/query
// the given table.
func (bt *Bigtable) Table(tableName string) *bigtable.Table {
	return bt.Open(tableName)
}
