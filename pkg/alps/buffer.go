package alps

import (
	"bytes"
	"sync"
)

type safeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

// Write waits to acquire a lock on the buffer before modifying it, so threadsafe access is possible
func (sb *safeBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

// Bytes waits to acquire a lock on the buffer to ensure any writes have already completed
// Returns a copy of the buffer so the returned value is safe to use or modify.
func (sb *safeBuffer) Bytes() []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	tmp := sb.buf.Bytes()
	ret := make([]byte, len(tmp))
	copy(ret, tmp)
	return ret
}
