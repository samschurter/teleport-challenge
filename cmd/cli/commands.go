package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

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

var rootCmd = &cobra.Command{
	Use:   "alps",
	Short: "ALPS is an Arbitrary Linux Process Service",
}

func start(c *http.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start a job",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			var job = struct {
				Job  string   `json:"job"`
				Args []string `json:"args"`
			}{
				Job:  args[0],
				Args: args[1:],
			}
			b, err := json.Marshal(job)
			if err != nil {
				log.Fatalf("failed to create job request: %v\n", err)
			}

			jobRes := postRequest(c, "https://localhost:4430/start", b)
			//TODO format this output
			fmt.Printf("response: %+v\n", jobRes)

		},
	}
}

func stop(c *http.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop a running job",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jobRes := postRequest(c, fmt.Sprintf("https://localhost:4430/stop/%s", args[0]), []byte{})
			//TODO format this output
			fmt.Printf("response: %+v\n", jobRes)
		},
	}
}

func status(c *http.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Get the status of a running or completed job",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jobBytes := getRequest(c, fmt.Sprintf("https://localhost:4430/stop/%s", args[0]))
			jobRes := &jobResponse{}
			err := json.Unmarshal(jobBytes, jobRes)
			if err != nil {
				log.Fatalf("failed to unmarshal response: %v\n", err)
			}
			//TODO format this output
			fmt.Printf("response: %+v\n", jobRes)
		},
	}
}

func output(c *http.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "output",
		Short: "Get the stderr and stdout of the job",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			stdOutBytes := getRequest(c, fmt.Sprintf("https://localhost:4430/stdout/%s", args[0]))
			stdErrBytes := getRequest(c, fmt.Sprintf("https://localhost:4430/stderr/%s", args[0]))
			//TODO format this output
			fmt.Printf("Stdout:\n\t%v\n\nStderr:\n\t%v\n\n", stdOutBytes, stdErrBytes)
		},
	}
}

func getRequest(c *http.Client, url string) []byte {
	req, err := http.NewRequest("GET", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		log.Fatalf("could not create request: %v\n", err)
	}
	res, err := c.Do(req)
	if err != nil {
		log.Fatalf("failed to complete request: %v\n", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("failed to read body: %v\n", err)
	}
	if res.Status != "200 OK" {
		showError(body)
		return nil
	}

	return body
}

func postRequest(c *http.Client, url string, data []byte) *jobResponse {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("could not create request: %v\n", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.Do(req)
	if err != nil {
		log.Fatalf("failed to complete request: %v\n", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("failed to read body: %v\n", err)
	}

	if res.Status != "200 OK" {
		showError(body)
		return nil
	}

	jobRes := &jobResponse{}
	err = json.Unmarshal(body, jobRes)
	if err != nil {
		log.Fatalf("failed to unmarshal response: %v\n", err)
	}

	return jobRes
}

func showError(body []byte) {
	errRes := errResponse{}
	err := json.Unmarshal(body, &errRes)
	if err != nil {
		log.Fatalf("failed to unmarshal error response: %v\nbody: %v", err, body)
	}
	//TODO format this output
	log.Printf("error response: %+v\n", errRes)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
