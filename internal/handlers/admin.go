package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func GetAdmin(c echo.Context) error {
	// TODO: implement admin panel
	return echo.NewHTTPError(http.StatusNotImplemented, "not implemented")
}

func PostAdmin(c echo.Context) error {
	// TODO: implement admin panel
	return echo.NewHTTPError(http.StatusNotImplemented, "not implemented")
}
