package alps

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	Stopped JobStatus = "stopped"
	Running JobStatus = "running"
)

// Job represents a running or stopped job and all related details
type Job struct {
	// Owner is any string, used to identify who started the job
	Owner string

	// ID is the UUIDv4 identifying the job. Use this to retrieve the job from the Hub.
	ID string

	// StartTime is set as soon as the job is started.
	//
	// StopTime will be zero until the job has stopped.
	startTime time.Time
	stopTime  time.Time

	// ExitCode will be available after the job has stopped.
	exitCode *int

	// Status will be available as soon as the job is started
	status JobStatus

	// stdErr and StdOut will contain the redirected stdErr and Stdout from the job
	stdErr *safeBuffer
	stdOut *safeBuffer

	command string
	args    []string
	mu      sync.RWMutex
	cmd     *exec.Cmd
	done    chan bool
}

// CommandString returns the entire command with arguments used to start the job
func (j *Job) CommandString() string {
	return j.command + strings.Join(j.args, " ")
}

func (j *Job) Status() JobStatus {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.status
}

func (j *Job) StartTime() time.Time {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.startTime
}

func (j *Job) StopTime() time.Time {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.stopTime
}

// ExitCode returns the job's exit code and OK if it's available, or -1 and false if it is not.
// returns -1 even if the exit code is not available to match the behavior of (*Cmd).ProcessState.ExitCode()
func (j *Job) ExitCode() (int, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	if j.exitCode == nil {
		return -1, false
	}
	return *j.exitCode, true
}

func (j *Job) StdOut() []byte {
	return j.stdOut.Bytes()
}

func (j *Job) StdErr() []byte {
	return j.stdErr.Bytes()
}

func newJob(owner, cmd string, args ...string) (*Job, error) {

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	j := &Job{
		Owner:   owner,
		ID:      id.String(),
		command: cmd,
		args:    args,
		mu:      sync.RWMutex{},
		done:    make(chan bool, 1),
	}

	return j, nil
}

func (j *Job) start() error {

	j.mu.Lock()
	defer j.mu.Unlock()

	j.cmd = exec.Command(j.command, j.args...)

	j.stdErr = &safeBuffer{}
	j.cmd.Stderr = j.stdErr

	j.stdOut = &safeBuffer{}
	j.cmd.Stdout = j.stdOut

	j.startTime = time.Now()

	j.status = Running

	err := j.cmd.Start()
	if err != nil {
		return err
	}

	go j.wait()

	return nil
}

func (j *Job) wait() {
	// ignore the error as we don't really care why the process exited. the ProcessState will have the exitCode either way
	j.cmd.Wait()
	// as soon as the process has exited, acquire a lock before proceeding
	j.mu.Lock()
	defer j.mu.Unlock()
	ec := j.cmd.ProcessState.ExitCode()
	j.exitCode = &ec
	j.status = Stopped
	j.stopTime = time.Now()
	j.done <- true
}

func (j *Job) stop() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.status == Stopped {
		return ErrAlreadyStopped
	}

	// there is a slight chance the process will have exited on it's own between checking process state and calling kill
	// in that case, stop() will return an error.
	var errFinished = errors.New("os: process already finished")
	err := j.cmd.Process.Kill()
	if errors.As(err, &errFinished) {
		return ErrAlreadyStopped
	} else if err != nil {
		return err
	}
	return nil
}

func (j *Job) waitStop(ctx context.Context) error {
	for {
		select {
		case <-j.done:
			return nil
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for process to exit")
		}
	}
}
