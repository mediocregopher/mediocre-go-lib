package mcfg

import (
	"context"
	. "testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHook(t *T) {
	{ // test Then
		aCh := make(chan bool)
		bCh := make(chan bool)
		h := Nop()
		h.Then(func(context.Context) error {
			aCh <- true
			<-aCh
			return nil
		})
		h.Then(func(context.Context) error {
			bCh <- true
			<-bCh
			return nil
		})
		errCh := make(chan error)
		go func() {
			errCh <- h(context.Background())
		}()

		assert.True(t, <-aCh)
		// make sure bCh isn't being written to till aCh is closed
		select {
		case <-bCh:
			assert.Fail(t, "bCh shouldn't be written to yet")
		case <-time.After(250 * time.Millisecond):
			close(aCh)
		}
		assert.True(t, <-bCh)
		// make sure errCh isn't being written to till bCh is closed
		select {
		case <-errCh:
			assert.Fail(t, "errCh shouldn't be written to yet")
		case <-time.After(250 * time.Millisecond):
			close(bCh)
		}
		assert.Nil(t, <-errCh)
	}

	{ // test Also
		aCh := make(chan bool)
		bCh := make(chan bool)
		h := Nop()
		h.Also(func(context.Context) error {
			aCh <- true
			<-aCh
			return nil
		})
		h.Also(func(context.Context) error {
			bCh <- true
			<-bCh
			return nil
		})
		errCh := make(chan error)
		go func() {
			errCh <- h(context.Background())
		}()

		// both channels should get written to, then closed, then errCh should
		// get written to
		assert.True(t, <-aCh)
		assert.True(t, <-bCh)
		// make sure errCh isn't being written to till both channels are written
		select {
		case <-errCh:
			assert.Fail(t, "errCh shouldn't be written to yet")
		case <-time.After(250 * time.Millisecond):
			close(aCh)
			close(bCh)
		}
		assert.Nil(t, <-errCh)
	}
}

func TestPopulateParams(t *T) {
	{
		cfg := New()
		a := cfg.ParamInt("a", 0, "")
		cfgChild := cfg.Child("foo")
		b := cfgChild.ParamInt("b", 0, "")
		c := cfgChild.ParamInt("c", 0, "")

		err := cfg.populateParams(SourceCLI{
			Args: []string{"--a=1", "--foo-b=2"},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, *a)
		assert.Equal(t, 2, *b)
		assert.Equal(t, 0, *c)
	}

	{ // test that required params are enforced
		cfg := New()
		a := cfg.ParamInt("a", 0, "")
		cfgChild := cfg.Child("foo")
		b := cfgChild.ParamInt("b", 0, "")
		c := cfgChild.ParamRequiredInt("c", "")

		err := cfg.populateParams(SourceCLI{
			Args: []string{"--a=1", "--foo-b=2"},
		})
		assert.Error(t, err)

		err = cfg.populateParams(SourceCLI{
			Args: []string{"--a=1", "--foo-b=2", "--foo-c=3"},
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, *a)
		assert.Equal(t, 2, *b)
		assert.Equal(t, 3, *c)
	}
}
