// Package documentai provides a client for Google Cloud Document AI.
//
// This package offers a simple interface for processing documents using
// Google Cloud Document AI processors.
//
// Example usage:
//
//	import (
//		"context"
//		"github.com/zarvhq/zarv-go/pkg/gcp/documentai"
//	)
//
//	func main() {
//		ctx := context.Background()
//		cfg := &documentai.Cfg{
//			ProjectID: "my-project",
//			Location:  "us",
//		}
//
//		client, err := documentai.NewClient(ctx, cfg)
//		if err != nil {
//			panic(err)
//		}
//
//		doc, err := client.ProcessDocument(ctx, fileBytes, "application/pdf", "processor-id")
//		if err != nil {
//			panic(err)
//		}
//	}
package documentai
