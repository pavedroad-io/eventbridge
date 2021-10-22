// Copyright (c) PavedRoad. All rights reserved.
// Licensed under the Apache2. See LICENSE file in the project root
// for full license information.
//
package main

import (
	"encoding/json"
	"fmt"

	"log"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/pavedroad-io/eventbridge/s3"
	"github.com/pavedroad-io/go-core/logger"
)

const (
	//LogQueueJobType is type of Job from scheduler
	LogQueueJobType string = "io.pavedraod.eventbridge.logQueueJob"
)

type logQueueJob struct {
	JobID            uuid.UUID     `json:"jobID"`
	Payload          []byte        `json:"payload"`
	JobType          string        `json:"jobType"`
	customers        s3.Customer   `json:"customers"`
	s3Client         *minio.Client `json:"s3Client"`
	schedulerJobChan chan Job      `json:"schedulerJobChan"`

	// TODO: FIX to errors or custom errors
	jobErrors []string      `json:"jobErrors"`
	Stats     logQueueStats `json:"stats"`
}

type logQueueStats struct {
	RequestTimedOut  bool
	RequestStartTime time.Time
	RequestTime      time.Duration
}

// Process methods
func (j *logQueueJob) ID() string {
	return j.JobID.String()
}

func (j *logQueueJob) Type() string {
	return LogQueueJobType
}

func (j *logQueueJob) InitWithJobChan(job chan Job) error {
	j.schedulerJobChan = job
	return j.Init()
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
	var customers []s3.Customer
	var eConf Environment
	eConf.get()

	if eConf.LoadFrom == "disk" {
		customers, err = c.LoadFromDisk("customer.yaml")
		if err != nil {
			log.Fatalf("fail loading customer.yaml: %v\n", err)
		}
	} else {
		// Load from network
		customers, err = c.LoadFromNetwork(eConf.EventBridgeConfigURL)
		if err != nil {
			log.Fatalf("fail loading customer.yaml: %v\n", err)
		}
		fmt.Printf("Found %d customers\n", len(customers))
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
		pconf := s3.LogConfig{
			LoadFrom:     eConf.LoadFrom,
			LoadURL:      eConf.EventBridgePlogsURL,
			CustID:       c.ID.String(),
			PlogConfigID: c.Configuration.PlogConfigID,
		}

		if err := plogs.Load(pconf); err != nil {
			log.Printf("Failed to load past processed logs: %v\n", err)
			fmt.Printf("Failed to load past processed logs: %v\n", err)
		}

		// Build a list of providers the customer
		// uses
		plist := c.Providers

		// Actually buckets not logs
		for i, l := range c.Logs {
			p, err := plist.Lookup(l.Provider)
			if err != nil {
				log.Printf("Provider not found: %v\n", err)
				continue
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

				// See it is already processed
				//
				if plogs.Processed(l.Name, o.Key) {
					continue
				}

				/// create new stats object
				j.Stats.RequestStartTime = time.Now()
				f, err := s3.GetObject(s3Client, l.Name, o.Key, minio.GetObjectOptions{})
				if err != nil {
					j.Stats.RequestTimedOut = true
					log.Fatalln(err)
				}

				c.Configuration.Hook.Host = eConf.EventBridgePostHost

				item := s3.LogQueueItem{
					ID:           c.ID.String(),
					Bucket:       l.Name,
					Webhook:      c.Configuration.Hook,
					Filter:       l.FilterEvents,
					Name:         o.Key,
					Created:      time.Now(),
					Location:     f,
					LogFormat:    c.Logs[i].LogFormat,
					Processed:    false,
					PlogConfigID: c.Configuration.PlogConfigID,
					Prune:        c.Logs[i].PruneAfterProcessing,
				}

				// send item to kafka
				jitem, _ := json.Marshal(item)
				logger.Printf(string(jitem))

				// Write new Job to dispatcher Job
				// Channel
				nj := &logProcessorJob{}
				nj.Init()
				nj.Log = item

				j.Stats.RequestTime = time.Now().Sub(j.Stats.RequestStartTime)
				j.schedulerJobChan <- nj

				logQueue = append(logQueue, item)
			}
		}

		/*
			if eConf.LoadFrom == s3.NETWORK {
				fmt.Println("plots save: ", plogs)
				if err := plogs.SaveToNetwork(pconf); err != nil {
					return nil, err
				}

			}
		*/
	}

	payload, err := json.Marshal(logQueue)
	if err != nil {
		fmt.Errorf("Error %v\n", err)
		return nil, err
	}

	//		metaData: md,
	jd, err := json.Marshal(j)
	if err != nil {
		fmt.Println(err)
	}
	jrsp := &logResult{job: jd,
		jobType: j.Type(),
		payload: payload}

	return jrsp, nil
}

func (j *logQueueJob) newJob(url url.URL) logQueueJob {
	newJob := logQueueJob{}
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
