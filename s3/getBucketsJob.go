package s3

import (
	"context"

	"github.com/minio/minio-go/v7"
)

func ListBuckets(c *minio.Client) ([]minio.BucketInfo, error) {

	buckets, err := c.ListBuckets(context.Background())
	if err != nil {
		return nil, err
	}
	return buckets, nil
}
