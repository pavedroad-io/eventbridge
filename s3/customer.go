package s3

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type Customers struct {
	Customers []Customer `yaml:"customers"`
}

// Customer configuration and information
type Customer struct {
	// ID unique ID for this customer
	ID uuid.UUID `yaml:"id" json:"customersuuid"`

	// ShortName is the first portion of the UUID
	// used persisting customer data in a human readable
	// fashion
	shortName string `yaml:"short" json:"short"`

	// Customer Name
	Name string `yaml:"name" json:"name"`

	// Logs to monitor
	Logs []LogBuckets `yaml:"logs" json:"logs"`

	// Providers associated with logs
	Providers Providers `yaml:"providers" json:"providers"`

	// Syncconfiguration
	Configuration SyncConfiguration `yaml:"config" json:"configuration"`
}

func (c *Customer) ShortName() string {
	if c.Name == "" {
		c.Name = "default"
		fmt.Println("configname: ", c.Name)
	}

	if c.shortName != "" {
		fmt.Println("shortname is set: ", c.shortName)
		return c.shortName
	}

	c.shortName = strings.Split(c.ID.String(), "-")[0]
	fmt.Println("shortname is new: ", c.shortName)

	return c.shortName
}

func (c *Customer) LoadFromDisk(file string) ([]Customer, error) {
	cl := []Customer{}

	f, err := os.Open(file)
	if err != nil {
		fmt.Println("failed to open:", file, ", error:", err)
	}
	defer f.Close()

	byteValue, e := ioutil.ReadAll(f)
	if e != nil {
		fmt.Println("read failed for ", file)
		return nil, err
	}

	err = yaml.Unmarshal([]byte(byteValue), &cl)
	if err != nil {
		fmt.Println("Unmarshal faild", err)
		return nil, err
	}

	return cl, nil
}

func (c *Customer) LoadFromNetwork(url string) ([]Customer, error) {
	cl := []Customer{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("New Request faild", err)
	}

	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Do failed", err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Reading res.Body failed", err)
	}

	if err := json.Unmarshal(body, &cl); err != nil {
		fmt.Println("Unmarshall failed: ", err)
		log.Println("Unmarshall failed: ", err)
		return cl, err
	}

	return cl, nil
}
