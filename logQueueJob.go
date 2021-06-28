// Copyright (c) PavedRoad. All rights reserved.
// Licensed under the Apache2. See LICENSE file in the project root
// for full license information.
//
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go"
	"github.com/pavedroad-io/eventbridge/s3"
)

const (
	//LogQueueJobType is type of Job from scheduler
	LogQueueJobType string = "io.pavedraod.eventbridge.logQueueJob"
	//ClientTimeout in seconds to timeout client jobs
	ClientTimeout int = 30
)

type logQueueJob struct {
	ctx       context.Context `json:"ctx"`
	JobID     uuid.UUID       `json:"job_id"`
	Method    string          `json:"method"`
	Payload   []byte          `json:"payload"`
	JobType   string          `json:"job_type"`
	customers s3.Customer
	s3Client  *minio.Client

	// TODO: FIX to errors or custom errors
	jobErrors []string  `json:"jobErrors"`
	JobURL    *url.URL  `json:"job_url"`
	Stats     httpStats `json:"stats"`
}

type logQueueStats struct {
	RequestTimedOut bool
	RequestTime     time.Duration
}

// Process methods
func (j *logQueueJob) ID() string {
	return j.JobID.String()
}

func (j *logQueueJob) Type() string {
	return LogQueueJobType
}

func (j *logQueueJob) Init() error {

	// Generate UUID
	j.JobID = uuid.New()

	// Set job type
	j.JobType = LogQueueJobType

	j.Stats.RequestTimedOut = false

	// Set http client options
	if j.ClientTimeout == 0 {
		j.ClientTimeout = ClientTimeout
	}

	j.client = &http.Client{Timeout: time.Duration(j.ClientTimeout) * time.Second}

	return nil
}

func (j *logQueueJob) Run() (result Result, err error) {
	req, err := http.NewRequest("GET", j.JobURL.String(), nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	start := time.Now()
	resp, err := j.client.Do(req)

	end := time.Now()
	j.Stats.RequestTime = end.Sub(start)

	// client errors are handled with errors.New()
	// so there is no defined set to check for
	if err != nil {
		j.Stats.RequestTimedOut = true
		fmt.Println(err)
		return nil, err
	}

	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	md := j.buildMetadata(resp)
	jrsp := &httpResult{job: j,
		metaData: md,
		payload:  payload}

	return jrsp, nil
}

// buildMetadata returns a map of strings with an http.Response encoded
func (j *logQueueJob) buildMetadata(resp *http.Response) map[string]string {
	md := make(map[string]string)
	md["StatusCode"] = string(rune(resp.StatusCode))
	md["Proto"] = resp.Proto

	for n, v := range resp.Header {
		var hv string
		for _, s := range v {
			hv = hv + s + " "
		}
		md[n] = hv
	}

	md["RemoteAddr"] = resp.Request.RemoteAddr
	md["Method"] = resp.Request.Method

	return md
}

func (j *logQueueJob) newJob(url url.URL) logQueueJob {
	newJob := logQueueJob{}
	pu, err := url.Parse(url.String())
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	newJob.JobURL = pu

	// Set type and ID and http.Client
	newJob.Init()
	return newJob
}

func (j *logQueueJob) Pause() (status string, err error) {
	return "paused", nil
}

func (j *logQueueJob) Shutdown() error {
	return nil
}

func (j *logQueueJob) Errors() []error {
	return nil

}

func (j *logQueueJob) Metrics() []byte {
	jblob, err := json.Marshal(j.Stats)
	if err != nil {
		return []byte("Marshal metrics failed")
	}

	return jblob
}
