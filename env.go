package httpExporter

import "os"

// Environment variable names.
const (
	// Http endpoint
	envEndpoint = "OTEL_EXPORTER_HTTP_ENDPOINT"
)

// envOr returns an env variable's value if it is exists or the default if not.
func envOr(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultValue
}
