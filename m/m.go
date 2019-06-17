// Package m implements functionality specific to how I like my programs to
// work. It acts as glue between many of the other packages in this framework,
// putting them together in the way I find most useful.
package m

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

type cmpKey int

const (
	cmpKeyCfgSrc cmpKey = iota
	cmpKeyInfoLog
)

func debugLog(cmp *mcmp.Component, msg string, ctxs ...context.Context) {
	level := mlog.DebugLevel
	if len(ctxs) > 0 {
		if ok, _ := ctxs[0].Value(cmpKeyInfoLog).(bool); ok {
			level = mlog.InfoLevel
		}
	}

	mlog.From(cmp).Log(mlog.Message{
		Level:       level,
		Description: msg,
		Contexts:    ctxs,
	})
}

// RootComponent returns a Component which should be used as the root Component
// when implementing most programs.
//
// The returned Component will automatically handle setting up global
// configuration parameters like "log-level", as well as parsing those
// and all other parameters when the Init even is triggered on it.
func RootComponent() *mcmp.Component {
	cmp := new(mcmp.Component)

	// embed confuration source which should be used into the context.
	cmp.SetValue(cmpKeyCfgSrc, mcfg.Source(new(mcfg.SourceCLI)))

	// set up log level handling
	logger := mlog.NewLogger()
	mlog.SetLogger(cmp, logger)

	// set up parameter parsing
	mrun.InitHook(cmp, func(context.Context) error {
		src, _ := cmp.Value(cmpKeyCfgSrc).(mcfg.Source)
		if src == nil {
			return merr.New("Component not sourced from m package", cmp.Context())
		} else if err := mcfg.Populate(cmp, src); err != nil {
			return merr.Wrap(err, cmp.Context())
		}
		return nil
	})

	logLevelStr := mcfg.String(cmp, "log-level",
		mcfg.ParamDefault("info"),
		mcfg.ParamUsage("Maximum log level which will be printed."))
	mrun.InitHook(cmp, func(context.Context) error {
		logLevel := mlog.LevelFromString(*logLevelStr)
		if logLevel == nil {
			return merr.New("invalid log level", cmp.Context(),
				mctx.Annotated("log-level", *logLevelStr))
		}
		logger.SetMaxLevel(logLevel)
		return nil
	})

	return cmp
}

// RootServiceComponent extends RootComponent so that it better supports long
// running processes which are expected to handle requests from outside clients.
//
// Additional behavior it adds includes setting up an http endpoint where debug
// information about the running process can be accessed.
func RootServiceComponent() *mcmp.Component {
	cmp := RootComponent()

	// services expect to use many different configuration sources
	cmp.SetValue(cmpKeyCfgSrc, mcfg.Source(mcfg.Sources{
		new(mcfg.SourceEnv),
		new(mcfg.SourceCLI),
	}))

	// it's useful to show debug entries (from this package specifically) as
	// info logs for long-running services.
	cmp.SetValue(cmpKeyInfoLog, true)

	// TODO set up the debug endpoint.
	return cmp
}

// MustInit will call mrun.Init on the given Component, which must have been
// created in this package, and exit the process if mrun.Init does not complete
// successfully.
func MustInit(cmp *mcmp.Component) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	debugLog(cmp, "initializing")
	if err := mrun.Init(ctx, cmp); err != nil {
		mlog.From(cmp).Fatal("initialization failed", merr.Context(err))
	}
	debugLog(cmp, "initialization completed successfully")
}

// MustShutdown is like MustInit, except that it triggers the Shutdown event on
// the Component.
func MustShutdown(cmp *mcmp.Component) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	debugLog(cmp, "shutting down")
	if err := mrun.Shutdown(ctx, cmp); err != nil {
		mlog.From(cmp).Fatal("shutdown failed", merr.Context(err))
	}
	debugLog(cmp, "shutting down completed successfully")
}

// Exec calls MustInit on the given Component, then blocks until an interrupt
// signal is received, then calls MustShutdown on the Component, until finally
// exiting the process.
func Exec(cmp *mcmp.Component) {
	MustInit(cmp)
	{
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		s := <-ch
		debugLog(cmp, "signal received, stopping", mctx.Annotated("signal", s))
	}
	MustShutdown(cmp)

	debugLog(cmp, "exiting process")
	os.Stdout.Sync()
	os.Stderr.Sync()
	os.Exit(0)
}
