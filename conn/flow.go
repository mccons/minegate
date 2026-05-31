package conn

import (
	"sync"
	"sync/atomic"

	"github.com/pozii/minegate/internal"
)

// FlowController provides backpressure and flow control.
// It limits the maximum queue size for OOM protection.
type FlowController struct {
	maxItems   int
	maxBytes   int64
	current    atomic.Int64
	mu         sync.Mutex
	cond       *sync.Cond
}

// NewFlowController creates a new FlowController.
// maxItems: maximum number of packets waiting
// maxBytes: maximum bytes waiting
func NewFlowController(maxItems, maxBytes int) *FlowController {
	fc := &FlowController{
		maxItems: maxItems,
		maxBytes: int64(maxBytes),
	}
	fc.cond = sync.NewCond(&fc.mu)
	return fc
}

// Acquire attempts to allocate space of the given size.
// If the queue is full, it waits or returns an error.
func (fc *FlowController) Acquire(size int) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	for fc.current.Load() >= fc.maxBytes {
		fc.cond.Wait()
	}

	fc.current.Add(int64(size))
	return nil
}

// AcquireNonBlock attempts to allocate space without blocking.
func (fc *FlowController) AcquireNonBlock(size int) error {
	if fc.current.Load()+int64(size) > fc.maxBytes {
		return internal.ErrQueueFull
	}
	fc.current.Add(int64(size))
	return nil
}

// Release frees the allocated space.
func (fc *FlowController) Release(size int) {
	fc.current.Add(-int64(size))
	fc.cond.Signal()
}

// Available returns the available space.
func (fc *FlowController) Available() int64 {
	return fc.maxBytes - fc.current.Load()
}

// Used returns the used space.
func (fc *FlowController) Used() int64 {
	return fc.current.Load()
}

// Utilization returns the fill ratio (0.0 - 1.0).
func (fc *FlowController) Utilization() float64 {
	used := fc.current.Load()
	if used == 0 {
		return 0
	}
	return float64(used) / float64(fc.maxBytes)
}

// DroppableQueue is a queue that can drop packets under overload.
// It is suitable for low-priority packets like chunk data.
type DroppableQueue struct {
	items    []interface{}
	maxSize  int
	mu       sync.Mutex
	dropped  atomic.Int64
}

// NewDroppableQueue creates a new DroppableQueue.
func NewDroppableQueue(maxSize int) *DroppableQueue {
	return &DroppableQueue{
		items:   make([]interface{}, 0, maxSize),
		maxSize: maxSize,
	}
}

// Push adds an item to the queue. Drops the oldest item if the queue is full.
func (dq *DroppableQueue) Push(item interface{}) {
	dq.mu.Lock()
	defer dq.mu.Unlock()

	if len(dq.items) >= dq.maxSize {
		// Drop oldest item
		dq.items = dq.items[1:]
		dq.dropped.Add(1)
	}

	dq.items = append(dq.items, item)
}

// Pop retrieves an item from the queue.
func (dq *DroppableQueue) Pop() (interface{}, bool) {
	dq.mu.Lock()
	defer dq.mu.Unlock()

	if len(dq.items) == 0 {
		return nil, false
	}

	item := dq.items[0]
	dq.items = dq.items[1:]
	return item, true
}

// Len returns the number of items in the queue.
func (dq *DroppableQueue) Len() int {
	dq.mu.Lock()
	defer dq.mu.Unlock()
	return len(dq.items)
}

// Dropped returns the total number of dropped items.
func (dq *DroppableQueue) Dropped() int64 {
	return dq.dropped.Load()
}

// BoundedQueue is a channel-based bounded queue.
// It blocks when capacity is full (backpressure).
type BoundedQueue struct {
	ch chan interface{}
}

// NewBoundedQueue creates a BoundedQueue with the given capacity.
func NewBoundedQueue(capacity int) *BoundedQueue {
	return &BoundedQueue{
		ch: make(chan interface{}, capacity),
	}
}

// Push adds an item to the queue (blocks if capacity is full).
func (bq *BoundedQueue) Push(item interface{}) {
	bq.ch <- item
}

// TryPush attempts to add an item to the queue (non-blocking).
func (bq *BoundedQueue) TryPush(item interface{}) bool {
	select {
	case bq.ch <- item:
		return true
	default:
		return false
	}
}

// Pop retrieves an item from the queue (blocks).
func (bq *BoundedQueue) Pop() interface{} {
	return <-bq.ch
}

// TryPop attempts to retrieve an item from the queue (non-blocking).
func (bq *BoundedQueue) TryPop() (interface{}, bool) {
	select {
	case item := <-bq.ch:
		return item, true
	default:
		return nil, false
	}
}

// Close closes the queue.
func (bq *BoundedQueue) Close() {
	close(bq.ch)
}
