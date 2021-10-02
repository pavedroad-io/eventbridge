package s3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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

const (
	NETWORK    string = "network"
	FILESYSTEM string = "filesystem"
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

// LogConfig passed data allowing processed logs
// to be read from a disk file or a eest end point
type LogConfig struct {
	LoadFrom string `json:"loadFrom"` // network or filesystem
	LoadURL  string `json:"loadURL"`
	// http://........pavedroad/plogs/UUID
	// where UUID is specific to this customer

	CustID string `json:"custID"` // Is prefixed to the file name on disk
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

func (pls *ProcessedLogs) Load(conf LogConfig) error {
	switch conf.LoadFrom {
	case NETWORK:
		return pls.LoadFromNetwokr(conf)
	case FILESYSTEM:
		return pls.LoadFromDisk(conf.CustID)
	default:
		msg := fmt.Errorf("Missing or invalid LoadFrom in LogConfig %w\n", conf.LoadFrom)
		return msg
	}

	return nil
}

func (pls *ProcessedLogs) Save(conf LogConfig) error {
	switch conf.LoadFrom {
	case NETWORK:
		return pls.SaveToNetwork(conf)
	case FILESYSTEM:
		return pls.SaveToDisk(conf.CustID)
	default:
		msg := fmt.Errorf("Missing or invalid LoadFrom in LogConfig %w\n", conf.LoadFrom)
		return msg
	}

	return nil
}

func (pls *ProcessedLogs) LoadFromNetwokr(conf LogConfig) error {
	respPlogs := ProcessedLogs{}

	req, err := http.NewRequest("GET", conf.LoadURL+"/"+pls.ID.String(), nil)
	if err != nil {
		log.Println("New Request faild", err)
	}

	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Do failed", err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Reading res.Body failed", err)
	}

	log.Println("Plogs load failed: ", string(body))

	if err := json.Unmarshal(body, &respPlogs); err != nil {
		fmt.Println("Unmarshall failed: ", err)
		log.Println("Unmarshall failed: ", err)
		return err
	}

	// Parse the plogsconf key to a UUID
	strid := conf.LoadURL[strings.LastIndexAny(conf.LoadURL, "/")+1:]
	uuid, err := uuid.Parse(strid)
	if err != nil {
		fmt.Errorf("Invalid PlogsUUID %v:%w\n", strid, err)
	} else {
		pls.ID = uuid
	}
	pls.ProcessedItems = respPlogs.ProcessedItems

	return nil
}

func (pls *ProcessedLogs) SaveToNetwork(conf LogConfig) error {

	payload, err := json.Marshal(pls)
	if err != nil {
		fmt.Println("Marshall failed: ", err)
		log.Println("Marshall failed: ", err)
		return err
	}

	req, err := http.NewRequest("PUT", conf.LoadURL+"/"+pls.ID.String(), bytes.NewBuffer(payload))
	if err != nil {
		log.Println("New Request faild", err)
	}

	req.Header.Add("content-type", "application/json")

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Do failed", err)
	}

	return nil
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

func (pls *ProcessedLogs) AddProcessLog(ID string, log ProcessedLogItem, conf LogConfig) error {

	/*
		if err := pls.LoadFromDisk(ID); err != nil {
			fmt.Println("LoadFromDisk failed: ", err)
		}
	*/

	pls.ProcessedItems = append(pls.ProcessedItems, log)
	if conf.LoadFrom == FILESYSTEM {
		if err := pls.SaveToDisk(ID); err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}
