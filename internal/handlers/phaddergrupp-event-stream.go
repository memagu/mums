package handlers

import (
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/loaders"
	"github.com/memagu/mums/internal/roles"
	"github.com/memagu/mums/pkg/httpx"
)

type templateDataMumsAvailableWidget struct {
	PhaddergruppID int64
	db.PhaddergruppData
	MumsAvailable             int64
	HasMumsAvailable          bool
	HasMumsPurchaseQuantities bool
	MumsCapacityReached       bool
	MumsPurchaseQuantities    []int64
}

func emitMumsAvailableWidgetUpdate(c echo.Context, eventData db.MumsAvailableUpdate) error {
	phaddergruppID := auth.GetPhaddergruppID(c)
	phaddergruppData := loaders.GetPhaddergrupp(c)
	purchaseQuantities := mumsPurchaseQuantities(eventData.MumsAvailable, phaddergruppData)

	templateData := templateDataMumsAvailableWidget{
		PhaddergruppID:            phaddergruppID,
		PhaddergruppData:          phaddergruppData,
		MumsAvailable:             eventData.MumsAvailable,
		HasMumsAvailable:          eventData.MumsAvailable > 0,
		HasMumsPurchaseQuantities: len(purchaseQuantities) > 0,
		MumsCapacityReached:       eventData.MumsAvailable >= phaddergruppData.MumsCapacityPerUser,
		MumsPurchaseQuantities:    purchaseQuantities,
	}

	var sb strings.Builder
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp#fragment-mums-available-widget", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return nil
	}

	return httpx.EmitSSE(c, "mums-available-widget-update", sb.String())
}

type mumsAvailableBadgeTemplateData struct {
	UserAccountID int64
	DoOOB         bool
	MumsAvailable int64
}

func emitMumsAvailableBadgeUpdate(c echo.Context, eventData db.MumsAvailableUpdate) {
	templateData := mumsAvailableBadgeTemplateData{
		UserAccountID: eventData.UserAccountID,
		DoOOB:         true,
		MumsAvailable: eventData.MumsAvailable,
	}

	var sb strings.Builder
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp#fragment-mums-available-badge", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return
	}

	_ = httpx.EmitSSE(c, "mums-available-badge-update", sb.String())
}

func handlePhaddergruppEvent(c echo.Context, event db.DBEvent) {
	if event.Type == db.DBDelete && event.Table == "phaddergrupps" {
		httpx.EmitSSE(c, "phaddergrupp-deleted", "")
		return
	}

	if event.Type != db.DBUpdate || event.Table != "phaddergrupp_mappings" {
		return
	}

	eventData, ok := event.Data.(db.MumsAvailableUpdate)
	if !ok {
		return
	}

	phaddergruppID := auth.GetPhaddergruppID(c)

	if eventData.PhaddergruppID != phaddergruppID {
		return
	}

	if eventData.UserAccountID == auth.GetUserAccountID(c) {
		emitMumsAvailableWidgetUpdate(c, eventData)
	}

	if auth.GetPhaddergruppRole(c) == roles.Phadder {
		emitMumsAvailableBadgeUpdate(c, eventData)
	}
}

func GetPhaddergruppEventStream(c echo.Context) error {
	database := db.GetDB(c)

	httpx.SetupSSE(c)

	subID, events := database.Subscribe(16)
	defer database.Unsubscribe(subID)

	timer := time.NewTimer(config.Server.SSETimeout)
	defer timer.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-timer.C:
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}

			handlePhaddergruppEvent(c, event)

			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(config.Server.SSETimeout)
		}
	}
}
