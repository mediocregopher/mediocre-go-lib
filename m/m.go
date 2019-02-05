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

// NewServiceCtx returns a Context which should be used as the root Context when
// creating long running services, such as an RPC service or database.
//
// The returned Context will automatically handle setting up global
// configuration parameters like "log-level", as well as an http endpoint where
// debug information about the running process can be accessed.
//
// TODO set up the debug endpoint.
func NewServiceCtx() context.Context {
	ctx := context.Background()

	// set up log level handling
	logger := mlog.NewLogger()
	ctx = mlog.Set(ctx, logger)
	ctx, logLevelStr := mcfg.String(ctx, "log-level", "info", "Maximum log level which will be printed.")
	ctx = mrun.OnStart(ctx, func(context.Context) error {
		logLevel := mlog.LevelFromString(*logLevelStr)
		if logLevel == nil {
			return merr.New("invalid log level", "log-level", *logLevelStr)
		}
		logger.SetMaxLevel(logLevel)
		return nil
	})

	return ctx
}

// Run performs the work of populating configuration parameters, triggering the
// start event, waiting for an interrupt, and then triggering the stop event.
// Run will block until the stop event is done. If any errors are encountered a
// fatal is thrown.
func Run(ctx context.Context) {
	log := mlog.From(ctx)
	if err := mcfg.Populate(ctx, CfgSource()); err != nil {
		log.Fatal(ctx, "error populating configuration", merr.KV(err))
	} else if err := mrun.Start(ctx); err != nil {
		log.Fatal(ctx, "error triggering start event", merr.KV(err))
	}

	{
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		s := <-ch
		log.Info(ctx, "signal received, stopping", mlog.KV{"signal": s})
	}

	if err := mrun.Stop(ctx); err != nil {
		log.Fatal(ctx, "error triggering stop event", merr.KV(err))
	}
	log.Info(ctx, "exiting process")
}
