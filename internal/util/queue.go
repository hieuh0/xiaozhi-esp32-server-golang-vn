package util

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrQueueClosed = errors.New("queue closed or cleared")
var ErrQueueTimeout = errors.New("queue pop timeout")
var ErrQueueEmpty = errors.New("queue empty (non-blocking pop)")
var ErrQueueCtxDone = errors.New("queue ctx done")

// Queue is a generic, thread-safe queue based on chan.
type Queue[T any] struct {
	mu     sync.Mutex
	ch     chan T
	cap    int
	closed bool
	gen    uint64
}

// NewQueue creates a new Queue with the given capacity.
func NewQueue[T any](capacity int) *Queue[T] {
	return &Queue[T]{
		ch:  make(chan T, capacity),
		cap: capacity,
	}
}

func safeQueueSendWithTimeout[T any](ch chan T, val T, timeout time.Duration) (sent bool, closed bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	defer func() {
		if recover() != nil {
			sent = false
			closed = true
		}
	}()

	select {
	case ch <- val:
		return true, false
	case <-timer.C:
		return false, false
	}
}

// Push adds an item to the queue. Returns error if queue is closed.
func (q *Queue[T]) Push(val T) error {
	for {
		q.mu.Lock()
		if q.closed {
			q.mu.Unlock()
			return ErrQueueClosed
		}
		ch := q.ch
		gen := q.gen
		q.mu.Unlock()

		sent, closed := safeQueueSendWithTimeout(ch, val, 10*time.Second)
		if sent {
			return nil
		}
		if !closed {
			return errors.New("push timeout (10s)")
		}

		q.mu.Lock()
		queueClosed := q.closed
		shouldRetry := !queueClosed && q.gen != gen
		q.mu.Unlock()

		if queueClosed {
			return ErrQueueClosed
		}
		if shouldRetry {
			continue
		}
		return ErrQueueClosed
	}
}

// Pop tries to get an item from the queue.
// ctx: supports cancellation; returns immediately when ctx.Done() fires
// timeout=0: block until item arrives or queue is cleared
// timeout<0: non-blocking
// timeout>0: wait up to timeout duration
func (q *Queue[T]) Pop(ctx context.Context, timeout time.Duration) (T, error) {
	var zero T
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return zero, ErrQueueClosed
	}
	ch := q.ch
	q.mu.Unlock()

	if timeout < 0 {
		// Non-blocking
		select {
		case v, ok := <-ch:
			if !ok {
				return zero, ErrQueueClosed
			}
			return v, nil
		default:
			return zero, ErrQueueEmpty
		}
	} else if timeout == 0 {
		// Blocking, supports ctx.Done()
		select {
		case v, ok := <-ch:
			if !ok {
				return zero, ErrQueueClosed
			}
			return v, nil
		case <-ctx.Done():
			return zero, ErrQueueCtxDone
		}
	} else {
		// Timeout, supports ctx.Done()
		select {
		case v, ok := <-ch:
			if !ok {
				return zero, ErrQueueClosed
			}
			return v, nil
		case <-time.After(timeout):
			return zero, ErrQueueTimeout
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}
}

// ClearAndDrain empties the queue, returns the items drained from the swapped-out queue,
// and ensures all Pop calls on the old queue return immediately.
func (q *Queue[T]) ClearAndDrain() []T {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return nil
	}
	oldCh := q.ch
	q.ch = make(chan T, q.cap)
	q.gen++
	close(oldCh)
	q.mu.Unlock()

	drained := make([]T, 0, len(oldCh))
	for item := range oldCh {
		drained = append(drained, item)
	}
	return drained
}

// Clear empties the queue and ensures all Pop calls return immediately.
func (q *Queue[T]) Clear() {
	_ = q.ClearAndDrain()
}

// Close closes the queue permanently. All Push/Pop will error after this.
func (q *Queue[T]) Close() {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		q.gen++
		close(q.ch)
	}
	q.mu.Unlock()
}
