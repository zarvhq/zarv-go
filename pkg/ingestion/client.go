package ingestion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// The environment a Client reads when its Config leaves a field empty.
const (
	// EnvURL is the gateway's base URL. Unset, DefaultURL is used.
	EnvURL = "ZARV_INGESTION_URL"
	// EnvKey is the Bearer key, for a gateway that asks for one. Unset, no
	// Authorization header is sent.
	EnvKey = "ZARV_INGESTION_KEY"
)

// DefaultURL is the gateway inside the cluster, the same in every environment.
// It takes no key: only the cluster reaches it.
const DefaultURL = "http://data-ingest.data-stack.svc.cluster.local:8080"

// Defaults applied by New when the Config leaves a field at its zero value.
const (
	DefaultTimeout     = 30 * time.Second
	DefaultMaxAttempts = 4
	defaultPath        = "/v1/ingestion"
)

// ErrInvalidConfig is what New wraps when the configuration cannot build a
// Client. The message names the variable or field, never its value.
var ErrInvalidConfig = errors.New("ingestion: invalid configuration")

// Config configures a Client. URL falls back to EnvURL, then DefaultURL; Key
// falls back to EnvKey, then to none.
type Config struct {
	// URL is the gateway's base URL. Without a path, /v1/ingestion is used;
	// with one, it is used as it is. Defaults to $ZARV_INGESTION_URL, then
	// DefaultURL.
	URL string
	// Key is the Bearer key. Defaults to $ZARV_INGESTION_KEY; empty sends none.
	Key string
	// HTTPClient sends the requests. Defaults to one with DefaultTimeout.
	HTTPClient *http.Client
	// Logger receives failed sends. Defaults to slog.Default().
	Logger *slog.Logger
	// MaxAttempts bounds the tries of one send, retries included. Zero means
	// DefaultMaxAttempts.
	MaxAttempts int
}

// Client sends events to the ingestion gateway. It is safe for concurrent use.
type Client struct {
	endpoint    string
	key         string
	http        *http.Client
	log         *slog.Logger
	maxAttempts int
	sleep       func(context.Context, time.Duration) error
}

// New builds a Client from cfg and the environment. It fails at once, naming
// the variable, when the URL given is not an absolute http(s) URL, so a
// misconfigured service fails at start rather than on its first event.
func New(cfg Config) (*Client, error) {
	raw := strings.TrimSpace(firstNonEmpty(cfg.URL, os.Getenv(EnvURL)))
	if raw == "" {
		raw = DefaultURL
	}
	endpoint, err := endpointOf(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s %v", ErrInvalidConfig, EnvURL, err)
	}

	key := strings.TrimSpace(firstNonEmpty(cfg.Key, os.Getenv(EnvKey)))

	if cfg.MaxAttempts < 0 {
		return nil, fmt.Errorf("%w: MaxAttempts is %d, it cannot be negative", ErrInvalidConfig, cfg.MaxAttempts)
	}
	attempts := cfg.MaxAttempts
	if attempts == 0 {
		attempts = DefaultMaxAttempts
	}

	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: DefaultTimeout}
	}
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	return &Client{endpoint: endpoint, key: key, http: hc, log: log, maxAttempts: attempts, sleep: sleepCtx}, nil
}

// endpointOf validates the base URL and resolves the route. The error never
// quotes the input: a URL can carry credentials in its userinfo.
func endpointOf(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("does not parse as a URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("must be an absolute http or https URL")
	}
	if u.Host == "" {
		return "", errors.New("has no host")
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = defaultPath
	}
	return u.String(), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
