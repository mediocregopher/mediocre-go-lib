// Package mbigquery implements connecting to Google's BigQuery service and
// simplifying a number of interactions with it.
package mbigquery

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/m"
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mdb"
	"github.com/mediocregopher/mediocre-go-lib/mlog"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
)

// TODO this file needs tests

func isErrAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 409 {
		return true
	}
	return false
}

// BigQuery is a wrapper around a bigquery client providing more functionality.
type BigQuery struct {
	*bigquery.Client
	gce *mdb.GCE
	log *mlog.Logger

	// key is dataset/tableName
	tablesL        sync.Mutex
	tables         map[[2]string]*bigquery.Table
	tableUploaders map[[2]string]*bigquery.Uploader
}

// Cfg configures and returns a BigQuery instance which will be usable once Run
// is called on the passed in Cfg instance.
func Cfg(cfg *mcfg.Cfg) *BigQuery {
	cfg = cfg.Child("bigquery")
	bq := BigQuery{
		gce:            mdb.CfgGCE(cfg),
		tables:         map[[2]string]*bigquery.Table{},
		tableUploaders: map[[2]string]*bigquery.Uploader{},
	}
	bq.log = m.Log(cfg, &bq)
	cfg.Start.Then(func(ctx context.Context) error {
		bq.log.Info("connecting to bigquery")
		var err error
		bq.Client, err = bigquery.NewClient(ctx, bq.gce.Project, bq.gce.ClientOptions()...)
		return mlog.ErrWithKV(err, &bq)
	})
	return &bq
}

// KV implements the mlog.KVer interface.
func (bq *BigQuery) KV() mlog.KV {
	return bq.gce.KV()
}

// Table initializes and returns the table instance with the given dataset and
// schema information. This method caches the Table/Uploader instances it
// returns, so multiple calls with the same dataset/tableName will only actually
// create those instances on the first call.
func (bq *BigQuery) Table(
	ctx context.Context,
	dataset, tableName string,
	schemaObj interface{},
) (
	*bigquery.Table, *bigquery.Uploader, error,
) {
	bq.tablesL.Lock()
	defer bq.tablesL.Unlock()

	key := [2]string{dataset, tableName}
	if table, ok := bq.tables[key]; ok {
		return table, bq.tableUploaders[key], nil
	}

	kv := mlog.KV{"dataset": dataset, "table": tableName}
	bq.log.Debug("creating/grabbing table", kv)

	schema, err := bigquery.InferSchema(schemaObj)
	if err != nil {
		return nil, nil, mlog.ErrWithKV(err, bq, kv)
	}

	ds := bq.Dataset(dataset)
	if err := ds.Create(ctx, nil); err != nil && !isErrAlreadyExists(err) {
		return nil, nil, mlog.ErrWithKV(err, bq, kv)
	}

	table := ds.Table(tableName)
	meta := &bigquery.TableMetadata{
		Name:   tableName,
		Schema: schema,
	}
	if err := table.Create(ctx, meta); err != nil && !isErrAlreadyExists(err) {
		return nil, nil, mlog.ErrWithKV(err, bq, kv)
	}
	uploader := table.Uploader()

	bq.tables[key] = table
	bq.tableUploaders[key] = uploader
	return table, uploader, nil
}

////////////////////////////////////////////////////////////////////////////////

const timeFormat = "2006-01-02 15:04:05 MST"

// Time wraps a time.Time object and provides marshaling/unmarshaling for
// bigquery's time format.
type Time struct {
	time.Time
}

// MarshalText implements the encoding.TextMarshaler interface.
func (t Time) MarshalText() ([]byte, error) {
	str := t.Time.Format(timeFormat)
	return []byte(str), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (t *Time) UnmarshalText(b []byte) error {
	tt, err := time.Parse(timeFormat, string(b))
	if err != nil {
		return err
	}
	t.Time = tt
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (t *Time) MarshalJSON() ([]byte, error) {
	b, err := t.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(b))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *Time) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	return t.UnmarshalText([]byte(str))
}
