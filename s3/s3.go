package s3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func s3main() {
	c := Customer{}
	customers, err := c.LoadFromDisk("customer.yaml")
	if err != nil {
		log.Fatalf("fail loading customer.yaml: %v\n", err)

	}

	var eConf Environment
	eConf.Get()

	opts := minio.ListObjectsOptions{
		Recursive: true,
		Prefix:    "",
	}

	// This will be a Job
	var logQueue []LogQueueItem
	var plogs ProcessedLogs
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
			s3Client, err := NewClient(p)
			if err != nil {
				log.Fatalln(err)
			}
			objects, err := ListBucketObjects(s3Client, l.Name, opts)
			if err != nil {
				log.Fatalln(err)
			}

			for _, o := range objects {

				// fmt.Println(o.Key)
				f, err := GetObject(s3Client, l.Name, o.Key, minio.GetObjectOptions{})
				if err != nil {
					log.Fatalln(err)
				}

				if plogs.Processed(l.Name, o.Key) {
					//					fmt.Printf("Skipping %s bucket %s logs\n", l.Name, o.Key)
					continue
				}

				item := LogQueueItem{
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

	// READ from env
	pconf := LogConfig{
		LoadFrom: "network",
		LoadURL:  "",
	}

	for _, l := range logQueue {
		switch l.LogFormat {
		case S3:
			po, err := ParseS3(l.Location)
			if err != nil {
				fmt.Printf("Parse failed with error: %v\n", err)
			}
			for _, eventData := range po {
				j, _ := json.Marshal(eventData)

				postBody := bytes.NewBuffer(j)

				resp, err := http.Post("http://localhost:12001/eventbridge", "application/json", postBody)

				if err != nil {
					log.Printf("HTTP POST failed error %v\n", err)
				}
				if resp.StatusCode != 200 {
					log.Printf("HTTP POST failed non 200 status code %v\n", resp.StatusCode)
				}
			}

			l.Processed = true
			if l.Prune {
				if err := os.Remove(l.Location); err != nil {
					log.Printf("Failed to prune %s error %v\n", l.Location, err)
				}

			}

			nid, err := uuid.Parse(l.ID)
			if err != nil {
				fmt.Printf("Fail converting ID %s to UUID err %v\n", l.ID, err)
			}
			i := ProcessedLogItem{
				Date:     time.Now(),
				Bucket:   l.Bucket,
				Name:     l.Name,
				FileName: l.Location,
				Pruned:   l.Prune,
			}
			plogs.ID = nid
			plogs.AddProcessLog(l.ID, i, pconf)
		}
	}
}
