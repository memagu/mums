package httpx

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func InternalError(c echo.Context, op string, err error) error {
	c.Logger().Errorf("Database error during %s: %v", op, err)
	return echo.NewHTTPError(http.StatusInternalServerError, "An unexpected error occurred. Please try again.")
}
