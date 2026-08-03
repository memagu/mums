package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/loaders"
	"github.com/memagu/mums/pkg/httpx"
)

type phaddergruppStatsTemplateData struct {
	basePageData
	PhaddergruppID int64
	db.PhaddergruppData
	db.PhaddergruppStats
}

func GetPhaddergruppStats(c echo.Context) error {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	stats, err := database.ReadPhaddergruppStats(database, phaddergruppID)
	if err != nil {
		return handleDBError(c, "phaddergrupp stats read", err)
	}

	templateData := phaddergruppStatsTemplateData{
		basePageData: basePageData{
			IsLoggedIn:        auth.GetIsLoggedIn(c),
			AllowedErrorCodes: []int{http.StatusInternalServerError},
			CSRFToken:         csrfToken(c),
		},
		PhaddergruppID:    phaddergruppID,
		PhaddergruppData:  loaders.GetPhaddergrupp(c),
		PhaddergruppStats: stats,
	}
	return c.Render(http.StatusOK, "phaddergrupp-stats", templateData)
}

func emitPhaddergruppStatsUpdate(c echo.Context) {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	stats, err := database.ReadPhaddergruppStats(database, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp stats read: %v", err)
		return
	}

	templateData := phaddergruppStatsTemplateData{
		PhaddergruppID:    phaddergruppID,
		PhaddergruppData:  loaders.GetPhaddergrupp(c),
		PhaddergruppStats: stats,
	}

	var sb strings.Builder
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp-stats#fragment-phaddergrupp-stats", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return
	}

	_ = httpx.EmitSSE(c, "phaddergrupp-stats-update", sb.String())
}
