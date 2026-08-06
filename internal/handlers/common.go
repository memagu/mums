package handlers

import (
	"net/url"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/config"
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

func clientTimeLocation(c echo.Context) *time.Location {
	cookie, err := c.Cookie(config.Auth.TZCookie)
	if err != nil {
		return time.Local
	}
	timeZone, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return time.Local
	}
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return time.Local
	}
	return location
}
