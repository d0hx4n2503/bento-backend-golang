package http

import (
	"fmt"
	"net/http"

	"github.com/d0hx4n2503/bento-backend-golang/docs"
	"github.com/labstack/echo/v5"
)

func mapSwaggerRoutes(e *echo.Echo) error {
	ui, err := docs.Files.ReadFile("swagger/index.html")
	if err != nil {
		return fmt.Errorf("read embedded Swagger UI: %w", err)
	}
	spec, err := docs.Files.ReadFile("openapi/openapi.json")
	if err != nil {
		return fmt.Errorf("read embedded OpenAPI document: %w", err)
	}
	serveUI := func(c *echo.Context) error {
		return c.Blob(http.StatusOK, "text/html; charset=utf-8", ui)
	}
	serveSpec := func(c *echo.Context) error {
		return c.Blob(http.StatusOK, "application/json; charset=utf-8", spec)
	}
	e.GET("/swagger", func(c *echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/swagger/")
	})
	e.GET("/swagger/", serveUI)
	e.GET("/swagger/index.html", serveUI)
	e.GET("/swagger/openapi.json", serveSpec)
	return nil
}
