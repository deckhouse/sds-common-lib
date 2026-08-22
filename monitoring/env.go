package monitoring

import "os"

// BindAddressFromEnv reads METRICS_BIND_ADDRESS, falling back to
// DefaultMetricsBindAddress when it is unset or empty.
func BindAddressFromEnv() string {
	return BindAddressFromEnvOr(DefaultMetricsBindAddress)
}

// BindAddressFromEnvOr reads METRICS_BIND_ADDRESS, falling back to fallback when
// it is unset or empty. Use it for a binary that cannot have the fleet-wide
// default, because something else in its pod holds that port.
func BindAddressFromEnvOr(fallback string) string {
	if addr := os.Getenv(EnvMetricsBindAddress); addr != "" {
		return addr
	}
	return fallback
}
