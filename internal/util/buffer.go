package util

import (
	"bytes"
	"sync"
)

// SafeBuffer is a goroutine-safe buffer.
type SafeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *SafeBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *SafeBuffer) Read(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Read(p)
}

func (b *SafeBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

func (b *SafeBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Bytes()
}

func (b *SafeBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Reset()
}

func (b *SafeBuffer) Cap() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Cap()
}
