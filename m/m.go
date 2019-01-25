// Package m is the glue which holds all the other packages in this project
// together. While other packages in this project are intended to be able to be
// used separately and largely independently, this package combines them in ways
// which I specifically like.
package m

import (
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

// TODO Create a function, `NewService() mctx.Context` which preloads the
// context with log-level param. Will one day also add debug server. Problem
// comes because mlog isn't quite designed right and setting log-level from
// config won't propagate changes to child contexts which have already called
// mlog.From.

// Run performs the work of populating configuration parameters, triggering the
// start event, waiting for an interrupt, and then triggering the stop event.
// Run will block until the stop event is done. If any errors are encountered a
// fatal is thrown.
func Run(ctx mctx.Context) {
	log := mlog.From(ctx)
	if err := mcfg.Populate(ctx, CfgSource()); err != nil {
		log.Fatal("error populating configuration", merr.KV(err))
	} else if err := mrun.Start(ctx); err != nil {
		log.Fatal("error triggering start event", merr.KV(err))
	}

	{
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		s := <-ch
		log.Info("signal received, stopping", mlog.KV{"signal": s})
	}

	if err := mrun.Stop(ctx); err != nil {
		log.Fatal("error triggering stop event", merr.KV(err))
	}
	log.Info("exiting process")
}
