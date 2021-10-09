package chanman

import (
	"context"
	"sync"
)

var logger Logger = NewBuiltinLogger()

// Chanman is a channel queue manager
type Options struct {
	// Limit is the maximum number of items to be queued
	Limit int
	// CallbackFn is the function to call when an item is added to the queue
	CallbackFn func(interface{}) error
}

// Chanman is a channel queue manager
type Chanman struct {
	ctx     context.Context
	opts    *Options
	queueCh chan interface{}
	count   int
	mu      sync.Mutex
}

// New creates a new Chanman instance
func New(ctx context.Context, options *Options) *Chanman {
	return &Chanman{
		ctx:     ctx,
		opts:    options,
		queueCh: make(chan interface{}),
	}
}

// Listen starts listening for queue items
func (cm *Chanman) Listen() {
	defer cm.Quit()

Loop:
	for {
		select {
		case <-cm.ctx.Done():
			break Loop
		case data, ok := <-cm.queueCh:
			if ok {
				cm.opts.CallbackFn(data)
			} else {
				break Loop
			}
		}
	}
}

// Add adds a new item to the queue
func (cm *Chanman) Add(data interface{}) {
	cm.mu.Lock()
	cm.count += 1
	cm.mu.Unlock()

	if isChClosed(cm.queueCh) {
		logger.Errorf("Failed to add item %q to queue. Channel is already closed.", data)
		return
	}

	if cm.isLimitExceeded() {
		logger.Errorf("Failed to add %q. Queue limit (%d) exceeded", data, cm.opts.Limit)
		return
	}

	// Now it is safe to add item to queue
	cm.queueCh <- data
}

// quit closes the queue and quit channels
func (cm *Chanman) Quit() {
	logger.Infof("Closing queue and quit channels")
	cm.closeCh()
}

// isLimitExceeded returns true if the channel limit has been reached
func (cm *Chanman) isLimitExceeded() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.count > cm.opts.Limit
}

// closeCh closes  channel gracefully
func (cm *Chanman) closeCh() {
	if !isChClosed(cm.queueCh) {
		close(cm.queueCh)
	}
}

// isChClosed returns true if a channel is already closed
func isChClosed(c chan interface{}) bool {
	select {
	case <-c:
		return true
	default:
	}
	return false
}
