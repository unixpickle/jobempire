package jobadmin

import "sync"

// A nextNotifier facilitates notifications when a new
// item in a stream of items becomes available.
type nextNotifier struct {
	countLock    sync.Mutex
	closed       bool
	curCount     int
	closeChan    chan struct{}
	nextListener chan struct{}
}

// Wait waits until the stream receives a new item.
// If the stream already has more than c items, it returns
// true immediately.
//
// The cancel channel, if non-nil, can be closed or
// messaged to cancel the wait early.
//
// If the stream is closed or the wait is cancelled before
// a new message is received,
func (n *nextNotifier) Wait(c int, cancel <-chan struct{}) bool {
	n.countLock.Lock()
	if n.closed || n.curCount > c {
		n.countLock.Unlock()
		return !n.closed
	}
	if n.nextListener == nil {
		n.nextListener = make(chan struct{})
	}
	if n.closeChan == nil {
		n.closeChan = make(chan struct{})
	}
	listener := n.nextListener
	n.countLock.Unlock()

	if cancel == nil {
		cancel = make(chan struct{})
	}

	select {
	case <-listener:
		return true
	case <-cancel:
		return false
	case <-n.closeChan:
		// Address times when the notifier is closed
		// right after the last item is sent.
		select {
		case <-listener:
			return true
		default:
			return false
		}
	}
}

// Notify pushes another message to the stream.
// All waiting Goroutines will be notified.
func (n *nextNotifier) Notify() {
	n.countLock.Lock()
	defer n.countLock.Unlock()
	n.curCount++
	if n.nextListener != nil {
		close(n.nextListener)
		n.nextListener = nil
	}
}

// Close closes the stream.
func (n *nextNotifier) Close() {
	n.countLock.Lock()
	defer n.countLock.Unlock()
	if !n.closed {
		n.closed = true
		if n.closeChan != nil {
			close(n.closeChan)
		}
	}
}

// Closed returns whether or not the notifier is currently
// closed.
func (n *nextNotifier) Closed() bool {
	n.countLock.Lock()
	defer n.countLock.Unlock()
	return n.closed
}
