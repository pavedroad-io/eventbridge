// Copyright (c) PavedRoad. All rights reserved.
// Licensed under the Apache2. See LICENSE file in the project root
// for full license information.
//
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/pavedroad-io/eventbridge/s3"
)

//_ "eventbridge/s3"
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
	//customers string
	s3Client *minio.Client

	// TODO: FIX to errors or custom errors
	jobErrors []string      `json:"jobErrors"`
	JobURL    *url.URL      `json:"job_url"`
	Stats     logQueueStats `json:"stats"`
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

	return nil
}

func (j *logQueueJob) Run() (result Result, err error) {
	c := s3.Customer{}
	customers, err := c.LoadFromDisk("customer.yaml")
	if err != nil {
		log.Fatalf("fail loading customer.yaml: %v\n", err)

	}
	opts := minio.ListObjectsOptions{
		Recursive: true,
		Prefix:    "",
	}

	var logQueue []s3.LogQueueItem
	var plogs s3.ProcessedLogs
	for _, c := range customers {
		// Load a list of previously processed logs
		// For now ignore error if not found
		plogs.LoadFromDisk(c.ID.String())

		// Build a list of providers the customer
		// uses
		plist := c.Providers

		for i, l := range c.Logs {
			p, err := plist.Lookup(l.Provider)
			if err != nil {
				log.Printf("Provider not found: %v\n", err)
			}
			s3Client, err := s3.NewClient(p)
			if err != nil {
				log.Fatalln(err)
			}
			objects, err := s3.ListBucketObjects(s3Client, l.Name, opts)
			if err != nil {
				log.Fatalln(err)
			}

			for _, o := range objects {

				// fmt.Println(o.Key)
				f, err := s3.GetObject(s3Client, l.Name, o.Key, minio.GetObjectOptions{})
				if err != nil {
					log.Fatalln(err)
				}

				if plogs.Processed(l.Name, o.Key) {
					//                  fmt.Printf("Skipping %s bucket %s logs\n", l.Name, o.Key)
					continue
				}

				item := s3.LogQueueItem{
					ID:        c.ID.String(),
					Bucket:    l.Name,
					Name:      o.Key,
					Created:   time.Now(),
					Location:  f,
					LogFormat: c.Logs[i].LogFormat,
					Processed: false,
					Prune:     c.Logs[i].PruneAfterProcessing,
				}
				logQueue = append(logQueue, item)
			}
		}
	}

	return nil, nil
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
