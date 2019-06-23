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
	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

// SQL is a wrapper around a sqlx client which provides more functionality.
type SQL struct {
	*sqlx.DB
	cmp *mcmp.Component
}

// InstMySQL returns a SQL instance which will be initialized when the Init
// event is triggered on the given Component. The SQL instance will have Close
// called on it when the Shutdown event is triggered on the given Component.
//
// defaultDB indicates the name of the database in MySQL to use by default,
// though it will be overwritable in the config.
func InstMySQL(cmp *mcmp.Component, defaultDB string) *SQL {
	sql := SQL{cmp: cmp.Child("mysql")}

	addr := mcfg.String(sql.cmp, "addr",
		mcfg.ParamDefault("[::1]:3306"),
		mcfg.ParamUsage("Address where MySQL server can be found"))
	user := mcfg.String(sql.cmp, "user",
		mcfg.ParamDefault("root"),
		mcfg.ParamUsage("User to authenticate to MySQL server as"))
	pass := mcfg.String(sql.cmp, "password",
		mcfg.ParamUsage("Password to authenticate to MySQL server with"))
	db := mcfg.String(sql.cmp, "database",
		mcfg.ParamDefault(defaultDB),
		mcfg.ParamUsage("MySQL database to use"))

	mrun.InitHook(sql.cmp, func(ctx context.Context) error {
		sql.cmp.Annotate("addr", *addr, "user", *user)
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", *user, *pass, *addr, *db)
		mlog.From(sql.cmp).Debug("constructed dsn", mctx.Annotate(ctx, "dsn", dsn))
		mlog.From(sql.cmp).Info("connecting to MySQL server", ctx)
		var err error
		sql.DB, err = sqlx.ConnectContext(ctx, "mysql", dsn)
		return merr.Wrap(err, sql.cmp.Context(), ctx)
	})

	mrun.ShutdownHook(sql.cmp, func(ctx context.Context) error {
		mlog.From(sql.cmp).Info("closing connection to MySQL server", ctx)
		return merr.Wrap(sql.Close(), sql.cmp.Context(), ctx)
	})

	return &sql
}

// Context returns the annotated Context from this instance's initialization.
func (sql *SQL) Context() context.Context {
	return sql.cmp.Context()
}
