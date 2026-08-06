package httpx

import (
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
)

const ctxKeyClientTimeLocation = "clientTimeLocation"

func ClientTimeZoneMiddleware(cookieName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ctxKeyClientTimeLocation, resolveClientTimeLocation(c, cookieName))
			return next(c)
		}
	}
}

func GetClientTimeLocation(c echo.Context) *time.Location {
	if location, ok := c.Get(ctxKeyClientTimeLocation).(*time.Location); ok && location != nil {
		return location
	}
	return time.Local
}

func resolveClientTimeLocation(c echo.Context, cookieName string) *time.Location {
	cookie, err := c.Cookie(cookieName)
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
