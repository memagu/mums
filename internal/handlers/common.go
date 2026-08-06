package handlers

import (
	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/pkg/httpx"
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
	return httpx.InternalError(c, op, err)
}
