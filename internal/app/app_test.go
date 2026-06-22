package app_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/template-go-api/internal/app"
	"github.com/meigma/template-go-api/internal/config"
	"github.com/meigma/template-go-api/internal/observability"
)

func TestAppWiring(t *testing.T) {
	t.Parallel()

	cfg := config.Load(viper.New())
	logger := observability.NewLogger(io.Discard, slog.LevelError, "json")
	application := app.New(cfg, logger, "test")

	handler := application.Handler()
	require.NotNil(t, handler)

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, healthReq)
	assert.Equal(t, http.StatusOK, healthRec.Code)

	createReq := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"title":"x"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	assert.Equal(t, http.StatusCreated, createRec.Code)
}
