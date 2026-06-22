package config

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	cfg := Load(viper.New())
	assert.Equal(t, defaultAddr, cfg.Addr)
	assert.Equal(t, defaultLogLevel, cfg.LogLevel)
	assert.Equal(t, defaultLogFormat, cfg.LogFormat)
	assert.Equal(t, defaultRequestTimeout, cfg.RequestTimeout)
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("TEMPLATE_GO_API_ADDR", ":9999")
	t.Setenv("TEMPLATE_GO_API_LOG_LEVEL", "debug")

	vp := viper.New()
	vp.SetEnvPrefix("TEMPLATE_GO_API")
	vp.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	vp.AutomaticEnv()

	cfg := Load(vp)
	assert.Equal(t, ":9999", cfg.Addr)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestValidate(t *testing.T) {
	t.Parallel()

	base := Config{
		Addr:           ":8080",
		RequestTimeout: time.Second,
		ShutdownGrace:  time.Second,
		LogFormat:      "json",
	}
	require.NoError(t, base.Validate())

	emptyAddr := base
	emptyAddr.Addr = ""
	require.Error(t, emptyAddr.Validate())

	badFormat := base
	badFormat.LogFormat = "xml"
	require.Error(t, badFormat.Validate())

	negativeTimeout := base
	negativeTimeout.RequestTimeout = -time.Second
	require.Error(t, negativeTimeout.Validate())
}
