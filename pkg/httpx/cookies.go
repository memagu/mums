package httpx

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func SetCookie(c echo.Context, name, value string, maxAge int, secure bool) {
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = value
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.Secure = secure
	cookie.SameSite = http.SameSiteLaxMode
	cookie.MaxAge = maxAge
	c.SetCookie(cookie)
}

func ClearCookie(c echo.Context, name string, secure bool) {
	SetCookie(c, name, "", -1, secure)
}
