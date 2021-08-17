package s3

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type LogFormat string

func (lf *LogFormat) String() string {
	return lf.String()
}

const (
	W3C   string = "w3c"
	S3    string = "s3"
	rStor string = "w3c"
)

// LogBuckets Information on a bucket to monitor
type LogBuckets struct {

	// Name of the bucket
	Name string `yaml:"name"`

	// LogFormat S3, w3c, etc
	LogFormat string `yaml:"logFormat"`

	// Provider credentials
	Provider string `yaml:"provider"`

	// PruneAfterProcessing aka delete when donw
	PruneAfterProcessing bool `yaml:"pruneAfterProcessing"`

	// FilterEvents to apply to the log
	FilterEvents S3Filter `yaml:"filter"`
}

type LogQueue []LogQueueItem

type LogQueueItem struct {
	ID        string        `json:"id"`
	Bucket    string        `json:"bucket"`
	Webhook   WebHookConfig `json:"webhook"`
	Filter    S3Filter      `json:"filter"`
	Name      string        `json:"name"`
	Created   time.Time     `json:"created"`
	Location  string        `json:"location"`
	LogFormat string        `json:"logFormat"`
	Processed bool          `json:"processed"`
	Prune     bool          `json:"prune"`
}

// ProcessedLogs for a given customer
type ProcessedLogs struct {
	// ID customer ID
	ID uuid.UUID `json:"id"`
	// Processed list of processed logs
	ProcessedItems []ProcessedLogItem `json:"processedItems"`
}

// ProcessedLogItem information on a processed log
type ProcessedLogItem struct {
	// Date log was processed
	Date time.Time `json:"date"`

	// Name of bucket containing the log
	Bucket string `json:"bucket"`

	// Name of log in the bucket
	Name string `json:"name"`

	// FileName of downloaded log
	FileName string `json:"fileName"`

	// Pruned if log was delete, FileName, after processing
	Pruned bool `json:"pruned"`
}

// Processed returns true if we've processed this log before
//   false if we have not
func (pls *ProcessedLogs) Processed(bucket, name string) bool {
	for _, l := range pls.ProcessedItems {
		if l.Bucket == bucket && l.Name == name {
			return true
		}
	}
	return false
}

func (pls *ProcessedLogs) LoadFromDisk(ID string) error {
	file := ID + "processed.yaml"

	pl := ProcessedLogs{}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		//msg := fmt.Errorf("File not found: [%s]\n", file)
		//return msg
		return nil
	}

	f, err := os.Open(file)
	if err != nil {
		fmt.Println("failed to open:", file, ", error:", err)
	}
	defer f.Close()

	byteValue, e := ioutil.ReadAll(f)
	if e != nil {
		fmt.Println("read failed for ", file)
		return err
	}

	err = yaml.Unmarshal([]byte(byteValue), &pl)
	if err != nil {
		fmt.Println("Unmarshal failed", err)
		return err
	}

	nid, err := uuid.Parse(ID)
	if err != nil {
		fmt.Errorf("Fail converting ID %s to UUID err %w\n", ID, err)
		return err
	}
	pls.ID = nid
	pls.ProcessedItems = pl.ProcessedItems

	return nil
}

func (pls *ProcessedLogs) SaveToDisk(ID string) error {
	file := ID + "processed.yaml"

	yb, err := yaml.Marshal(pls)
	if err != nil {
		fmt.Println("Marshal faild", err)
		return err
	}

	err = ioutil.WriteFile(file, yb, fs.ModePerm)
	if err != nil {
		fmt.Errorf("write failed for %s error %w\n",
			file, err)
		return err
	}

	return nil
}

func (pls *ProcessedLogs) AddProcessLog(ID string, log ProcessedLogItem) error {

	if err := pls.LoadFromDisk(ID); err != nil {
		fmt.Println(err)
	}
	pls.ProcessedItems = append(pls.ProcessedItems, log)
	if err := pls.SaveToDisk(ID); err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
