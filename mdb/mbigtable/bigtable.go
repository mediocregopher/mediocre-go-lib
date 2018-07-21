// Package mbigtable implements connecting to Google's Bigtable service and
// simplifying a number of interactions with it.
package mbigtable

import (
	"context"
	"strings"

	"cloud.google.com/go/bigtable"
	"github.com/mediocregopher/mediocre-go-lib/m"
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mdb"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
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
	log *mlog.Logger
}

// Cfg configurs and returns a Bigtable instance which will be usable once
// StartRun is called on the passed in Cfg instance.
//
// defaultInstance can be given as the instance name to use as the default
// parameter value. If empty the parameter will be required to be set.
func Cfg(cfg *mcfg.Cfg, defaultInstance string) *Bigtable {
	cfg = cfg.Child("bigtable")
	var bt Bigtable
	bt.gce = mdb.CfgGCE(cfg)
	bt.log = m.Log(cfg, &bt)

	var inst *string
	{
		name, descr := "instance", "name of the bigtable instance in the project to connect to"
		if defaultInstance != "" {
			inst = cfg.ParamString(name, defaultInstance, descr)
		} else {
			inst = cfg.ParamRequiredString(name, descr)
		}
	}

	cfg.Start.Then(func(ctx context.Context) error {
		bt.Instance = *inst
		bt.log.Info("connecting to bigtable")
		var err error
		bt.Client, err = bigtable.NewClient(
			ctx,
			bt.gce.Project, bt.Instance,
			bt.gce.ClientOptions()...,
		)
		return mlog.ErrWithKV(err, &bt)
	})
	return &bt
}

// KV implements the mlog.KVer interface.
func (bt *Bigtable) KV() mlog.KV {
	return bt.gce.KV().Set("instance", bt.Instance)
}

// EnsureTable ensures that the given table exists and has (at least) the given
// column families.
//
// This method requires admin privileges on the bigtable instance.
func (bt *Bigtable) EnsureTable(ctx context.Context, name string, colFams ...string) error {
	kv := bt.KV().Set("table", name)
	bt.log.Info("ensuring table", kv)

	bt.log.Debug("creating admin client", kv)
	adminClient, err := bigtable.NewAdminClient(ctx, bt.gce.Project, bt.Instance)
	if err != nil {
		return mlog.ErrWithKV(err, kv)
	}
	defer adminClient.Close()

	bt.log.Debug("creating bigtable table (if needed)", kv)
	err = adminClient.CreateTable(ctx, name)
	if err != nil && !isErrAlreadyExists(err) {
		return mlog.ErrWithKV(err, kv)
	}

	for _, colFam := range colFams {
		kv := kv.Set("family", colFam)
		bt.log.Debug("creating bigtable column family (if needed)", kv)
		err := adminClient.CreateColumnFamily(ctx, name, colFam)
		if err != nil && !isErrAlreadyExists(err) {
			return mlog.ErrWithKV(err, kv)
		}
	}

	return nil
}

// Table returns the bigtable.Table instance which can be used to write/query
// the given table.
func (bt *Bigtable) Table(tableName string) *bigtable.Table {
	return bt.Open(tableName)
}
