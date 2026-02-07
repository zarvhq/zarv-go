package documentai

import (
	"context"

	documentai "cloud.google.com/go/documentai/apiv1"
	documentaipb "cloud.google.com/go/documentai/apiv1/documentaipb"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Cfg holds configuration needed to talk to Document AI.
type Cfg struct {
	ProjectID       string
	Location        string
	CredentialsJSON []byte // Optional: if not provided, uses Application Default Credentials (Workload Identity)
}

// Client exposes the minimal Document AI operations used by the service.
type Client interface {
	ProcessDocument(ctx context.Context, file []byte, mimeType string, processorID string) (*documentaipb.Document, error)
}

type client struct {
	docAIClient *documentai.DocumentProcessorClient
	cfg         *Cfg
}

// NewClient builds a Document AI client using optional explicit credentials.
func NewClient(ctx context.Context, cfg *Cfg) (Client, error) {
	var opts []option.ClientOption
	if cfg.CredentialsJSON != nil {
		creds, err := google.CredentialsFromJSON(ctx, cfg.CredentialsJSON, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, err
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	docAIClient, err := documentai.NewDocumentProcessorClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &client{
		docAIClient: docAIClient,
		cfg:         cfg,
	}, nil
}

// ProcessDocument sends a document to the configured processor and returns the parsed result.
func (c *client) ProcessDocument(ctx context.Context, file []byte, mimeType string, processorID string) (*documentaipb.Document, error) {
	req := &documentaipb.ProcessRequest{
		Name: "projects/" + c.cfg.ProjectID + "/locations/" + c.cfg.Location + "/processors/" + processorID,
		Source: &documentaipb.ProcessRequest_RawDocument{
			RawDocument: &documentaipb.RawDocument{
				Content:  file,
				MimeType: mimeType,
			},
		},
	}

	resp, err := c.docAIClient.ProcessDocument(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetDocument(), nil
}
