package ingestion

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

const secret = "s3cr3t-ingestion-key"

func TestNewReadsTheGatewayFromTheEnvironment(t *testing.T) {
	t.Setenv(EnvURL, "http://ingestion.data.svc:8080")
	t.Setenv(EnvKey, secret)

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got, want := c.endpoint, "http://ingestion.data.svc:8080/v1/ingestion"; got != want {
		t.Errorf("endpoint = %q, want %q", got, want)
	}
	if c.key != secret {
		t.Error("the key was not read from the environment")
	}
}

func TestNewFailsWithoutTheURLNamingTheVariable(t *testing.T) {
	t.Setenv(EnvURL, "")
	t.Setenv(EnvKey, secret)

	_, err := New(Config{})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want ErrInvalidConfig", err)
	}
	if !strings.Contains(err.Error(), EnvURL) {
		t.Errorf("the error does not name %s: %v", EnvURL, err)
	}
}

func TestNewRefusesAURLThatIsNotAnAbsoluteHTTPURL(t *testing.T) {
	t.Setenv(EnvKey, secret)
	for _, raw := range []string{
		"ingestion.data.svc:8080",  // no scheme
		"/v1/ingestion",            // relative
		"ftp://ingestion.data.svc", // not http
		"http://",                  // no host
		"http://bad host:8080",     // does not parse
		"   ",                      // blank
	} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv(EnvURL, raw)
			_, err := New(Config{})
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("New accepted %q: err = %v", raw, err)
			}
			if !strings.Contains(err.Error(), EnvURL) {
				t.Errorf("the error does not name %s: %v", EnvURL, err)
			}
		})
	}
}

func TestNewFailsWithoutTheKeyNamingTheVariable(t *testing.T) {
	t.Setenv(EnvURL, "http://ingestion.data.svc:8080")
	t.Setenv(EnvKey, "")

	_, err := New(Config{})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want ErrInvalidConfig", err)
	}
	if !strings.Contains(err.Error(), EnvKey) {
		t.Errorf("the error does not name %s: %v", EnvKey, err)
	}
}

// A secret must never appear in an error: errors end up in logs. A URL can
// carry one in its userinfo, so the error never quotes the URL either.
func TestAConfigErrorNeverCarriesASecret(t *testing.T) {
	t.Setenv(EnvURL, "ftp://svc:"+secret+"@gw:8080")
	t.Setenv(EnvKey, secret)

	_, err := New(Config{})
	if err == nil {
		t.Fatal("New accepted an invalid URL")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("the error carries the key: %v", err)
	}
}

func TestConfigOverridesTheEnvironment(t *testing.T) {
	t.Setenv(EnvURL, "http://from-env:8080")
	t.Setenv(EnvKey, "from-env")

	c, err := New(Config{URL: "https://from-config.example", Key: "from-config"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.endpoint != "https://from-config.example/v1/ingestion" || c.key != "from-config" {
		t.Errorf("endpoint, key = %q, %q; want the Config's values", c.endpoint, c.key)
	}
}

// A URL that already names a path is used as it is: the gateway's route is its
// config, and a stream may serve another path.
func TestAURLWithAPathIsUsedAsItIs(t *testing.T) {
	t.Setenv(EnvKey, secret)
	for raw, want := range map[string]string{
		"http://gw:8080":                "http://gw:8080/v1/ingestion",
		"http://gw:8080/":               "http://gw:8080/v1/ingestion",
		"http://gw:8080/v1/ingestion":   "http://gw:8080/v1/ingestion",
		"https://gw.example/custom/ing": "https://gw.example/custom/ing",
	} {
		c, err := New(Config{URL: raw})
		if err != nil {
			t.Fatalf("New(%q): %v", raw, err)
		}
		if c.endpoint != want {
			t.Errorf("New(%q).endpoint = %q, want %q", raw, c.endpoint, want)
		}
	}
}

func TestNewAppliesTheDefaults(t *testing.T) {
	t.Setenv(EnvURL, "http://gw:8080")
	t.Setenv(EnvKey, secret)

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.http == nil || c.http.Timeout != DefaultTimeout {
		t.Errorf("http client = %+v, want one with a %s timeout", c.http, DefaultTimeout)
	}
	if c.log != slog.Default() {
		t.Error("the logger is not slog.Default()")
	}
	if c.maxAttempts != DefaultMaxAttempts {
		t.Errorf("maxAttempts = %d, want %d", c.maxAttempts, DefaultMaxAttempts)
	}
}

func TestNewKeepsWhatTheCallerGives(t *testing.T) {
	t.Setenv(EnvURL, "http://gw:8080")
	t.Setenv(EnvKey, secret)
	hc := &http.Client{Timeout: 5 * time.Second}
	log := slog.New(slog.DiscardHandler)

	c, err := New(Config{HTTPClient: hc, Logger: log, MaxAttempts: 2})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.http != hc || c.log != log || c.maxAttempts != 2 {
		t.Error("New replaced a value the caller provided")
	}
}

func TestNewRefusesANegativeMaxAttempts(t *testing.T) {
	t.Setenv(EnvURL, "http://gw:8080")
	t.Setenv(EnvKey, secret)

	if _, err := New(Config{MaxAttempts: -1}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("err = %v, want ErrInvalidConfig", err)
	}
}
