package s3

import (
	"fmt"
	"testing"
	"time"
)

var customerList []Customer
var customer Customer
var caller SyncInitiator

func TestMain(t *testing.T) {
	customerList, _ = customer.LoadFromDisk("customer.yaml")
	caller = SyncInitiator{
		CustomerID:         "1234",
		UserID:             "5678",
		AuthorizationToken: "none",
		ReferenceID:        "none"}
}

func TestLoadCustomers(t *testing.T) {
	_, err := customer.LoadFromDisk("customer.yaml")
	if err != nil {
		t.Errorf("fail loading customer.yaml: %v\n", err)
	}
}

func TestGenerateManifests(t *testing.T) {
	for _, c := range customerList {
		c.Configuration.GenerateManifests(&c, caller)
	}
}
func TestDeployManifests(t *testing.T) {
	for _, c := range customerList {
		data, err := c.Configuration.DeployManifests(&c, caller)
		if err != nil {
			t.Errorf("Delete deployment failed: %s\n", err)
		}
		fmt.Println(string(data))
	}
}

func TestGetDeployment(t *testing.T) {
	// Give enough time for deployment to complete
	time.Sleep(2 * time.Second)
	for _, c := range customerList {
		data, err := c.Configuration.GetDeployments(&c, caller)
		if err != nil {
			t.Errorf("Get deployment failed: %s\n", err)
		}
		fmt.Println(string(data))
	}
}

func TestDeleteDeployment(t *testing.T) {
	// Give enough time for deployment to complete
	time.Sleep(30 * time.Second)
	for _, c := range customerList {
		data, err := c.Configuration.DeleteDeployment(&c, caller)
		if err != nil {
			t.Errorf("Delete deployment failed: %s\n", err)
			fmt.Println(string(data))
		}
	}
}
