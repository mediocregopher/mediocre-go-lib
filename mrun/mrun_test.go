package mrun

import (
	"context"
	"errors"
	. "testing"
	"time"
)

func TestThreadWait(t *T) {
	testErr := errors.New("test error")

	cancelCh := func(t time.Duration) <-chan struct{} {
		tCtx, _ := context.WithTimeout(context.Background(), t*2)
		return tCtx.Done()
	}

	wait := func(ctx context.Context, shouldTake time.Duration) error {
		start := time.Now()
		err := Wait(ctx, cancelCh(shouldTake*2))
		if took := time.Since(start); took < shouldTake || took > shouldTake*4/3 {
			t.Fatalf("wait took %v, should have taken %v", took, shouldTake)
		}
		return err
	}

	t.Run("noBlock", func(t *T) {
		t.Run("noErr", func(t *T) {
			ctx := context.Background()
			ctx = WithThreads(ctx, 1, func() error { return nil })
			if err := Wait(ctx, nil); err != nil {
				t.Fatal(err)
			}
		})

		t.Run("err", func(t *T) {
			ctx := context.Background()
			ctx = WithThreads(ctx, 1, func() error { return testErr })
			if err := Wait(ctx, nil); err != testErr {
				t.Fatalf("should have got test error, got: %v", err)
			}
		})
	})

	t.Run("block", func(t *T) {
		t.Run("noErr", func(t *T) {
			ctx := context.Background()
			ctx = WithThreads(ctx, 1, func() error {
				time.Sleep(1 * time.Second)
				return nil
			})
			if err := wait(ctx, 1*time.Second); err != nil {
				t.Fatal(err)
			}
		})

		t.Run("err", func(t *T) {
			ctx := context.Background()
			ctx = WithThreads(ctx, 1, func() error {
				time.Sleep(1 * time.Second)
				return testErr
			})
			if err := wait(ctx, 1*time.Second); err != testErr {
				t.Fatalf("should have got test error, got: %v", err)
			}
		})

		t.Run("canceled", func(t *T) {
			ctx := context.Background()
			ctx = WithThreads(ctx, 1, func() error {
				time.Sleep(5 * time.Second)
				return testErr
			})
			if err := Wait(ctx, cancelCh(500*time.Millisecond)); err != ErrDone {
				t.Fatalf("should have got ErrDone, got: %v", err)
			}
		})
	})
}
