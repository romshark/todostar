package broadcast_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/romshark/todostar/pkg/broadcast"

	"github.com/stretchr/testify/require"
)

type TestEvent1 struct{ Data int }

func (TestEvent1) Topic() int64 { return 1 }

type TestEvent2 struct{ Data string }

func (TestEvent2) Topic() int64 { return 2 }

func TestSubUnsub(t *testing.T) {
	b := broadcast.NewTopicBroadcaster()

	var wg sync.WaitGroup
	var sub1Calls, sub2Calls atomic.Int32

	sub1 := broadcast.Subscribe(b, func(TestEvent1) {
		defer wg.Done()
		sub1Calls.Add(1)
	})
	sub2 := broadcast.Subscribe(b, func(TestEvent1) {
		defer wg.Done()
		sub2Calls.Add(1)
	})

	notified := broadcast.Notify(b, TestEvent2{})
	require.Zero(t, notified, "no subscribers for TestEvent2")
	require.Zero(t, sub1Calls.Load())
	require.Zero(t, sub2Calls.Load())

	wg.Add(2)
	notified = broadcast.Notify(b, TestEvent1{Data: 42})
	require.Equal(t, 2, notified)
	wg.Wait()

	require.Equal(t, int32(1), sub1Calls.Load())
	require.Equal(t, int32(1), sub2Calls.Load())

	sub1.Close()
	sub2.Close()

	notified = broadcast.Notify(b, TestEvent2{}) // No-op
	require.Zero(t, notified, "all subscribers unsubscribed")
	require.Equal(t, int32(1), sub1Calls.Load())
	require.Equal(t, int32(1), sub2Calls.Load())
}
