package mredis

import (
	"reflect"
	"sync"
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mtest"

	"github.com/mediocregopher/radix/v3"
)

func TestStream(t *T) {
	cmp := mtest.Component()
	redis := InstRedis(cmp)

	streamKey := "stream-" + mrand.Hex(8)
	group := "group-" + mrand.Hex(8)
	stream := NewStream(redis, StreamOpts{
		Key:           streamKey,
		Group:         group,
		Consumer:      "consumer-" + mrand.Hex(8),
		InitialCursor: "0",
	})

	mtest.Run(cmp, t, func() {
		// once the test is ready to be finished up this will be closed
		finishUpCh := make(chan struct{})

		// continually publish messages, adding them to the expEntries
		t.Log("creating publisher")
		pubDone := make(chan struct{})
		expEntries := map[radix.StreamEntryID]radix.StreamEntry{}
		go func() {
			defer close(pubDone)
			tick := time.NewTicker(50 * time.Millisecond)
			defer tick.Stop()

			for {
				var id radix.StreamEntryID
				key, val := mrand.Hex(8), mrand.Hex(8)
				if err := redis.Do(radix.Cmd(&id, "XADD", streamKey, "*", key, val)); err != nil {
					t.Fatalf("error XADDing: %v", err)
				}
				expEntries[id] = radix.StreamEntry{
					ID:     id,
					Fields: map[string]string{key: val},
				}

				select {
				case <-tick.C:
					continue
				case <-finishUpCh:
					return
				}
			}
		}()

		gotEntriesL := new(sync.Mutex)
		gotEntries := map[radix.StreamEntryID]radix.StreamEntry{}

		// spawn some workers which will process the StreamEntry's. We do this
		// to try and suss out any race conditions with Nack'ing. Each worker
		// will have a random chance of Nack'ing, until finishUpCh is closed and
		// then they will Ack everything.
		t.Log("creating workers")
		const numWorkers = 5
		wg := new(sync.WaitGroup)
		entryCh := make(chan StreamEntry, numWorkers*10)
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for entry := range entryCh {
					select {
					case <-finishUpCh:
					default:
						if mrand.Intn(10) == 0 {
							entry.Nack()
							continue
						}
					}

					if err := entry.Ack(); err != nil {
						t.Fatalf("error calling Ack: %v", err)
					}
					gotEntriesL.Lock()
					gotEntries[entry.ID] = entry.StreamEntry
					gotEntriesL.Unlock()
				}
			}()
		}

		t.Log("consuming...")
		waitTimer := time.After(5 * time.Second)
	loop:
		for {
			select {
			case <-waitTimer:
				break loop
			default:
			}

			entry, ok, err := stream.Next()
			if err != nil {
				t.Fatalf("error calling Next (1): %v", err)
			} else if ok {
				entryCh <- entry
			}
		}

		// after 5 seconds we declare that it's time to finish up
		t.Log("finishing up...")
		close(finishUpCh)
		<-pubDone

		// Keep consuming until all messages have come in, then tell the workers
		// to clean themselves up.
		t.Log("consuming last of the entries")
		for {
			entry, ok, err := stream.Next()
			if err != nil {
				t.Fatalf("error calling Next (2): %v", err)
			} else if ok {
				entryCh <- entry
			} else {
				break // must be empty
			}
		}
		close(entryCh)
		wg.Wait()
		t.Log("all workers cleaned up")

		// call XPENDING to see if anything comes back, nothing should.
		t.Log("checking for leftover pending entries")
		var xpendingRes []interface{}
		err := redis.Do(radix.Cmd(&xpendingRes, "XPENDING", streamKey, group))
		if err != nil {
			t.Fatalf("error calling XPENDING: %v", err)
		} else if numPending := xpendingRes[0].(int64); numPending != 0 {
			t.Fatalf("XPENDING says there's %v pending msgs, there should be 0", numPending)
		}

		if len(expEntries) != len(gotEntries) {
			t.Errorf("len(expEntries):%d != len(gotEntries):%d", len(expEntries), len(gotEntries))
		}

		for id, expEntry := range expEntries {
			gotEntry, ok := gotEntries[id]
			if !ok {
				t.Errorf("did not consume entry %s", id)
			} else if !reflect.DeepEqual(gotEntry, expEntry) {
				t.Errorf("expEntry:%#v != gotEntry:%#v", expEntry, gotEntry)
			}
		}
	})
}
