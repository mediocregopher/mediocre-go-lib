package mredis

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest"

	"github.com/mediocregopher/radix/v3"
)

func TestRedis(t *T) {
	cmp := mtest.Component()
	redis := InstRedis(cmp)
	mtest.Run(cmp, t, func() {
		var info string
		if err := redis.Do(radix.Cmd(&info, "INFO")); err != nil {
			t.Fatal(err)
		} else if len(info) < 0 {
			t.Fatal("empty info return")
		}
	})
}
