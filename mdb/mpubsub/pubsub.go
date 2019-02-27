// Package mpubsub implements connecting to Google's PubSub service and
// simplifying a number of interactions with it.
package mpubsub

import (
	"context"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mdb"
	"github.com/mediocregopher/mediocre-go-lib/merr"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
	oldctx "golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO Consume (and probably BatchConsume) don't properly handle the Client
// being closed.

func isErrAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	s, ok := status.FromError(err)
	return ok && s.Code() == codes.AlreadyExists
}

// Message aliases the type in the official driver
type Message = pubsub.Message

// PubSub is a wrapper around a pubsub client providing more functionality.
type PubSub struct {
	*pubsub.Client

	gce *mdb.GCE
	ctx context.Context
}

// WithPubSub returns a PubSub instance which will be initialized and configured
// when the start event is triggered on the returned Context (see mrun.Start).
// The PubSub instance will have Close called on it when the stop event is
// triggered on the returned Context(see mrun.Stop).
//
// gce is optional and can be passed in if there's an existing gce object which
// should be used, otherwise a new one will be created with mdb.MGCE.
func WithPubSub(parent context.Context, gce *mdb.GCE) (context.Context, *PubSub) {
	ctx := mctx.NewChild(parent, "pubsub")
	if gce == nil {
		ctx, gce = mdb.WithGCE(ctx, "")
	}

	ps := &PubSub{
		gce: gce,
	}

	ctx = mrun.WithStartHook(ctx, func(innerCtx context.Context) error {
		ps.ctx = mctx.MergeAnnotations(ps.ctx, ps.gce.Context())
		mlog.Info("connecting to pubsub", ps.ctx)
		var err error
		ps.Client, err = pubsub.NewClient(innerCtx, ps.gce.Project, ps.gce.ClientOptions()...)
		return merr.Wrap(err, ps.ctx)
	})
	ctx = mrun.WithStopHook(ctx, func(context.Context) error {
		return ps.Client.Close()
	})
	ps.ctx = ctx
	return mctx.WithChild(parent, ctx), ps
}

// Topic provides methods around a particular topic in PubSub
type Topic struct {
	ctx   context.Context
	ps    *PubSub
	topic *pubsub.Topic
	name  string
}

// Topic returns, after potentially creating, a topic of the given name
func (ps *PubSub) Topic(ctx context.Context, name string, create bool) (*Topic, error) {
	t := &Topic{
		ctx:  mctx.Annotate(ps.ctx, "topicName", name),
		ps:   ps,
		name: name,
	}

	var err error
	if create {
		t.topic, err = ps.Client.CreateTopic(ctx, name)
		if isErrAlreadyExists(err) {
			t.topic = ps.Client.Topic(name)
		} else if err != nil {
			return nil, merr.Wrap(err, t.ctx, ctx)
		}
	} else {
		t.topic = ps.Client.Topic(name)
		if exists, err := t.topic.Exists(t.ctx); err != nil {
			return nil, merr.Wrap(err, t.ctx, ctx)
		} else if !exists {
			return nil, merr.New("topic dne", t.ctx, ctx)
		}
	}
	return t, nil
}

// Publish publishes a message with the given data as its body to the Topic
func (t *Topic) Publish(ctx context.Context, data []byte) error {
	_, err := t.topic.Publish(ctx, &Message{Data: data}).Get(ctx)
	if err != nil {
		return merr.Wrap(err, t.ctx, ctx)
	}
	return nil
}

// Subscription provides methods around a subscription to a topic in PubSub
type Subscription struct {
	ctx   context.Context
	topic *Topic
	sub   *pubsub.Subscription
	name  string

	// only used in tests to trigger batch processing
	batchTestTrigger chan bool
}

// Subscription returns a Subscription instance, after potentially creating it,
// for the Topic
func (t *Topic) Subscription(ctx context.Context, name string, create bool) (*Subscription, error) {
	name = t.name + "_" + name
	s := &Subscription{
		ctx:   mctx.Annotate(t.ctx, "subName", name),
		topic: t,
		name:  name,
	}

	var err error
	if create {
		s.sub, err = t.ps.CreateSubscription(ctx, name, pubsub.SubscriptionConfig{
			Topic: t.topic,
		})
		if isErrAlreadyExists(err) {
			s.sub = t.ps.Subscription(name)
		} else if err != nil {
			return nil, merr.Wrap(err, s.ctx, ctx)
		}
	} else {
		s.sub = t.ps.Subscription(name)
		if exists, err := s.sub.Exists(ctx); err != nil {
			return nil, merr.Wrap(err, s.ctx, ctx)
		} else if !exists {
			return nil, merr.New("sub dne", s.ctx, ctx)
		}
	}
	return s, nil
}

// ConsumerFunc is a function which messages being consumed will be passed. The
// returned boolean and returned error are independent. If the bool is false the
// message will be returned to the queue for retrying later. If an error is
// returned it will be logged.
//
// The Context will be canceled once the deadline has been reached (as set when
// Consume is called).
type ConsumerFunc func(context.Context, *Message) (bool, error)

// ConsumerOpts are options which effect the behavior of a Consume method call
type ConsumerOpts struct {
	// Default 30s. The timeout each message has to complete before its context
	// is cancelled and the server re-publishes it
	Timeout time.Duration

	// Default 1. Number of concurrent messages to consume at a time
	Concurrent int

	// TODO DisableBatchAutoTrigger
	// Currently there is no auto-trigger behavior, batches only get processed
	// on a dumb ticker. This is necessary for the way I plan to have the
	// datastore writing, but it's not the expected behavior of a batch getting
	// triggered everytime <Concurrent> messages come in.
}

