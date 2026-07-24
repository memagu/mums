package httpx

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

func MustGet[T any](c echo.Context, key string, middleware string) T {
	v, ok := c.Get(key).(T)
	if !ok {
		panic(fmt.Sprintf("%s is not set in context, was %s not applied?", key, middleware))
	}
	return v
}
