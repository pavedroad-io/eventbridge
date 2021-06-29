package s3

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

func ListBucketObjects(client *minio.Client, bucket string, opts minio.ListObjectsOptions) (objects []minio.ObjectInfo, err error) {

	for obj := range client.ListObjects(context.Background(), bucket, opts) {
		if obj.Err != nil {
			fmt.Println(obj.Err)
			return nil, nil
		}
		objects = append(objects, obj)
	}
	return objects, nil
}
