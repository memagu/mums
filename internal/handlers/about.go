package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
)

type aboutPageData struct {
	basePageData
}

func GetAbout(c echo.Context) error {
	pageData := aboutPageData{
		basePageData: basePageData{IsLoggedIn: auth.GetIsLoggedIn(c)},
	}
	return c.Render(http.StatusOK, "about", pageData)
}
