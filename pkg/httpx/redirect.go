package httpx

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func Redirect(c echo.Context, statusCode int, url string) error {
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", url)
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(statusCode, url)
}
