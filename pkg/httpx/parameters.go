package httpx

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func QueryParamInt64(c echo.Context, key string) (int64, error) {
	s := c.QueryParam(key)
	if s == "" {
		return 0, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bad Request: %s is required", key))
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bad Request: %s must be a valid integer", key))
	}
	return v, nil
}
