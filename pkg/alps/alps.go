package alps

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrNotExist is returned when trying to access a job by an ID which is not found
	ErrNotExist = errors.New("job does not exist")

	// ErrAlreadyStopped is returned when trying to stop a job that is already stopped
	ErrAlreadyStopped = errors.New("job is already stopped")
)

// Hub holds the list of jobs that are running or have run and provides thread safe access to control those jobs
type Hub struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

// NewHub initializes and returns a Hub ready to add jobs
func NewHub() *Hub {
	return &Hub{
		jobs: make(map[string]*Job),
		mu:   sync.RWMutex{},
	}
}

// GetJob returns the Job with the given ID, or an error if the job does not exist.
func (h *Hub) GetJob(id string) (*Job, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	j, ok := h.jobs[id]
	if !ok {
		return nil, ErrNotExist
	}
	return j, nil
}

// AddJob assigns the job a UUIDv4, initializes the job and starts it running, and adds it to the hub for control
// Returns the job ID
func (h *Hub) AddJob(owner, cmd string, args ...string) (*Job, error) {

	j, err := newJob(owner, cmd, args...)
	if err != nil {
		return nil, err
	}

	err = j.start()
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.jobs[j.ID] = j

	return j, nil
}

// StopJob sends a kill signal to the process and waits for the process to exit.
// Context can be provided with a timeout or deadline to prevent getting stuck waiting for a hung process to exit.
// Returns an error if the job does not exist or is already stopped.
func (h *Hub) StopJob(id string, ctx context.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	j, ok := h.jobs[id]
	if !ok {
		return ErrNotExist
	}

	err := j.stop()
	if err != nil {
		return err
	}

	err = j.waitStop(ctx)
	if err != nil {
		return err
	}

	return nil
}
