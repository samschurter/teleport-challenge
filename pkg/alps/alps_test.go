package alps_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/samschurter/teleport-challenge/pkg/alps"
)

func TestHub(t *testing.T) {
	hub := alps.NewHub()
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("run # %d", i), func(t *testing.T) {
			t.Parallel()
			t.Run("testAddJob", testAddJob(t, hub))
			t.Run("testGetJob", testGetJob(t, hub))
			t.Run("testStopJob", testStopJob(t, hub))
			t.Run("testOutput", testOutput(t, hub))
		})
	}
}

// Simple AddJob test, does not test various combinations of commands that may throw errors
func testAddJob(t *testing.T, hub *alps.Hub) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		before := time.Now()
		job, err := hub.AddJob("sam", "ls")
		if err != nil {
			t.Fatalf("hub.AddJob() err = %v; want %v", err, nil)
		}
		if job.Status() != alps.Running {
			t.Errorf("job.Status = %v; want %v", job.Status(), alps.Running)
		}
		after := time.Now()
		start := job.StartTime()
		if start.Before(before) || start.After(after) {
			t.Errorf("job.StartTime() = %v; want after %v and before %v", start, before, after)
		}
	}
}

func testGetJob(t *testing.T, hub *alps.Hub) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		job, _ := hub.AddJob("sam", "sleep", "1")
		_, err := hub.GetJob(job.ID)
		if err != nil {
			t.Errorf("hub.GetJob(id) err = %v; want nil", err)
		}

		_, err = hub.GetJob("junk")
		if !errors.Is(err, alps.ErrNotExist) {
			t.Errorf("hub.GetJob(junk) err = %v; want %v", err, alps.ErrNotExist)
		}
	}
}

func testStopJob(t *testing.T, hub *alps.Hub) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		job, _ := hub.AddJob("sam", "sleep", "1")
		// test that a job stops properly
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
		defer cancel()
		err := hub.StopJob(job.ID, ctx)
		if err != nil {
			t.Fatalf("hub.StopJob() err = %v; want %v", err, nil)
		}
		if job.Status() != alps.Stopped {
			t.Errorf("job.Status() = %v; want %v", job.Status(), alps.Stopped)
		}
		if job.ExitCode() != -1 {
			t.Errorf("job.ExitCode() = %d; want %d", job.ExitCode(), -1)
		}
		if job.StopTime().IsZero() {
			t.Errorf("job.StopTime() = %v; want not zero", job.StopTime())
		}

		// test that a job throws an error if it's already stopped
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*500)
		defer cancel()
		err = hub.StopJob(job.ID, ctx)
		if !errors.Is(err, alps.ErrAlreadyStopped) {
			t.Errorf("err = %v; want %v", err, alps.ErrAlreadyStopped)
		}
	}
}

func testOutput(t *testing.T, hub *alps.Hub) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		job, _ := hub.AddJob("sam", "sleep", "foo")
		time.Sleep(10 * time.Millisecond) // make sure the job has time to complete
		stderr := string(job.StdErr())
		if len(stderr) == 0 {
			t.Errorf("job.Stderr() len = 0; want > 0")
		}

		output := "hello world!"
		job, _ = hub.AddJob("sam", "echo", output)
		time.Sleep(10 * time.Millisecond) // make sure the job has time to complete and write output
		stdout := strings.TrimSpace(string(job.StdOut()))
		if stdout != output {
			t.Errorf("job.Stdout() = %q; want %q", stdout, output)
		}
	}
}
