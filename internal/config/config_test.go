package config

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	cfg := Load(viper.New())
	assert.Equal(t, defaultAddr, cfg.Addr)
	assert.Equal(t, defaultMetricsAddr, cfg.MetricsAddr)
	assert.Equal(t, defaultLogLevel, cfg.LogLevel)
	assert.Equal(t, defaultLogFormat, cfg.LogFormat)
	assert.Equal(t, defaultRequestTimeout, cfg.RequestTimeout)
	assert.Empty(t, cfg.CORSAllowedOrigins)
	assert.Empty(t, cfg.TrustedProxyHeader)
	assert.Equal(t, StoreMemory, cfg.Store)
	assert.Empty(t, cfg.DatabaseURL)
	assert.Zero(t, cfg.DBMaxConns)
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("TEMPLATE_GO_API_ADDR", ":9999")
	t.Setenv("TEMPLATE_GO_API_LOG_LEVEL", "debug")
	t.Setenv("TEMPLATE_GO_API_TRUSTED_PROXY_HEADER", "X-Real-IP")

	vp := viper.New()
	vp.SetEnvPrefix("TEMPLATE_GO_API")
	vp.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	vp.AutomaticEnv()

	cfg := Load(vp)
	assert.Equal(t, ":9999", cfg.Addr)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "X-Real-IP", cfg.TrustedProxyHeader)
}

func TestLoadCORSOriginsFromFlags(t *testing.T) {
	t.Parallel()

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterFlags(flags)
	require.NoError(t, flags.Set("cors-allowed-origins", "https://a.example,https://b.example"))

	vp := viper.New()
	require.NoError(t, vp.BindPFlags(flags))

	cfg := Load(vp)
	assert.Equal(t, []string{"https://a.example", "https://b.example"}, cfg.CORSAllowedOrigins)
}

func TestValidate(t *testing.T) {
	t.Parallel()

	base := Config{
		Addr:           ":8080",
		RequestTimeout: time.Second,
		ShutdownGrace:  time.Second,
		LogFormat:      "json",
		Store:          StoreMemory,
	}
	require.NoError(t, base.Validate())

	emptyAddr := base
	emptyAddr.Addr = ""
	require.Error(t, emptyAddr.Validate())

	badFormat := base
	badFormat.LogFormat = "xml"
	require.Error(t, badFormat.Validate())

	metricsSameAsAddr := base
	metricsSameAsAddr.MetricsAddr = base.Addr
	require.Error(t, metricsSameAsAddr.Validate())

	negativeTimeout := base
	negativeTimeout.RequestTimeout = -time.Second
	require.Error(t, negativeTimeout.Validate())

	unknownStore := base
	unknownStore.Store = "redis"
	require.Error(t, unknownStore.Validate())

	postgresNoURL := base
	postgresNoURL.Store = StorePostgres
	require.Error(t, postgresNoURL.Validate())

	postgresWithURL := base
	postgresWithURL.Store = StorePostgres
	postgresWithURL.DatabaseURL = "postgres://localhost:5432/app"
	require.NoError(t, postgresWithURL.Validate())
}
