// Package gcs provides a client for Google Cloud Storage operations.
//
// This package offers a simple interface for interacting with Google Cloud Storage,
// supporting both production (with Workload Identity) and local development scenarios.
//
// Example usage:
//
//	import (
//		"context"
//		"github.com/zarvhq/zarv-go/pkg/gcp/gcs"
//	)
//
//	func main() {
//		ctx := context.Background()
//		cfg := &gcs.Cfg{
//			BucketName: "my-bucket",
//		}
//
//		client, err := gcs.NewClient(ctx, cfg)
//		if err != nil {
//			panic(err)
//		}
//
//		// Upload a file
//		err = client.PutObject("path/file.txt", "text/plain", "", []byte("content"))
//		if err != nil {
//			panic(err)
//		}
//
//		// Download a file
//		data, err := client.GetObject("path/file.txt")
//		if err != nil {
//			panic(err)
//		}
//	}
package gcs
