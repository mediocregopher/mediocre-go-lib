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

// MNew returns a Bigtable instance which will be initialized and configured
// when the start event is triggered on the returned Context (see mrun.Start).
// The Bigtable instance will have Close called on it when the stop event is
// triggered on the returned Context (see mrun.Stop).
//
// gce is optional and can be passed in if there's an existing gce object which
// should be used, otherwise a new one will be created with mdb.MGCE.
//
// defaultInstance can be given as the instance name to use as the default
// parameter value. If empty the parameter will be required to be set.
func MNew(ctx context.Context, gce *mdb.GCE, defaultInstance string) (context.Context, *Bigtable) {
	if gce == nil {
		ctx, gce = mdb.MGCE(ctx, "")
	}

	bt := &Bigtable{
		gce: gce,
		ctx: mctx.NewChild(ctx, "bigtable"),
	}

	// TODO the equivalent functionality as here will be added with annotations
	// bt.log.SetKV(bt)

	var inst *string
	{
		const name, descr = "instance", "name of the bigtable instance in the project to connect to"
		if defaultInstance != "" {
			bt.ctx, inst = mcfg.String(bt.ctx, name, defaultInstance, descr)
		} else {
			bt.ctx, inst = mcfg.RequiredString(bt.ctx, name, descr)
		}
	}

	bt.ctx = mrun.OnStart(bt.ctx, func(innerCtx context.Context) error {
		bt.Instance = *inst
		mlog.Info(bt.ctx, "connecting to bigtable", bt)
		var err error
		bt.Client, err = bigtable.NewClient(
			innerCtx,
			bt.gce.Project, bt.Instance,
			bt.gce.ClientOptions()...,
		)
		return merr.WithKV(err, bt.KV())
	})
	bt.ctx = mrun.OnStop(bt.ctx, func(context.Context) error {
		return bt.Client.Close()
	})
	return mctx.WithChild(ctx, bt.ctx), bt
}

// KV implements the mlog.KVer interface.
func (bt *Bigtable) KV() map[string]interface{} {
	kv := bt.gce.KV()
	kv["bigtableInstance"] = bt.Instance
	return kv
}

// EnsureTable ensures that the given table exists and has (at least) the given
// column families.
//
// This method requires admin privileges on the bigtable instance.
func (bt *Bigtable) EnsureTable(ctx context.Context, name string, colFams ...string) error {
	kv := mlog.KV{"bigtableTable": name}
	mlog.Info(bt.ctx, "ensuring table", kv)

	mlog.Debug(bt.ctx, "creating admin client", kv)
	adminClient, err := bigtable.NewAdminClient(ctx, bt.gce.Project, bt.Instance)
	if err != nil {
		return merr.WithKV(err, bt.KV(), kv.KV())
	}
	defer adminClient.Close()

	mlog.Debug(bt.ctx, "creating bigtable table (if needed)", kv)
	err = adminClient.CreateTable(ctx, name)
	if err != nil && !isErrAlreadyExists(err) {
		return merr.WithKV(err, bt.KV(), kv.KV())
	}

	for _, colFam := range colFams {
		kv := kv.Set("family", colFam)
		mlog.Debug(bt.ctx, "creating bigtable column family (if needed)", kv)
		err := adminClient.CreateColumnFamily(ctx, name, colFam)
		if err != nil && !isErrAlreadyExists(err) {
			return merr.WithKV(err, bt.KV(), kv.KV())
		}
	}

	return nil
}

// Table returns the bigtable.Table instance which can be used to write/query
// the given table.
func (bt *Bigtable) Table(tableName string) *bigtable.Table {
	return bt.Open(tableName)
}
