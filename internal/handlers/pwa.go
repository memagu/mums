package handlers

import (
	"github.com/labstack/echo/v4"
)

func GetManifest(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "application/manifest+json")
	return c.File("web/static/manifest.webmanifest")
}

func GetServiceWorker(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "application/javascript")
	return c.File("web/static/sw.js")
}
