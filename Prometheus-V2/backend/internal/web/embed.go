package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

//go:embed all:dist
var distFS embed.FS

func RegisterStatic(e *echo.Echo) error {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}
	staticHandler := http.FileServer(http.FS(sub))

	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path
		if strings.HasPrefix(path, "/api/") || path == "/healthz" || path == "/readyz" || path == "/metrics" {
			return echo.ErrNotFound
		}

		if !hasStaticFile(sub, strings.TrimPrefix(path, "/")) {
			c.Request().URL.Path = "/"
		}
		staticHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
	return nil
}

func hasStaticFile(fsys fs.FS, name string) bool {
	if name == "" {
		return true
	}
	_, err := fs.Stat(fsys, name)
	return err == nil
}
