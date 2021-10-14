package s3

import (
	"fmt"
	"log"
	"testing"

	"github.com/minio/minio-go/v7"
)

/*
	Credentials: "veU44P58g9OeJZG5dFMu0wYYCRoDcBJGeJwddHFu",
	Key:         "JBUYC2CL0DIT0GVO3DBU",
*/

var testProvider Provider = Provider{
	Name:        "wasabi",
	Credentials: "veU44P58g9OeJZG5dFMu0wYYCRoDcBJGeJwddHFu",
	Key:         "JBUYC2CL0DIT0GVO3DBU",
	Region:      "us-west-1",
	Endpoint:    "s3.us-west-1.wasabisys.com",
}

func TestListBuck(t *testing.T) {
	opts := minio.ListObjectsOptions{
		Recursive: true,
		Prefix:    "",
	}

	/*
		if err != nil {
			log.Printf("Provider not found: %v\n", err)
			continue
		}
	*/

	fmt.Println("Using provider: ", testProvider)
	s3Client, err := NewClient(testProvider)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("s3.ListBucketObjects: ", "pipeline-artifact-logs")
	objects, err := ListBucketObjects(s3Client, "pipeline-artifact-logs", opts)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(objects)
}
