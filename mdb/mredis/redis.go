// Package mredis implements connecting to a redis instance.
package mredis

import (
	"context"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
	"github.com/mediocregopher/radix/v3"
)

// Redis is a wrapper around a redis client which provides more functionality.
type Redis struct {
	radix.Client
	cmp *mcmp.Component
}

// InstRedis instantiates a Redis instance which will be initialized when the
// Init event is triggered on the given Component. The redis client will have
// Close called on it when the Shutdown event is triggered on the given
// Component.
func InstRedis(parent *mcmp.Component) *Redis {
	cmp := parent.Child("redis")
	client := new(struct{ radix.Client })

	addr := mcfg.String(cmp, "addr",
		mcfg.ParamDefault("127.0.0.1:6379"),
		mcfg.ParamUsage("Address redis is listening on"))
	poolSize := mcfg.Int(cmp, "pool-size",
		mcfg.ParamDefault(4),
		mcfg.ParamUsage("Number of connections in pool"))
	mrun.InitHook(cmp, func(ctx context.Context) error {
		cmp.Annotate("addr", *addr, "poolSize", *poolSize)
		mlog.From(cmp).Info("connecting to redis", ctx)
		var err error
		client.Client, err = radix.NewPool("tcp", *addr, *poolSize)
		return err
	})
	mrun.ShutdownHook(cmp, func(ctx context.Context) error {
		mlog.From(cmp).Info("shutting down redis", ctx)
		return client.Close()
	})

	return &Redis{
		Client: client,
		cmp:    cmp,
	}
}
