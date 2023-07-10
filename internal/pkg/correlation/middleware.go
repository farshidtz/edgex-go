package correlation

import (
	"context"
	"net/url"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func ManageHeader(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		r := c.Request()
		correlationID := r.Header.Get(common.CorrelationHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		// lint:ignore SA1029 legacy
		// nolint:staticcheck // See golangci-lint #741
		ctx := context.WithValue(r.Context(), common.CorrelationHeader, correlationID)

		contentType := r.Header.Get(common.ContentType)
		// lint:ignore SA1029 legacy
		// nolint:staticcheck // See golangci-lint #741
		ctx = context.WithValue(ctx, common.ContentType, contentType)

		c.SetRequest(r.WithContext(ctx))

		return next(c)
	}
}

func LoggingMiddleware(lc logger.LoggingClient) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if lc.LogLevel() == models.TraceLog {
				r := c.Request()
				begin := time.Now()
				correlationId := FromContext(r.Context())
				lc.Trace("Begin request", common.CorrelationHeader, correlationId, "path", r.URL.Path)
				next(c)
				lc.Trace("Response complete", common.CorrelationHeader, correlationId, "duration", time.Since(begin).String())
			}
			return nil
		}
	}
}

// UrlDecodeMiddleware decode the path variables
// After invoking the router.UseEncodedPath() func, the path variables needs to decode before passing to the controller
func UrlDecodeMiddleware(lc logger.LoggingClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			for k, v := range vars {
				unescape, err := url.PathUnescape(v)
				if err != nil {
					lc.Debugf("failed to decode the %s from the value %s", k, v)
					return
				}
				vars[k] = unescape
			}
			next.ServeHTTP(w, r)
		})
	}
}
