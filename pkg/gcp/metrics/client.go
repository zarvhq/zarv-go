// Package metrics provides a thin client to publish custom metrics to
// Google Cloud Monitoring (Stackdriver).
package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Cfg contains the configuration required to send metrics.
type Cfg struct {
	ProjectID       string
	CredentialsJSON []byte // Optional: use Workload Identity if empty
	Endpoint        string // Optional: custom endpoint (e.g., emulator)
	// Timeout defines a default deadline for write operations when the caller
	// does not provide a context deadline. Zero applies a 5s default.
	Timeout time.Duration

	// Monitored resource information. Defaults to global with project_id.
	ResourceType   string
	ResourceLabels map[string]string
}

// Client defines the operations supported by the metrics publisher.
type Client interface {
	// WriteGauge sends a single gauge datapoint with the given metric type and labels.
	// External/custom metrics in Cloud Monitoring support GAUGE and CUMULATIVE.
	WriteGauge(ctx context.Context, metricType string, labels map[string]string, value float64) error
	// WriteCumulative sends a cumulative datapoint. Requires the interval start time.
	WriteCumulative(ctx context.Context, metricType string, labels map[string]string, start time.Time, value float64) error
	// Close closes underlying connections.
	Close() error
}

type client struct {
	project  string
	api      *monitoring.MetricClient
	resource *monitoredrespb.MonitoredResource
	timeout  time.Duration
}

// NewClient creates a new metrics client backed by Cloud Monitoring.
func NewClient(ctx context.Context, cfg *Cfg) (Client, error) {
	if cfg == nil || cfg.ProjectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	opts := []option.ClientOption{}
	if len(cfg.CredentialsJSON) > 0 {
		creds, err := google.CredentialsFromJSON(ctx, cfg.CredentialsJSON, monitoring.DefaultAuthScopes()...)
		if err != nil {
			return nil, fmt.Errorf("error creating credentials from JSON: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}
	if cfg.Endpoint != "" {
		opts = append(opts, option.WithEndpoint(cfg.Endpoint))
	}

	api, err := monitoring.NewMetricClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring client: %w", err)
	}

	resourceType := cfg.ResourceType
	if resourceType == "" {
		resourceType = "global"
	}

	resourceLabels := map[string]string{
		"project_id": cfg.ProjectID,
	}

	maps.Copy(resourceLabels, cfg.ResourceLabels)
	to := cfg.Timeout
	if to <= 0 {
		to = 5 * time.Second
	}

	return &client{
		project: "projects/" + cfg.ProjectID,
		api:     api,
		resource: &monitoredrespb.MonitoredResource{
			Type:   resourceType,
			Labels: resourceLabels,
		},
		timeout: to,
	}, nil
}

// WriteGauge sends a single gauge datapoint to Cloud Monitoring.
func (c *client) WriteGauge(ctx context.Context, metricType string, labels map[string]string, value float64) error {
	return c.writePoint(ctx, metricType, labels, value, time.Time{}, metricpb.MetricDescriptor_GAUGE)
}

// WriteCumulative sends a cumulative datapoint to Cloud Monitoring; start marks the interval beginning.
func (c *client) WriteCumulative(ctx context.Context, metricType string, labels map[string]string, start time.Time, value float64) error {
	return c.writePoint(ctx, metricType, labels, value, start, metricpb.MetricDescriptor_CUMULATIVE)
}

// Close closes the Monitoring client.
func (c *client) Close() error {
	if c.api == nil {
		return nil
	}
	if err := c.api.Close(); err != nil {
		slog.Error("failed to close monitoring client", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (c *client) writePoint(ctx context.Context, metricType string, labels map[string]string, value float64, start time.Time, kind metricpb.MetricDescriptor_MetricKind) error {
	if metricType == "" {
		return fmt.Errorf("metricType cannot be empty")
	}
	if !strings.HasPrefix(metricType, "custom.googleapis.com/") && !strings.HasPrefix(metricType, "external.googleapis.com/") {
		return fmt.Errorf("metricType must be a custom or external metric (custom.googleapis.com/... or external.googleapis.com/...)")
	}
	if kind == metricpb.MetricDescriptor_CUMULATIVE {
		now := time.Now().UTC()
		if start.IsZero() {
			return fmt.Errorf("start time is required for cumulative metrics")
		}
		if !start.Before(now) {
			return fmt.Errorf("start time must be before current time for cumulative metrics")
		}
	}

	// Ensure we don't hang indefinitely on network issues; caller can override with their own deadline.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	metricLabels := make(map[string]string, len(labels))
	maps.Copy(metricLabels, labels)

	now := time.Now().UTC()
	interval := &monitoringpb.TimeInterval{EndTime: timestamppb.New(now)}
	if kind == metricpb.MetricDescriptor_CUMULATIVE {
		interval.StartTime = timestamppb.New(start)
	}

	ts := &monitoringpb.TimeSeries{
		Metric: &metricpb.Metric{
			Type:   metricType,
			Labels: metricLabels,
		},
		Resource:   c.resource,
		MetricKind: kind,
		ValueType:  metricpb.MetricDescriptor_DOUBLE,
		Points: []*monitoringpb.Point{
			{
				Interval: interval,
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_DoubleValue{DoubleValue: value},
				},
			},
		},
	}

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       c.project,
		TimeSeries: []*monitoringpb.TimeSeries{ts},
	}

	if err := c.api.CreateTimeSeries(ctx, req); err != nil {
		return fmt.Errorf("failed to write time series: %w", err)
	}
	return nil
}
