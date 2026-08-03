package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type basePageData struct {
	IsLoggedIn        bool
	AllowedErrorCodes []int
	CSRFToken         string
}

func csrfToken(c echo.Context) string {
	token, _ := c.Get("csrf").(string)
	return token
}

func handleDBError(c echo.Context, op string, err error) error {
	c.Logger().Errorf("Database error during %s: %v", op, err)
	return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
}
