package httpExporter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"net/http"
	"sync"
	"bytes"
	"io"
	"io/ioutil"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	defaultURL = "http://localhost:4000/"
)

// Exporter implements the SpanExporter interface that allows us to export span data
type Exporter struct {
	url         string
	serviceName string
	client      *http.Client
	logger      *log.Logger

	stoppedMu sync.RWMutex
	stopped   bool
}

var (
	_ sdktrace.SpanExporter = &Exporter{}
)

// Options contains configuration for the exporter.
type config struct {
	client *http.Client
	logger *log.Logger
}

// Option defines a function that configures the exporter.
type Option interface {
	apply(config) config
}

type optionFunc func(config) config

func (fn optionFunc) apply(cfg config) config {
	return fn(cfg)
}

// WithLogger configures the exporter to use the passed logger.
func WithLogger(logger *log.Logger) Option {
	return optionFunc(func(cfg config) config {
		cfg.logger = logger
		return cfg
	})
}

// WithClient configures the exporter to use the passed HTTP client.
func WithClient(client *http.Client) Option {
	return optionFunc(func(cfg config) config {
		cfg.client = client
		return cfg
	})
}

func New(collectorURL string, opts ...Option) (*Exporter, error) {
	if collectorURL == "" {
		// Use endpoint from env var or default collector URL.
		collectorURL = envOr(envEndpoint, defaultURL)
	}
	u, err := url.Parse(collectorURL)
	if err != nil {
		return nil, fmt.Errorf("invalid collector URL %q: %v", collectorURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid collector URL %q: no scheme or host", collectorURL)
	}

	cfg := config{}
	for _, opt := range opts {
		cfg = opt.apply(cfg)
	}

	if cfg.client == nil {
		cfg.client = http.DefaultClient
	}
	return &Exporter{
		url:    collectorURL,
		client: cfg.client,
		logger: cfg.logger,
	}, nil
}

// Export spans to fluent instance
func (e *Exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {

	e.stoppedMu.RLock()
	stopped := e.stopped
	e.stoppedMu.RUnlock()
	if stopped {
		e.logf("exporter stopped, not exporting span batch")
		return nil
	}

	if len(spans) == 0 {
		e.logf("no spans to export")
		return nil
	}

	httpSpans := convertSpansToHttp(spans)
	body, err := json.Marshal(&httpSpans)

	if err != nil {
		return e.errf("unable to serialize span data")
	}

	if body == nil{
		return e.errf("empty span data")
	}

	e.logf("about to send a POST request to %s with body %s", e.url, body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewBuffer(body))
	if err != nil {
		return e.errf("failed to create request to %s: %v", e.url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return e.errf("request to %s failed: %v", e.url, err)
	}
	defer resp.Body.Close()	

	_, err = io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		return e.errf("failed to read response body: %v", err)
	}

	if resp.StatusCode < 200 && resp.StatusCode > 300{
		return e.errf("failed to send spans to server with status %d", resp.StatusCode)
	}
	e.logf("Spans sent with response code %d", resp.StatusCode)

	return nil
}

// Shutdown stops the exporter flushing any pending exports.
func (e *Exporter) Shutdown(ctx context.Context) error {
	e.stoppedMu.Lock()
	e.stopped = true
	e.stoppedMu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil
}

func (e *Exporter) logf(format string, args ...interface{}) {
	if e.logger != nil {
		e.logger.Printf(format)
	}
}

func (e *Exporter) errf(format string, args ...interface{}) error {
	e.logf(format, args...)
	return fmt.Errorf(format)
}

// MarshalLog is the marshaling function used by the logging system to represent this exporter.
func (e *Exporter) MarshalLog() interface{} {
	return struct {
		Type string
		URL  string
	}{
		Type: "http",
		URL:  e.url,
	}
}

