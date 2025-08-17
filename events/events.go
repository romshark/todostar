package events

import "github.com/romshark/todostar/pkg/broadcast"

type EventTodosChanged struct{}

func (EventTodosChanged) Topic() int64 { return 1 }

var Broadcaster = broadcast.NewTopicBroadcaster()

func NotifyTodosChanged() int {
	return broadcast.Notify(Broadcaster, EventTodosChanged{})
}

func OnTodosChanged(
	callback func(EventTodosChanged),
) broadcast.Subscription[EventTodosChanged] {
	return broadcast.Subscribe(Broadcaster, callback)
}
