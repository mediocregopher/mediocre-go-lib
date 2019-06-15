// Package mtest implements functionality useful for testing.
package mtest

import (
	"context"
	"testing"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

type envCmpKey int

// Component creates and returns a root Component suitable for testing.
func Component() *mcmp.Component {
	cmp := new(mcmp.Component)
	logger := mlog.NewLogger()
	logger.SetMaxLevel(mlog.DebugLevel)
	mlog.SetLogger(cmp, logger)

	mrun.InitHook(cmp, func(context.Context) error {
		envVals := mcmp.SeriesValues(cmp, envCmpKey(0))
		env := make([]string, 0, len(envVals))
		for _, val := range envVals {
			tup := val.([2]string)
			env = append(env, tup[0]+"="+tup[1])
		}
		return mcfg.Populate(cmp, &mcfg.SourceEnv{Env: env})
	})

	return cmp
}

// Env sets the given environment variable on the given Component, such that it
// will be used as if it was a real environment variable when the Run function
// from this package is called.
//
// This function will panic if not called on the root Component.
func Env(cmp *mcmp.Component, key, val string) {
	if len(cmp.Path()) != 0 {
		panic("Env should only be called on the root Component")
	}
	mcmp.AddSeriesValue(cmp, envCmpKey(0), [2]string{key, val})
}

// Run performs the following using the given Component:
//
// - Calls mrun.Init, which calls mcfg.Populate using any variables set by Env.
//
// - Calls the passed in body callback.
//
// - Calls mrun.Shutdown
//
// The intention is that Run is used within a test on a Component created via
// this package's Component function, after any setup functions have been called
// (e.g. mnet.AddListener).
func Run(cmp *mcmp.Component, t *testing.T, body func()) {
	if err := mrun.Init(context.Background(), cmp); err != nil {
		t.Fatal(err)
	}

	body()

	if err := mrun.Shutdown(context.Background(), cmp); err != nil {
		t.Fatal(err)
	}
}
