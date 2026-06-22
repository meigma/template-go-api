// Package config defines the API server's runtime configuration, loaded from
// flags and TEMPLATE_GO_API_* environment variables via Viper.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultAddr              = ":8080"
	defaultReadTimeout       = 5 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultWriteTimeout      = 10 * time.Second
	defaultIdleTimeout       = 120 * time.Second
	defaultRequestTimeout    = 15 * time.Second
	defaultShutdownGrace     = 15 * time.Second
	defaultLogLevel          = "info"
	defaultLogFormat         = "json"
)

// Config holds runtime settings for the API server.
type Config struct {
	// Addr is the host:port the HTTP server listens on.
	Addr string
	// ReadTimeout bounds the time spent reading an entire request.
	ReadTimeout time.Duration
	// ReadHeaderTimeout bounds the time spent reading request headers.
	ReadHeaderTimeout time.Duration
	// WriteTimeout bounds the time spent writing the response.
	WriteTimeout time.Duration
	// IdleTimeout bounds how long an idle keep-alive connection is kept open.
	IdleTimeout time.Duration
	// RequestTimeout bounds per-request processing in the timeout middleware.
	RequestTimeout time.Duration
	// ShutdownGrace bounds graceful shutdown before in-flight requests are dropped.
	ShutdownGrace time.Duration
	// LogLevel is the minimum slog level (debug, info, warn, error).
	LogLevel string
	// LogFormat selects the slog handler (json or text).
	LogFormat string
	// CORSAllowedOrigins lists the origins permitted by the CORS middleware.
	// Empty (the default) disables CORS entirely.
	CORSAllowedOrigins []string
	// TrustedProxyHeader names a proxy-set header (for example X-Real-IP) to
	// read the client IP from. Empty (the default) trusts only the direct TCP
	// peer, which cannot be spoofed.
	TrustedProxyHeader string
}

// RegisterFlags declares the server configuration flags on flags. Binding them
// to a Viper instance makes flags, environment variables, and defaults compose.
func RegisterFlags(flags *pflag.FlagSet) {
	flags.String("addr", defaultAddr, "host:port the HTTP server listens on")
	flags.String("log-level", defaultLogLevel, "log level: debug, info, warn, or error")
	flags.String("log-format", defaultLogFormat, "log format: json or text")
	flags.Duration("read-timeout", defaultReadTimeout, "maximum duration for reading an entire request")
	flags.Duration("read-header-timeout", defaultReadHeaderTimeout, "maximum duration for reading request headers")
	flags.Duration("write-timeout", defaultWriteTimeout, "maximum duration before timing out response writes")
	flags.Duration("idle-timeout", defaultIdleTimeout, "maximum time to wait for the next keep-alive request")
	flags.Duration("request-timeout", defaultRequestTimeout, "per-request processing timeout")
	flags.Duration("shutdown-grace", defaultShutdownGrace, "maximum duration to await graceful shutdown")
	flags.StringSlice("cors-allowed-origins", nil, "allowed CORS origins (comma-separated); empty disables CORS")
	flags.String(
		"trusted-proxy-header",
		"",
		"proxy header to read the client IP from (for example X-Real-IP); empty trusts the TCP peer",
	)
}

// Load reads the server configuration from vp, applying defaults for unset keys.
func Load(vp *viper.Viper) Config {
	setDefaults(vp)

	return Config{
		Addr:               vp.GetString("addr"),
		ReadTimeout:        vp.GetDuration("read-timeout"),
		ReadHeaderTimeout:  vp.GetDuration("read-header-timeout"),
		WriteTimeout:       vp.GetDuration("write-timeout"),
		IdleTimeout:        vp.GetDuration("idle-timeout"),
		RequestTimeout:     vp.GetDuration("request-timeout"),
		ShutdownGrace:      vp.GetDuration("shutdown-grace"),
		LogLevel:           vp.GetString("log-level"),
		LogFormat:          vp.GetString("log-format"),
		CORSAllowedOrigins: vp.GetStringSlice("cors-allowed-origins"),
		TrustedProxyHeader: vp.GetString("trusted-proxy-header"),
	}
}

// Validate checks that the configuration is internally consistent.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return errors.New("addr must not be empty")
	}
	if c.RequestTimeout <= 0 {
		return errors.New("request-timeout must be positive")
	}
	if c.ShutdownGrace <= 0 {
		return errors.New("shutdown-grace must be positive")
	}
	if c.LogFormat != "json" && c.LogFormat != "text" {
		return fmt.Errorf("log-format must be %q or %q, got %q", "json", "text", c.LogFormat)
	}

	return nil
}

func setDefaults(vp *viper.Viper) {
	vp.SetDefault("addr", defaultAddr)
	vp.SetDefault("read-timeout", defaultReadTimeout)
	vp.SetDefault("read-header-timeout", defaultReadHeaderTimeout)
	vp.SetDefault("write-timeout", defaultWriteTimeout)
	vp.SetDefault("idle-timeout", defaultIdleTimeout)
	vp.SetDefault("request-timeout", defaultRequestTimeout)
	vp.SetDefault("shutdown-grace", defaultShutdownGrace)
	vp.SetDefault("log-level", defaultLogLevel)
	vp.SetDefault("log-format", defaultLogFormat)
	vp.SetDefault("cors-allowed-origins", []string{})
	vp.SetDefault("trusted-proxy-header", "")
}
