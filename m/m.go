// Package m is the glue which holds all the other packages in this project
// together. While other packages in this project are intended to be able to be
// used separately and largely independently, this package combines them in ways
// which I specifically like.
package m

import (
	"context"
	"os"
	"os/signal"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

// CfgSource returns an mcfg.Source which takes in configuration info from the
// environment and from the CLI.
func CfgSource() mcfg.Source {
	return mcfg.Sources{
		mcfg.SourceEnv{},
		mcfg.SourceCLI{},
	}
}

// ServiceContext returns a Context which should be used as the root Context
// when creating long running services, such as an RPC service or database.
//
// The returned Context will automatically handle setting up global
// configuration parameters like "log-level", as well as an http endpoint where
// debug information about the running process can be accessed.
//
// TODO set up the debug endpoint.
func ServiceContext() context.Context {
	ctx := context.Background()

	// set up log level handling
	logger := mlog.NewLogger()
	ctx = mlog.WithLogger(ctx, logger)
	ctx, logLevelStr := mcfg.WithString(ctx, "log-level", "info", "Maximum log level which will be printed.")
	ctx = mrun.WithStartHook(ctx, func(context.Context) error {
		logLevel := mlog.LevelFromString(*logLevelStr)
		if logLevel == nil {
			ctx := mctx.Annotate(ctx, "log-level", *logLevelStr)
			return merr.New("invalid log level", ctx)
		}
		logger.SetMaxLevel(logLevel)
		return nil
	})

	return ctx
}

// Start performs the work of populating configuration parameters and triggering
// the start event. It will return once the Start event has completed running.
func Start(ctx context.Context) {
	// no logging should happen before populate, primarily because log-level
	// hasn't been populated yet, but also because it makes help output on cli
	// look weird.
	if err := mcfg.Populate(ctx, CfgSource()); err != nil {
		mlog.Fatal("error populating configuration", ctx, merr.Context(err))
	} else if err := mrun.Start(ctx); err != nil {
		mlog.Fatal("error triggering start event", ctx, merr.Context(err))
	}
	mlog.Info("start hooks completed", ctx)
}

// StartWaitStop performs the work of populating configuration parameters,
// triggering the start event, waiting for an interrupt, and then triggering the
// stop event.  Run will block until the stop event is done. If any errors are
// encountered a fatal is thrown.
func StartWaitStop(ctx context.Context) {
	Start(ctx)

	{
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		s := <-ch
		mlog.Info("signal received, stopping", mctx.Annotate(ctx, "signal", s))
	}

	if err := mrun.Stop(ctx); err != nil {
		mlog.Fatal("error triggering stop event", ctx, merr.Context(err))
	}
	mlog.Info("exiting process", ctx)
}
