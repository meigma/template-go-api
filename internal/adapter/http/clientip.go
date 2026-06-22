package http

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// clientIPMiddleware resolves the client IP and stores it on the request context
// for chi's [middleware.GetClientIP]. When trustedProxyHeader is set it reads
// that proxy-overwritten header (for example X-Real-IP); otherwise it trusts
// only the direct TCP peer, which cannot be spoofed. It never reads
// X-Forwarded-For implicitly, and avoids the deprecated, spoofable
// [middleware.RealIP].
func clientIPMiddleware(trustedProxyHeader string) func(http.Handler) http.Handler {
	if trustedProxyHeader != "" {
		return middleware.ClientIPFromHeader(trustedProxyHeader)
	}

	return middleware.ClientIPFromRemoteAddr
}
