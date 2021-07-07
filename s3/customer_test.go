package s3

import (
	"fmt"
	"testing"
)

var customerList []Customer
var customer Customer

func TestMain(t *testing.T) {
}

func TestLoadCustomers(t *testing.T) {
	_, err := customer.LoadFromDisk("customer.yaml")
	if err != nil {
		fmt.Println("err: ", err)
		t.Errorf("fail loading customer.yaml: %v\n", err)
	}
}

func TestGenerateManifests(t *testing.T) {
	customerList, _ := customer.LoadFromDisk("customer.yaml")
	//fmt.Println(customerList)
	for _, c := range customerList {
		c.Configuration.GenerateManifests(&c)
	}
}
