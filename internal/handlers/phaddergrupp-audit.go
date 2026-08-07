package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/loaders"
	"github.com/memagu/mums/pkg/httpx"
)

type phaddergruppAuditTemplateData struct {
	basePageData
	PhaddergruppID int64
	db.PhaddergruppData
	ClientLocation *time.Location
	Transactions   []transactionLogEntry
}

func GetPhaddergruppAuditLog(c echo.Context) error {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	rows, err := db.ReadPhaddergruppTransactions(database, phaddergruppID, 0, "", "")
	if err != nil {
		return handleDBError(c, "phaddergrupp transaction read", err)
	}

	templateData := phaddergruppAuditTemplateData{
		basePageData: basePageData{
			IsLoggedIn:        auth.GetIsLoggedIn(c),
			AllowedErrorCodes: []int{http.StatusInternalServerError},
			CSRFToken:         csrfToken(c),
		},
		PhaddergruppID:   phaddergruppID,
		PhaddergruppData: loaders.GetPhaddergrupp(c),
		ClientLocation:   httpx.GetClientTimeLocation(c),
		Transactions:     normalizeTransactions(rows),
	}
	return c.Render(http.StatusOK, "phaddergrupp-audit", templateData)
}

func emitPhaddergruppAuditUpdate(c echo.Context) {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	rows, err := db.ReadPhaddergruppTransactions(database, phaddergruppID, 0, "", "")
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp transaction read: %v", err)
		return
	}

	templateData := phaddergruppAuditTemplateData{
		PhaddergruppID:   phaddergruppID,
		PhaddergruppData: loaders.GetPhaddergrupp(c),
		ClientLocation:   httpx.GetClientTimeLocation(c),
		Transactions:     normalizeTransactions(rows),
	}

	var sb strings.Builder
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp-audit#fragment-audit-log", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return
	}

	_ = httpx.EmitSSE(c, "phaddergrupp-audit-update", sb.String())
}
