package mpubsub

import (
	"context"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// this requires the pubsub emulator to be running
func TestPubSub(t *T) {
	cmp := mtest.Component()
	mtest.Env(cmp, "PUBSUB_GCE_PROJECT", "test")
	ps := InstPubSub(cmp)
	mtest.Run(cmp, t, func() {
		topicName := "testTopic_" + mrand.Hex(8)
		ctx := context.Background()

		// Topic shouldn't exist yet
		_, err := ps.Topic(ctx, topicName, false)
		require.Error(t, err)

		// ...so create it
		topic, err := ps.Topic(ctx, topicName, true)
		require.NoError(t, err)

		// Create a subscription and consumer
		sub, err := topic.Subscription(ctx, "testSub", true)
		require.NoError(t, err)

		msgCh := make(chan *Message)
		go sub.Consume(ctx, func(ctx context.Context, m *Message) (bool, error) {
			msgCh <- m
			return true, nil
		}, ConsumerOpts{})
		time.Sleep(1 * time.Second) // give consumer time to actually start

		// publish a message and make sure it gets consumed
		assert.NoError(t, topic.Publish(ctx, []byte("foo")))
		msg := <-msgCh
		assert.Equal(t, []byte("foo"), msg.Data)
	})
}

func TestBatchPubSub(t *T) {
	cmp := mtest.Component()
	mtest.Env(cmp, "PUBSUB_GCE_PROJECT", "test")
	ps := InstPubSub(cmp)
	mtest.Run(cmp, t, func() {
		topicName := "testBatchTopic_" + mrand.Hex(8)
		ctx := context.Background()

		topic, err := ps.Topic(ctx, topicName, true)
		require.NoError(t, err)

		readBatch := func(ch chan []*Message) map[byte]int {
			select {
			case <-time.After(1 * time.Second):
				assert.Fail(t, "waited too long to read batch")
				return nil
			case mm := <-ch:
				ret := map[byte]int{}
				for _, m := range mm {
					ret[m.Data[0]]++
				}
				return ret
			}
		}

		// we use the same sub across the next two sections to ensure that cleanup
		// also works
		sub, err := topic.Subscription(ctx, "testSub", true)
		require.NoError(t, err)
		sub.batchTestTrigger = make(chan bool)

		{ // no grouping
			// Create a subscription and consumer
			ctx, cancel := context.WithCancel(ctx)
			ch := make(chan []*Message)
			go func() {
				sub.BatchConsume(ctx,
					func(ctx context.Context, mm []*Message) (bool, error) {
						ch <- mm
						return true, nil
					},
					nil,
					ConsumerOpts{Concurrent: 5},
				)
				close(ch)
			}()
			time.Sleep(1 * time.Second) // give consumer time to actually start

			exp := map[byte]int{}
			for i := byte(0); i <= 9; i++ {
				require.NoError(t, topic.Publish(ctx, []byte{i}))
				exp[i] = 1
			}

			time.Sleep(1 * time.Second)
			sub.batchTestTrigger <- true
			gotA := readBatch(ch)
			assert.Len(t, gotA, 5)

			time.Sleep(1 * time.Second)
			sub.batchTestTrigger <- true
			gotB := readBatch(ch)
			assert.Len(t, gotB, 5)

			for i, c := range gotB {
				gotA[i] += c
			}
			assert.Equal(t, exp, gotA)

			time.Sleep(1 * time.Second) // give time to ack before cancelling
			cancel()
			<-ch
		}

		{ // with grouping
			ctx, cancel := context.WithCancel(ctx)
			ch := make(chan []*Message)
			go func() {
				sub.BatchConsume(ctx,
					func(ctx context.Context, mm []*Message) (bool, error) {
						ch <- mm
						return true, nil
					},
					func(a, b *Message) bool { return a.Data[0]%2 == b.Data[0]%2 },
					ConsumerOpts{Concurrent: 10},
				)
				close(ch)
			}()
			time.Sleep(1 * time.Second) // give consumer time to actually start

			exp := map[byte]int{}
			for i := byte(0); i <= 9; i++ {
				require.NoError(t, topic.Publish(ctx, []byte{i}))
				exp[i] = 1
			}

			time.Sleep(1 * time.Second)
			sub.batchTestTrigger <- true
			gotA := readBatch(ch)
			assert.Len(t, gotA, 5)
			gotB := readBatch(ch)
			assert.Len(t, gotB, 5)

			assertGotGrouped := func(got map[byte]int) {
				prev := byte(255)
				for i := range got {
					if prev != 255 {
						assert.Equal(t, prev%2, i%2)
					}
					prev = i
				}
			}

			assertGotGrouped(gotA)
			assertGotGrouped(gotB)
			for i, c := range gotB {
				gotA[i] += c
			}
			assert.Equal(t, exp, gotA)

			time.Sleep(1 * time.Second) // give time to ack before cancelling
			cancel()
			<-ch
		}
	})
}
