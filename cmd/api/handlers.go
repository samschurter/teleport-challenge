package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	gcontext "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/samschurter/teleport-challenge/pkg/alps"
)

type jobServer struct {
	hub *alps.Hub
}

type startRequest struct {
	Job  string   `json:"job"`
	Args []string `json:"args"`
}

type errResponse struct {
	Status int    `json:"status"`
	Error  string `json:"error"`
}

type jobResponse struct {
	Status     int       `json:"status,omitempty"`
	JobID      string    `json:"job_id,omitempty"`
	Job        string    `json:"job,omitempty"`
	Owner      string    `json:"owner,omitempty"`
	JobStatus  string    `json:"job_status,omitempty"`
	StartTime  time.Time `json:"start_time,omitempty"`
	StopTime   time.Time `json:"stop_time,omitempty"`
	ExitCode   int       `json:"exit_code,omitempty"`
	StdoutSize int       `json:"stdout_size,omitempty"`
	StderrSize int       `json:"stderr_size,omitempty"`
}

func httpError(w http.ResponseWriter, msg string, code int) {
	b, err := json.Marshal(errResponse{Status: code, Error: msg})
	if err != nil {
		http.Error(w, fmt.Errorf("error marshalling JSON error response: %w", err).Error(), http.StatusInternalServerError)
	}
	http.Error(w, string(b), code)
}

func (s *jobServer) start(w http.ResponseWriter, r *http.Request) {
	var sr startRequest
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := gcontext.Get(r, "user").(string)

	job, err := s.hub.AddJob(user, sr.Job, sr.Args...)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res := jobResponse{
		Status:    200,
		JobID:     job.ID,
		Owner:     job.Owner,
		StartTime: job.StartTime(),
	}

	writeResponse(w, res)
}

func (s *jobServer) stop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		httpError(w, "no id provided", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := s.hub.StopJob(id, ctx)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	job, err := s.hub.GetJob(id)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res := jobResponse{
		Status:    200,
		JobID:     job.ID,
		Owner:     job.Owner,
		StartTime: job.StartTime(),
		StopTime:  job.StopTime(),
	}

	writeResponse(w, res)
}

func (s *jobServer) status(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		httpError(w, "no id provided", http.StatusInternalServerError)
		return
	}

	job, err := s.hub.GetJob(id)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ec, _ := job.ExitCode()

	res := jobResponse{
		Status:     200,
		JobID:      job.ID,
		Job:        job.CommandString(),
		Owner:      job.Owner,
		JobStatus:  string(job.Status()),
		StartTime:  job.StartTime(),
		StopTime:   job.StopTime(),
		ExitCode:   ec,
		StdoutSize: len(job.StdOut()),
		StderrSize: len(job.StdErr()),
	}

	writeResponse(w, res)
}

func (s *jobServer) stdout(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		httpError(w, "no id provided", http.StatusInternalServerError)
		return
	}

	job, err := s.hub.GetJob(id)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(job.StdOut())
}

func (s *jobServer) stderr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		httpError(w, "no id provided", http.StatusInternalServerError)
		return
	}

	job, err := s.hub.GetJob(id)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(job.StdErr())
}

func writeResponse(w http.ResponseWriter, res jobResponse) {
	resBytes, err := json.Marshal(res)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resBytes)
}
