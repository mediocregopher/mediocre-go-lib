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

type cfgSrcKey int

// ProcessContext returns a Context which should be used as the root Context
// when implementing most processes.
//
// The returned Context will automatically handle setting up global
// configuration parameters like "log-level", as well as an http endpoint where
// debug information about the running process can be accessed.
func ProcessContext() context.Context {
	ctx := context.Background()

	// embed confuration source which should be used into the context.
	ctx = context.WithValue(ctx, cfgSrcKey(0), mcfg.Source(new(mcfg.SourceCLI)))

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

// ServiceContext extends ProcessContext so that it better supports long running
// processes which are expected to handle requests from outside clients.
//
// Additional behavior it adds includes setting up an http endpoint where debug
// information about the running process can be accessed.
func ServiceContext() context.Context {
	ctx := ProcessContext()

	// services expect to use many different configuration sources
	ctx = context.WithValue(ctx, cfgSrcKey(0), mcfg.Source(mcfg.Sources{
		new(mcfg.SourceEnv),
		new(mcfg.SourceCLI),
	}))

	// TODO set up the debug endpoint.
	return ctx
}

// Start performs the work of populating configuration parameters and triggering
// the start event. It will return once the Start event has completed running.
//
// This function returns a Context because there are cases where the Context
// will be modified during Start, such as if WithSubCommand was used. If the
// Context was not modified then the passed in Context will be returned.
func Start(ctx context.Context) context.Context {
	src, _ := ctx.Value(cfgSrcKey(0)).(mcfg.Source)
	if src == nil {
		mlog.Fatal("ctx not sourced from m package", ctx)
	}

	// no logging should happen before populate, primarily because log-level
	// hasn't been populated yet, but also because it makes help output on cli
	// look weird.
	ctx, err := mcfg.Populate(ctx, src)
	if err != nil {
		mlog.Fatal("error populating configuration", ctx, merr.Context(err))
	} else if err := mrun.Start(ctx); err != nil {
		mlog.Fatal("error triggering start event", ctx, merr.Context(err))
	}
	mlog.Info("start hooks completed", ctx)
	return ctx
}

// StartWaitStop performs the work of populating configuration parameters,
// triggering the start event, waiting for an interrupt, and then triggering the
// stop event.  Run will block until the stop event is done. If any errors are
// encountered a fatal is thrown.
func StartWaitStop(ctx context.Context) {
	ctx = Start(ctx)

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