func (co ConsumerOpts) withDefaults() ConsumerOpts {
	if co.Timeout == 0 {
		co.Timeout = 30 * time.Second
	}
	if co.Concurrent == 0 {
		co.Concurrent = 1
	}
	return co
}

// Consume uses the given ConsumerFunc and ConsumerOpts to process messages off
// the Subscription
func (s *Subscription) Consume(ctx context.Context, fn ConsumerFunc, opts ConsumerOpts) {
	opts = opts.withDefaults()
	s.sub.ReceiveSettings.MaxExtension = opts.Timeout
	s.sub.ReceiveSettings.MaxOutstandingMessages = opts.Concurrent

	octx := oldctx.Context(ctx)
	for {
		err := s.sub.Receive(octx, func(octx oldctx.Context, msg *Message) {
			innerOCtx, cancel := oldctx.WithTimeout(octx, opts.Timeout)
			defer cancel()
			innerCtx := context.Context(innerOCtx)

			ok, err := fn(innerCtx, msg)
			if err != nil {
				mlog.Warn("error consuming pubsub message",
					s.ctx, ctx, innerCtx, merr.Context(err))
			}

			if ok {
				msg.Ack()
			} else {
				msg.Nack()
			}
		})
		if octx.Err() == context.Canceled || err == nil {
			return
		} else if err != nil {
			mlog.Warn("error consuming from pubsub",
				s.ctx, ctx, merr.Context(err))
			time.Sleep(1 * time.Second)
		}
	}
}

// BatchConsumerFunc is similar to ConsumerFunc, except it takes in a batch of
// multiple messages at once. If the boolean returned will apply to every
// message in the batch.
type BatchConsumerFunc func(context.Context, []*Message) (bool, error)

// BatchGroupFunc is an optional param to BatchConsume which allows for grouping
// messages into separate groups. Each message received is attempted to be
// placed in a group. Grouping is done by calling this function with the
// received message and a random message from a group, and if this function
// returns true then the received message is placed into that group. If this
// returns false for all groups then a new group is created.
//
// This function should be a pure function.
type BatchGroupFunc func(a, b *Message) bool

// BatchConsume is like Consume, except it groups incoming messages together,
// allowing them to be processed in batches instead of individually.
//
// BatchConsume first collects messages internally for half the
// ConsumerOpts.Timeout value. Once that time has passed it will group all
// messages based on the BatchGroupFunc (if nil then all collected messages form
// one big group). The BatchConsumerFunc is called for each group, with the
// context passed in having a timeout of ConsumerOpts.Timeout/2.
//
// The ConsumerOpts.Concurrent value determines the maximum number of messages
// collected during the first section of the process (before BatchConsumerFn is
// called).
func (s *Subscription) BatchConsume(
	ctx context.Context,
	fn BatchConsumerFunc, gfn BatchGroupFunc,
	opts ConsumerOpts,
) {
	opts = opts.withDefaults()

	type promise struct {
		msg   *Message
		retCh chan bool // must be buffered by one
	}

	var groups [][]promise
	var groupsL sync.Mutex

	groupProm := func(prom promise) {
		groupsL.Lock()
		defer groupsL.Unlock()
		for i := range groups {
			if gfn == nil || gfn(groups[i][0].msg, prom.msg) {
				groups[i] = append(groups[i], prom)
				return
			}
		}
		groups = append(groups, []promise{prom})
	}

	wg := new(sync.WaitGroup)
	defer wg.Wait()

	processGroups := func() {
		groupsL.Lock()
		thisGroups := groups
		groups = nil
		groupsL.Unlock()

		// we do a waitgroup chain so as to properly handle the cancel
		// function. We hold wg (by adding one) until all routines spawned
		// here have finished, and once they have release wg and cancel
		thisCtx, cancel := context.WithTimeout(ctx, opts.Timeout/2)
		thisWG := new(sync.WaitGroup)
		thisWG.Add(1)
		wg.Add(1)
		go func() {
			thisWG.Wait()
			cancel()
			wg.Done()
		}()

		for i := range thisGroups {
			thisGroup := thisGroups[i]
			thisWG.Add(1)
			go func() {
				defer thisWG.Done()
				msgs := make([]*Message, len(thisGroup))
				for i := range thisGroup {
					msgs[i] = thisGroup[i].msg
				}
				ret, err := fn(thisCtx, msgs)
				if err != nil {
					mlog.Warn("error consuming pubsub batch messages",
						s.ctx, merr.Context(err))
				}
				for i := range thisGroup {
					thisGroup[i].retCh <- ret // retCh is buffered
				}
			}()
		}
		thisWG.Done()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(opts.Timeout / 2)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				processGroups()
			case <-s.batchTestTrigger:
				processGroups()
			case <-ctx.Done():
				return
			}
		}
	}()

	s.Consume(ctx, func(ctx context.Context, msg *Message) (bool, error) {
		retCh := make(chan bool, 1)
		groupProm(promise{msg: msg, retCh: retCh})
		select {
		case ret := <-retCh:
			return ret, nil
		case <-ctx.Done():
			return false, merr.New("reading from batch grouping process timed out", s.ctx, ctx)
		}
	}, opts)

}
