package handlers

import (
	"database/sql"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
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
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)
	phaddergruppData, err := database.ReadPhaddergrupp(database, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp read: %v", err)
		return nil
	}
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
	LastMumsaAt   sql.NullTime
}

func emitMumsAvailableBadgeUpdate(c echo.Context, eventData db.MumsAvailableUpdate) {
	database := db.GetDB(c)

	lastMumsaAt, err := database.ReadMemberLastMumsaAt(database, eventData.UserAccountID, eventData.PhaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during last mumsa read: %v", err)
		return
	}

	templateData := mumsAvailableBadgeTemplateData{
		UserAccountID: eventData.UserAccountID,
		DoOOB:         true,
		MumsAvailable: eventData.MumsAvailable,
		LastMumsaAt:   lastMumsaAt,
	}

	var sb strings.Builder
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp#fragment-mums-available-badge", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return
	}
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp#fragment-mums-recency", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return
	}

	_ = httpx.EmitSSE(c, "mums-available-badge-update", sb.String())
}

func emitPhaddergruppMemberListsUpdate(c echo.Context) {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	phaddergruppData, err := database.ReadPhaddergrupp(database, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp read: %v", err)
		return
	}
	summaries, err := database.ReadPhaddergruppUserSummariesByPhaddergruppID(database, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp user summary read: %v", err)
		return
	}

	templateData := phaddergruppPageData{
		PhaddergruppID:            phaddergruppID,
		PhaddergruppData:          phaddergruppData,
		PhaddergruppUserSummaries: summaries,
	}

	var sb strings.Builder
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp#fragment-member-lists", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return
	}

	_ = httpx.EmitSSE(c, "phaddergrupp-member-lists-update", sb.String())
}

func emitPhaddergruppHeaderUpdate(c echo.Context) {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	phaddergruppData, err := database.ReadPhaddergrupp(database, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp read: %v", err)
		return
	}

	templateData := phaddergruppPageData{
		PhaddergruppID:   phaddergruppID,
		PhaddergruppData: phaddergruppData,
	}

	var sb strings.Builder
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp#fragment-group-name", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return
	}

	_ = httpx.EmitSSE(c, "phaddergrupp-header-update", sb.String())
}

func handlePhaddergruppEvent(c echo.Context, event db.DBEvent) {
	phaddergruppID := auth.GetPhaddergruppID(c)

	if event.Type == db.DBDelete && event.Table == "phaddergrupps" {
		httpx.EmitSSE(c, "phaddergrupp-deleted", "")
		return
	}

	if event.Type == db.DBUpdate && event.Table == "phaddergrupps" {
		emitPhaddergruppHeaderUpdate(c)
		if auth.GetPhaddergruppRole(c) == roles.Phadder {
			emitPhaddergruppMemberListsUpdate(c)
		} else {
			database := db.GetDB(c)
			mumsAvailable, err := database.ReadMumsAvailable(database, auth.GetUserAccountID(c), phaddergruppID)
			if err != nil {
				c.Logger().Errorf("Database error during mums available read: %v", err)
			} else {
				emitMumsAvailableWidgetUpdate(c, db.MumsAvailableUpdate{
					UserAccountID:  auth.GetUserAccountID(c),
					PhaddergruppID: phaddergruppID,
					MumsAvailable:  mumsAvailable,
				})
			}
		}
		return
	}

	if event.Type == db.DBUpdate && event.Table == "user_profiles" {
		if auth.GetPhaddergruppRole(c) == roles.Phadder {
			emitPhaddergruppMemberListsUpdate(c)
			emitPhaddergruppStatsUpdate(c)
			emitPhaddergruppAuditUpdate(c)
			emitPhaddergruppPreviewUpdate(c)
		}
		return
	}

	if event.Table == "phaddergrupp_mappings" && (event.Type == db.DBCreate || event.Type == db.DBDelete) {
		mappingEvent, ok := event.Data.(db.PhaddergruppMappingEvent)
		if !ok || mappingEvent.PhaddergruppID != phaddergruppID {
			return
		}
		if event.Type == db.DBDelete && mappingEvent.UserAccountID == auth.GetUserAccountID(c) {
			httpx.EmitSSE(c, "phaddergrupp-kicked", "")
			return
		}
		if auth.GetPhaddergruppRole(c) == roles.Phadder {
			emitPhaddergruppMemberListsUpdate(c)
			emitPhaddergruppStatsUpdate(c)
			emitPhaddergruppAuditUpdate(c)
			emitPhaddergruppPreviewUpdate(c)
		}
		return
	}

	if event.Type != db.DBUpdate || event.Table != "phaddergrupp_mappings" {
		return
	}

	eventData, ok := event.Data.(db.MumsAvailableUpdate)
	if !ok {
		return
	}

	if eventData.PhaddergruppID != phaddergruppID {
		return
	}

	if eventData.UserAccountID == auth.GetUserAccountID(c) {
		emitMumsAvailableWidgetUpdate(c, eventData)
	}

	if auth.GetPhaddergruppRole(c) == roles.Phadder {
		emitMumsAvailableBadgeUpdate(c, eventData)
		emitPhaddergruppStatsUpdate(c)
		emitPhaddergruppAuditUpdate(c)
		emitPhaddergruppPreviewUpdate(c)
	}
}

func GetPhaddergruppEventStream(c echo.Context) error {
	database := db.GetDB(c)

	httpx.SetupSSE(c)

	subID, events := database.Subscribe(16)
	defer database.Unsubscribe(subID)

	timer := time.NewTimer(config.Server.SSETimeout)
	defer timer.Stop()

	heartbeat := time.NewTicker(config.Server.SSEHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-timer.C:
			return nil
		case <-heartbeat.C:
			if err := httpx.EmitSSEHeartbeat(c); err != nil {
				c.Logger().Errorf("failed to write SSE heartbeat: %v", err)
				return nil
			}
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
