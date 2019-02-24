// Package msql implements connecting to a MySQL/MariaDB instance (and possibly
// others) and simplifies a number of interactions with it.
package msql

import (
	"context"
	"fmt"

	// If something is importing msql it must need mysql, because that's all
	// that is implemented at the moment
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

// SQL is a wrapper around a sqlx client which provides more functionality.
type SQL struct {
	*sqlx.DB
	ctx context.Context
}

// WithMySQL returns a SQL instance which will be initialized and configured
// when the start event is triggered on the returned Context (see mrun.Start).
// The SQL instance will have Close called on it when the stop event is
// triggered on the returned Context (see mrun.Stop).
//
// defaultDB indicates the name of the database in MySQL to use by default,
// though it will be overwritable in the config.
func WithMySQL(parent context.Context, defaultDB string) (context.Context, *SQL) {
	ctx := mctx.NewChild(parent, "mysql")
	sql := new(SQL)

	ctx, addr := mcfg.WithString(ctx, "addr", "[::1]:3306", "Address where mysql server can be found")
	ctx, user := mcfg.WithString(ctx, "user", "root", "User to authenticate to mysql server as")
	ctx, pass := mcfg.WithString(ctx, "password", "", "Password to authenticate to mysql server with")
	ctx, db := mcfg.WithString(ctx, "database", defaultDB, "mysql database to use")
	ctx = mrun.WithStartHook(ctx, func(innerCtx context.Context) error {
		sql.ctx = mctx.Annotate(sql.ctx, "addr", *addr, "user", *user)

		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", *user, *pass, *addr, *db)
		mlog.Debug("constructed dsn", mctx.Annotate(sql.ctx, "dsn", dsn))
		mlog.Info("connecting to mysql server", sql.ctx)
		var err error
		sql.DB, err = sqlx.ConnectContext(innerCtx, "mysql", dsn)
		return merr.Wrap(sql.ctx, err)
	})
	ctx = mrun.WithStopHook(ctx, func(innerCtx context.Context) error {
		mlog.Info("closing connection to sql server", sql.ctx)
		return sql.Close()
	})

	sql.ctx = ctx
	return mctx.WithChild(parent, ctx), sql
}
