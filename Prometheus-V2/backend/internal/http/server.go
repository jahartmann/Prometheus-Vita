package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/antigravity/prometheus-v2/internal/api"
	authmw "github.com/antigravity/prometheus-v2/internal/domain/auth"
	"github.com/antigravity/prometheus-v2/internal/platform/metrics"
	"github.com/antigravity/prometheus-v2/internal/web"
	"github.com/getkin/kin-openapi/openapi3"
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
	// Auth is the small surface server.go uses to delegate authentication
	// endpoints to the auth domain. May be nil during early bring-up
	// (existing tests) — in that case the auth methods on apiServer return
	// 501 instead of panicking on a nil dispatch.
	Auth AuthHandler
	// AuthSigner is the JWT signer used by the RequireAuth middleware to
	// verify access tokens on protected endpoints. When nil (existing tests
	// and early bring-up), protected routes are registered without the
	// middleware so they remain reachable.
	AuthSigner *authmw.JWTSigner
}

// AuthHandler is the narrow interface server.go calls to implement the
// generated auth endpoints. The auth domain's HTTPHandler satisfies this
// interface; keeping the surface here avoids importing the domain package
// from the http package and lets tests inject fakes.
type AuthHandler interface {
	Login(c echo.Context) error
	Refresh(c echo.Context) error
	Logout(c echo.Context) error
	GetMe(c echo.Context) error
}

// apiServer implements api.ServerInterface. The auth methods are forwarded
// to the injected AuthHandler; GetSystemHealth is implemented inline because
// it has no dependencies beyond the request context.
type apiServer struct {
	auth AuthHandler
}

func (a apiServer) GetSystemHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, api.SystemHealth{
		Status:    api.Ok,
		Version:   "0.1.0",
		RequestId: c.Response().Header().Get(echo.HeaderXRequestID),
	})
}

// notWired returns a 501 when an auth method is invoked without a configured
// AuthHandler. This keeps the existing tests (which build a server with
// Deps.Auth == nil) safe: they only hit /healthz, /readyz, and
// /api/v1/system/health, none of which touch auth — but if a future test
// accidentally hits an auth path we surface a clear error instead of nil
// pointer panic.
func notWired() error {
	return echo.NewHTTPError(http.StatusNotImplemented, "auth not wired")
}

func (a apiServer) Login(c echo.Context) error {
	if a.auth == nil {
		return notWired()
	}
	return a.auth.Login(c)
}

func (a apiServer) Refresh(c echo.Context) error {
	if a.auth == nil {
		return notWired()
	}
	return a.auth.Refresh(c)
}

func (a apiServer) Logout(c echo.Context) error {
	if a.auth == nil {
		return notWired()
	}
	return a.auth.Logout(c)
}

func (a apiServer) GetMe(c echo.Context) error {
	if a.auth == nil {
		return notWired()
	}
	return a.auth.GetMe(c)
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
	// kin-openapi's gorillamux router matches the full request URL against
	// spec paths plus the Servers prefix. Echo group middleware does not
	// strip the group prefix from c.Request().URL.Path, so we keep a single
	// pathless server entry "/api/v1" — gorilla will trim that prefix before
	// matching against /auth/login, /system/health, etc. Using a path-only
	// URL (no scheme/host) avoids Host-header validation.
	spec.Servers = openapi3.Servers{{URL: "/api/v1"}}
	v1 := e.Group("/api/v1")
	v1.Use(oapimw.OapiRequestValidatorWithOptions(spec, &oapimw.Options{SilenceServersWarning: true}))

	// Manual route registration replaces api.RegisterHandlersWithBaseURL so we
	// can split public auth endpoints (login/refresh/logout) from protected
	// ones (auth/me, system/health). The OAPI validator sits on the v1 group
	// and runs first; RequireAuth sits on the protected sub-group and runs
	// second — Echo applies group middleware in registration order.
	server := &apiServer{auth: deps.Auth}
	v1.POST("/auth/login", server.Login)
	v1.POST("/auth/refresh", server.Refresh)
	v1.POST("/auth/logout", server.Logout)

	if deps.AuthSigner != nil {
		protected := v1.Group("")
		protected.Use(authmw.RequireAuth(deps.AuthSigner))
		protected.GET("/auth/me", server.GetMe)
		protected.GET("/system/health", server.GetSystemHealth)
	} else {
		// Fallback for tests / early bring-up without a JWT signer: register
		// the routes without auth so they remain reachable.
		v1.GET("/auth/me", server.GetMe)
		v1.GET("/system/health", server.GetSystemHealth)
	}

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
