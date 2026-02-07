package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const ErrObjectNotFound = "object not found"

type Cfg struct {
	BucketName      string
	DatalakeBucket  string // Datalake bucket name
	CredentialsJSON []byte // Optional: if not provided, uses Application Default Credentials (Workload Identity)
	Endpoint        string // Optional: for local development with fake-gcs-server
	Local           bool   // Set to true for local development
}

type Object struct {
	Key         string `json:"key,omitempty"`
	Data        []byte `json:"data,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
}

type SignedURL struct {
	URL       string `json:"url"`
	Method    string `json:"method"`
	ExpiresAt string `json:"expiresAt"`
}

type Client interface {
	GetObject(key string) (*Object, error)
	PutObject(obj *Object) error
	GetObjectSignedURL(objectKey, method string) (*SignedURL, error)
	PutObjectSignedURL(objectKey, method string) (*SignedURL, error)
	Close() error
}

type client struct {
	storage        *storage.Client
	bucketName     string
	datalakeBucket string
	ctx            context.Context
}

const (
	lifetimeSecs = 120
)

func NewClient(ctx context.Context, cfg *Cfg) (*client, error) {
	var opts []option.ClientOption

	// Use credentials JSON if provided, otherwise use Application Default Credentials (Workload Identity)
	if len(cfg.CredentialsJSON) > 0 {
		creds, err := google.CredentialsFromJSON(ctx, cfg.CredentialsJSON, storage.ScopeFullControl)
		if err != nil {
			return nil, fmt.Errorf("error creating credentials from JSON: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}
	// If no credentials provided, the SDK will automatically use Application Default Credentials
	// This works with Workload Identity in GKE

	// Configure for local development with fake-gcs-server
	if cfg.Local && cfg.Endpoint != "" {
		opts = append(opts, option.WithEndpoint(cfg.Endpoint))
		opts = append(opts, option.WithoutAuthentication())
	}

	// Create a Google Cloud Storage client
	storageClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating storage client: %w", err)
	}

	return &client{
		storage:        storageClient,
		bucketName:     cfg.BucketName,
		datalakeBucket: cfg.DatalakeBucket,
		ctx:            ctx,
	}, nil
}

// GetObject retrieves an object from a bucket and returns its data.
func (c *client) GetObject(key string) (*Object, error) {
	if key == "" {
		return nil, fmt.Errorf("object key is empty")
	}

	bucket := c.storage.Bucket(c.bucketName)
	obj := bucket.Object(key)

	// Get object attributes to read the content type and encoding
	attrs, err := obj.Attrs(c.ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, fmt.Errorf("%s: %s", ErrObjectNotFound, key)
		}
		return nil, fmt.Errorf("error getting object attributes: %w", err)
	}

	reader, err := obj.NewReader(c.ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, fmt.Errorf("%s: %s", ErrObjectNotFound, key)
		}
		return nil, fmt.Errorf("error getting object: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading object: %w", err)
	}

	return &Object{
		Key:         key,
		Data:        data,
		ContentType: attrs.ContentType,
		Encoding:    attrs.ContentEncoding,
	}, nil
}

// PutObject uploads an object to a bucket with the specified content type and encoding.
func (c *client) PutObject(obj *Object) error {
	if obj == nil {
		return fmt.Errorf("object is nil")
	}

	if obj.Key == "" {
		return fmt.Errorf("object key is empty")
	}

	bucket := c.storage.Bucket(c.bucketName)
	storageObj := bucket.Object(obj.Key)

	writer := storageObj.NewWriter(c.ctx)
	writer.ContentType = obj.ContentType
	writer.ContentEncoding = obj.Encoding

	if _, err := io.Copy(writer, bytes.NewReader(obj.Data)); err != nil {
		writer.Close()
		return fmt.Errorf("error writing object: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("error closing writer: %w", err)
	}

	return nil
}

// GetObjectSignedURL creates a signed URL that can be used to download an object from the main bucket.
// The signed URL is valid for the specified number of seconds.
func (c *client) GetObjectSignedURL(objectKey, method string) (*SignedURL, error) {
	if objectKey == "" {
		return nil, fmt.Errorf("object key is empty")
	}

	expiresAt := time.Now().Add(time.Duration(lifetimeSecs) * time.Second)

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  method,
		Expires: expiresAt,
	}

	url, err := c.storage.Bucket(c.bucketName).SignedURL(objectKey, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating signed URL: %w", err)
	}

	return &SignedURL{
		URL:       url,
		Method:    method,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// PutObjectSignedURL creates a signed URL that can be used to upload an object to the datalake bucket.
// The signed URL is valid for the specified number of seconds.
func (c *client) PutObjectSignedURL(objectKey, method string) (*SignedURL, error) {
	if objectKey == "" {
		return nil, fmt.Errorf("object key is empty")
	}

	expiresAt := time.Now().Add(time.Duration(lifetimeSecs) * time.Second)

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  method,
		Expires: expiresAt,
	}

	url, err := c.storage.Bucket(c.datalakeBucket).SignedURL(objectKey, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating signed URL: %w", err)
	}

	return &SignedURL{
		URL:       url,
		Method:    method,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// Close closes the GCS client
func (c *client) Close() error {
	if c.storage != nil {
		return c.storage.Close()
	}
	return nil
}
