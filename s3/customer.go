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

// Return list of customerss
//
//
// swagger:response customersList
type listResponse struct {
	// in: body
	UUID string `json:"uuid"`
}

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
	}

	if c.shortName != "" {
		return c.shortName
	}

	c.shortName = strings.Split(c.ID.String(), "-")[0]

	return c.shortName
}

func (c *Customer) LoadFromDisk(file string) ([]Customer, error) {
	cl := []Customer{}

	f, err := os.Open(file)
	if err != nil {
		log.Println("failed to open:", file, ", error:", err)
	}
	defer f.Close()

	byteValue, e := ioutil.ReadAll(f)
	if e != nil {
		log.Println("read failed for ", file)
		return nil, err
	}

	err = yaml.Unmarshal([]byte(byteValue), &cl)
	if err != nil {
		log.Println("Unmarshal failed", err)
		return nil, err
	}

	return cl, nil
}

func (c *Customer) LoadFromNetwork(url string) ([]Customer, error) {
	cl := []Customer{}
	var lr []listResponse
	requrl := url + "LIST"

	req, err := http.NewRequest("GET", requrl, nil)
	if err != nil {
		log.Println("New Request faild", err)
	}

	q := req.URL.Query()
	q.Add("count", "999999999")
	req.URL.RawQuery = q.Encode()

	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	fmt.Println(res, err)
	if err != nil || res.StatusCode != 200 {
		log.Printf("Do failed err: %v \nURL: %v\nBody: %v\n", err, requrl, res)
		return cl, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Reading res.Body failed", err)
	}

	if err := json.Unmarshal(body, &lr); err != nil {
		fmt.Println("Unmarshall failed: ", err)
		log.Println("Unmarshall failed: ", err)
		return cl, err
	}

	for _, v := range lr {

		nc, err := c.GetCustomer(url, v.UUID)
		if err != nil {
			log.Println("Failed loading customer: ", v)
		} else {
			// Only add customers with logs defined
			// TODO: 21/09/24:12:jscharber refactor Add data
			// attribute for active accounts
			if len(nc.Logs) > 0 {
				var lognames []string
				for _, l := range nc.Logs {
					lognames = append(lognames, l.Name)
				}
				log.Printf("Adding logs[%v] for %v\n", lognames, v.UUID)
				cl = append(cl, nc)
			} else {
				log.Printf("Skipping %v no logs defined\n", v.UUID)
			}
		}
	}
	return cl, nil
}

func (c *Customer) GetCustomer(url, id string) (Customer, error) {
	rid := "/" + id
	cust := Customer{}

	req, err := http.NewRequest("GET", url+rid, nil)
	if err != nil {
		log.Println("New Request failed", err)
		log.Println("URL ", url+rid)
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

	if err := json.Unmarshal(body, &cust); err != nil {
		fmt.Println("Unmarshall failed: ", err)
		log.Println("Unmarshall failed: ", err)
		return cust, err
	}

	return cust, nil
}
