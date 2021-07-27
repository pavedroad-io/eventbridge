// Copyright (c) PavedRoad. All rights reserved.
// Licensed under the Apache2. See LICENSE file in the project root
// for full license information.
//
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pavedroad-io/eventbridge/s3"
)

const (
	//LogProcessorJobType is type of Job from scheduler
	LogProcessorJobType string = "io.pavedraod.eventbridge.logprocessorjob"

	//ClientTimeout in seconds to timeout client jobs
	ClientTimeout int = 30
)

// logProcessorJob type for dispatcher to run
type logProcessorJob struct {
	ctx           context.Context `json:"ctx"`
	JobID         uuid.UUID       `json:"job_id"`
	JobType       string          `json:"job_type"`
	client        *http.Client    `json:"client"`
	ClientTimeout int             `json:"client_timeout"`
	Log           s3.LogQueueItem
	// TODO: FIX to errors or custom errors
	jobErrors []string  `json:"jobErrors"`
	JobURL    *url.URL  `json:"job_url"`
	Stats     httpStats `json:"stats"`
}

type httpStats struct {
	RequestTimedOut bool
	RequestTime     time.Duration
}

// Process methods
func (j *logProcessorJob) ID() string {
	return j.JobID.String()
}

func (j *logProcessorJob) Type() string {
	return LogProcessorJobType
}

func (j *logProcessorJob) InitWithJobChan(job chan Job) error {

	return j.Init()
}

func (j *logProcessorJob) Init() error {

	// Generate UUID
	j.JobID = uuid.New()

	// Set job type
	j.JobType = LogProcessorJobType

	j.Stats.RequestTimedOut = false

	// Set http client options
	if j.ClientTimeout == 0 {
		j.ClientTimeout = ClientTimeout
	}

	j.client = &http.Client{Timeout: time.Duration(j.ClientTimeout) * time.Second}

	return nil
}

func (j *logProcessorJob) Run() (result Result, err error) {

	var plogs s3.ProcessedLogs // Tracks logs we've already seen
	_log := j.Log

	switch _log.LogFormat {
	case s3.S3:
		loglines, err := s3.ParseS3(_log.Location)
		if err != nil {
			fmt.Printf("Parse failed with error: %v\n", err)
		}

		filter := _log.Filter

		for _, eventData := range loglines {
			// Parse operation field and skip if filer doesn't match
			opt := eventData.GetOperation()
			if !opt.FilterLine(eventData, filter) {
				continue
			}
			eventBytes, _ := json.Marshal(eventData)
			postBody := bytes.NewBuffer(eventBytes)

			resp, err := http.Post(
				"http://"+
					_log.Webhook.Host+":"+
					_log.Webhook.Port+
					"/"+_log.Webhook.Name,
				"application/json",
				postBody)

			if err != nil {
				log.Printf("HTTP POST failed error %v\n", err)
				jrsp := &logResult{}
				return jrsp.LogErrorResults(j, err)
			}
			if resp.StatusCode != 200 {
				err := fmt.Errorf("HTTP POST failed non 200%v\n", resp.StatusCode)
				jrsp := &logResult{}
				return jrsp.LogErrorResults(j, err)
			}
		}

		_log.Processed = true
		if _log.Prune {
			if err := os.Remove(_log.Location); err != nil {
				log.Printf("Failed to prune %s error %v\n", _log.Location, err)
			}

		}

		nid, err := uuid.Parse(_log.ID)
		if err != nil {
			fmt.Printf("Fail converting ID %s to UUID err %v\n", _log.ID, err)
		}
		pli := s3.ProcessedLogItem{
			Date:     time.Now(),
			Bucket:   _log.Bucket,
			Name:     _log.Name,
			FileName: _log.Location,
			Pruned:   _log.Prune,
		}
		plogs.ID = nid
		plogs.AddProcessLog(_log.ID, pli)
	}

	// To avoid casting, convert Job to JSON
	// and decode base on type via -> result.Decode()
	jd, err := json.Marshal(j)
	if err != nil {
		fmt.Println("Marshal result for job failed: ", jd)
	}

	jrsp := &logResult{job: jd,
		payload: nil,
		jobType: j.JobType}

	return jrsp, nil

	//	return nil, nil
}

// buildMetadata returns a map of strings with an http.Response encoded
func (j *logProcessorJob) buildMetadata(resp *http.Response) map[string]string {
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

func (j *logProcessorJob) newJob(url url.URL) logProcessorJob {
	newJob := logProcessorJob{}
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

func (j *logProcessorJob) Pause() (status string, err error) {
	return "paused", nil
}

func (j *logProcessorJob) Shutdown() error {
	return nil
}

func (j *logProcessorJob) Errors() []error {
	return nil

}

func (j *logProcessorJob) Metrics() []byte {
	jblob, err := json.Marshal(j.Stats)
	if err != nil {
		return []byte("Marshal metrics failed")
	}

	return jblob
}
