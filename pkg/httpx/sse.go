package httpx

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func SetupSSE(c echo.Context) {
	header := c.Response().Header()

	header.Set(echo.HeaderContentType, "text/event-stream")
	header.Set(echo.HeaderCacheControl, "no-cache")
	header.Set(echo.HeaderConnection, "keep-alive")
}

func FormatSSE(eventName, payload string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("event: %s\n", eventName))

	scanner := bufio.NewScanner(strings.NewReader(payload))
	for scanner.Scan() {
		sb.WriteString(fmt.Sprintf("data: %s\n", scanner.Text()))
	}

	sb.WriteString("\n")

	return sb.String()
}

func EmitSSE(c echo.Context, eventName, payload string) error {
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		c.Logger().Error("streaming unsupported: http.ResponseWriter does not implement http.Flusher")
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming unsupported")
	}

	formatted := FormatSSE(eventName, payload)
	if _, err := fmt.Fprint(c.Response(), formatted); err != nil {
		c.Logger().Errorf("failed to write SSE event: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to write SSE event")
	}

	flusher.Flush()
	return nil
}

func EmitSSEHeartbeat(c echo.Context) error {
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		c.Logger().Error("streaming unsupported: http.ResponseWriter does not implement http.Flusher")
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming unsupported")
	}

	if _, err := c.Response().Write([]byte(": ping\n\n")); err != nil {
		return err
	}

	flusher.Flush()
	return nil
}
