// Package mtest implements functionality useful for testing.
package mtest

import (
	"context"
	"testing"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

type envCtxKey int

// Context creates and returns a root Context suitable for testing.
func Context() context.Context {
	ctx := context.Background()
	logger := mlog.NewLogger()
	logger.SetMaxLevel(mlog.DebugLevel)
	return mlog.WithLogger(ctx, logger)
}

// WithEnv sets the given environment variable on the given Context, such that
// it will be used as if it was a real environment variable when the Run
// function from this package is called.
func WithEnv(ctx context.Context, key, val string) context.Context {
	prevEnv, _ := ctx.Value(envCtxKey(0)).([][2]string)
	env := make([][2]string, len(prevEnv), len(prevEnv)+1)
	copy(env, prevEnv)
	env = append(env, [2]string{key, val})
	return context.WithValue(ctx, envCtxKey(0), env)
}

// Run performs the following using the given Context:
//
// - Calls mcfg.Populate using any variables set by WithEnv.
//
// - Calls mrun.Start
//
// - Calls the passed in body callback.
//
// - Calls mrun.Stop
//
// The intention is that Run is used within a test on a Context created via
// NewCtx, after any setup functions have been called (e.g. mnet.WithListener).
func Run(ctx context.Context, t *testing.T, body func()) {
	envTups, _ := ctx.Value(envCtxKey(0)).([][2]string)
	env := make([]string, 0, len(envTups))
	for _, tup := range envTups {
		env = append(env, tup[0]+"="+tup[1])
	}

	ctx, err := mcfg.Populate(ctx, &mcfg.SourceEnv{Env: env})
	if err != nil {
		t.Fatal(err)
	} else if err := mrun.Start(ctx); err != nil {
		t.Fatal(err)
	}

	body()

	if err := mrun.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
