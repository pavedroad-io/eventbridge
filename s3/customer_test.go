package s3

import (
	"fmt"
	"testing"
)

var customerList Customer

func TestMain(t *testing.T) {
}

func TestLoadCustomers(t *testing.T) {
	customerList, err := customerList.LoadFromDisk("customer.yaml")
	if err != nil {
		fmt.Println("err: ", err)
		t.Errorf("fail loading customer.yaml: %v\n", err)
	}

	for _, c := range customerList {

		fmt.Println(c)
	}
}
