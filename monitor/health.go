package monitor

import (
	"sync"
)

type Update struct {
	ID    string `json:"id"`
	Color string `json:"color"`
}

var (
	subsMu      sync.Mutex
	subscribers = map[chan Update]struct{}{}
)

func Subscribe() chan Update {
	ch := make(chan Update, 1)
	subsMu.Lock()
	subscribers[ch] = struct{}{}
	subsMu.Unlock()
	return ch
}

func Unsubscribe(ch chan Update) {
	subsMu.Lock()
	delete(subscribers, ch)
	close(ch)
	subsMu.Unlock()
}

func publish(u Update) {
	subsMu.Lock()
	for ch := range subscribers {
		select {
		case ch <- u:
		default:
		}
	}
	subsMu.Unlock()
}

// Start is a placeholder for future health monitoring functionality
func Start(db interface{}) {
	// Tunnel monitoring removed - can be extended for other health checks in the future
}
