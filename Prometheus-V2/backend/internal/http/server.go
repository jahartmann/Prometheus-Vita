package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/antigravity/prometheus-v2/internal/api"
	"github.com/antigravity/prometheus-v2/internal/platform/metrics"
	"github.com/antigravity/prometheus-v2/internal/web"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	oapimw "github.com/oapi-codegen/echo-middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metricsExcludedPaths skips probe and self-scrape endpoints so they don't
// pollute the http_request_duration_seconds histogram and don't recursively
// observe the /metrics serializer.
var metricsExcludedPaths = map[string]struct{}{
	"/healthz": {},
	"/readyz":  {},
	"/metrics": {},
}

type Deps struct {
	Logger  *slog.Logger
	DB      DBPinger
	Redis   RedisPinger
	Metrics *metrics.Registry
}

type apiServer struct{}

func (apiServer) GetSystemHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, api.SystemHealth{
		Status:    api.Ok,
		Version:   "0.1.0",
		RequestId: c.Response().Header().Get(echo.HeaderXRequestID),
	})
}

func NewServer(deps Deps) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())

	RegisterHealth(e, deps.DB, deps.Redis)

	if deps.Metrics != nil {
		// TODO(auth): once auth middleware lands, restrict /metrics to admin or
		// to a scrape-target IP allowlist. Today this endpoint exposes Go
		// runtime details and the application's request matrix without auth.
		e.GET("/metrics", echo.WrapHandler(promhttp.HandlerFor(deps.Metrics.Registry, promhttp.HandlerOpts{})))

		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				start := time.Now()
				err := next(c)
				route := c.Path()
				if _, skip := metricsExcludedPaths[route]; skip {
					return err
				}
				if route == "" {
					route = "unknown"
				}
				status := c.Response().Status
				deps.Metrics.HTTPRequestsTotal.WithLabelValues(c.Request().Method, route, strconv.Itoa(status)).Inc()
				deps.Metrics.HTTPRequestDuration.WithLabelValues(c.Request().Method, route).Observe(time.Since(start).Seconds())
				return err
			}
		})
	}

	spec, err := api.GetSwagger()
	if err != nil {
		deps.Logger.Error("openapi spec load failed", slog.Any("error", err))
		return e
	}
	// kin-openapi's request validator matches request URL against spec.Servers; clearing
	// it lets the validator work when mounted on an Echo sub-group (the group prefix is
	// already stripped before the middleware runs).
	spec.Servers = nil
	v1 := e.Group("/api/v1")
	v1.Use(oapimw.OapiRequestValidator(spec))
	api.RegisterHandlersWithBaseURL(e, &apiServer{}, "/api/v1")

	if err := web.RegisterStatic(e); err != nil {
		deps.Logger.Error("static asset registration failed", slog.Any("error", err))
	}

	return e
}

func ListenAndServe(ctx context.Context, e *echo.Echo, addr string, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server starting", slog.String("addr", addr))
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
