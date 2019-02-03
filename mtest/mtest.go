// Package mtest implements functionality useful for testing.
package mtest

import (
	"testing"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
)

type ctxKey int

// NewCtx creates and returns a root Context suitable for testing.
func NewCtx() mctx.Context {
	ctx := mctx.New()
	mlog.From(ctx).SetMaxLevel(mlog.DebugLevel)
	return ctx
}

// SetEnv sets the given environment variable on the given Context, such that it
// will be used as if it was a real environment variable when the Run function
// from this package is called.
func SetEnv(ctx mctx.Context, key, val string) {
	mctx.GetSetMutableValue(ctx, false, ctxKey(0), func(i interface{}) interface{} {
		m, _ := i.(map[string]string)
		if m == nil {
			m = map[string]string{}
		}
		m[key] = val
		return m
	})
}

// Run performs the following using the given Context:
//
// - Calls mcfg.Populate using any variables set by SetEnv.
//
// - Calls mrun.Start
//
// - Calls the passed in body callback.
//
// - Calls mrun.Stop
//
// The intention is that Run is used within a test on a Context created via
// NewCtx, after any setup functions have been called (e.g. mnet.MListen).
func Run(ctx mctx.Context, t *testing.T, body func()) {
	envMap, _ := mctx.MutableValue(ctx, ctxKey(0)).(map[string]string)
	env := make([]string, 0, len(envMap))
	for key, val := range envMap {
		env = append(env, key+"="+val)
	}

	if err := mcfg.Populate(ctx, mcfg.SourceEnv{Env: env}); err != nil {
		t.Fatal(err)
	} else if err := mrun.Start(ctx); err != nil {
		t.Fatal(err)
	}

	body()

	if err := mrun.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
