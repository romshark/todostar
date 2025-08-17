package broadcast

import (
	"sync"
	"sync/atomic"
)

var idCounter atomic.Int64

type subscriptionID = int64

// TopicBroadcaster sends events to subscribers of specific topics.
type TopicBroadcaster struct {
	lock   sync.Mutex
	topics map[int64]map[subscriptionID]any
}

type Event interface {
	Topic() int64
}

// NewTopicBroadcaster creates a new topic broadcaster.
func NewTopicBroadcaster() *TopicBroadcaster {
	return &TopicBroadcaster{
		topics: make(map[int64]map[subscriptionID]any),
	}
}

type Subscription[T Event] struct {
	b  *TopicBroadcaster
	id subscriptionID
	c  any
}

// Close unsubscribes and destroys the subscription.
func (s Subscription[T]) Close() {
	s.b.lock.Lock()
	defer s.b.lock.Unlock()

	var ze T
	topic := ze.Topic()

	if subs, ok := s.b.topics[topic]; ok {
		delete(subs, s.id)
		if len(subs) == 0 {
			delete(s.b.topics, topic)
		}
	}
	if c, ok := s.c.(chan T); ok {
		close(c)
	}
}

// Subscribe creates a new subscription for T and calls callback when it's triggered.
func Subscribe[T Event](b *TopicBroadcaster, callback func(T)) Subscription[T] {
	return subscribe[T](b, callback)
}

func subscribe[T Event](b *TopicBroadcaster, c any) Subscription[T] {
	id := idCounter.Add(1)

	b.lock.Lock()
	defer b.lock.Unlock()

	var ze T
	topic := ze.Topic()
	if b.topics[topic] == nil {
		b.topics[topic] = make(map[subscriptionID]any)
	}
	b.topics[topic][id] = c
	return Subscription[T]{b: b, id: id, c: c}
}

// Notify sends event to all subscribers for T.
func Notify[T Event](b *TopicBroadcaster, event T) (notified int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	topic := event.Topic()

	if subs, ok := b.topics[topic]; ok {
		for _, c := range subs {
			switch c := c.(type) {
			case func(T):
				go c(event)
				notified++
			}
		}
	}
	return notified
}
