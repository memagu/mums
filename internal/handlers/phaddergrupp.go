package handlers

import (
	"fmt"
	"net/http"
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

type phaddergruppPageData struct {
	basePageData
	PhaddergruppID            int64
	IsPhadder                 bool
	MumsAvailable             int64
	HasMumsPurchaseQuantities bool
	RecentTransactions        []transactionLogEntry
	HasRecentTransactions     bool
	db.UserProfileData
	db.PhaddergruppData
	PhaddergruppUserSummaries db.PhaddergruppUserSummaries
	HasMumsAvailable          bool
	MumsCapacityReached       bool
	MumsPurchaseQuantities    []int64
	InviteURLN0lla            string
	InviteURLPhadder          string
	RecencyWindowMs           int64
	RecencyWindowLabel        string
}

func mumsaTimesAttr(times []time.Time) string {
	parts := make([]string, len(times))
	for i, t := range times {
		parts[i] = t.UTC().Format(time.RFC3339)
	}
	return strings.Join(parts, ",")
}

func mumsPurchaseQuantities(mumsAvailable int64, pd db.PhaddergruppData) []int64 {
	if pd.MumsPurchaseQuantityStep < 1 || pd.MumsMinPurchaseQuantity < 1 {
		return nil
	}
	remaining := pd.MumsCapacityPerUser - mumsAvailable
	capMax := min(pd.MumsMaxPurchaseQuantity, remaining)
	if capMax < pd.MumsMinPurchaseQuantity {
		return nil
	}
	largest := capMax - ((capMax - pd.MumsMinPurchaseQuantity) % pd.MumsPurchaseQuantityStep)

	var purchaseQuantities []int64
	for qty := largest; qty >= pd.MumsMinPurchaseQuantity; qty -= pd.MumsPurchaseQuantityStep {
		purchaseQuantities = append(purchaseQuantities, qty)
	}

	return purchaseQuantities
}

func phaddergruppPreviewRuns(conn *db.DB, phaddergruppID int64) ([]transactionLogEntry, error) {
	rows, err := db.ReadPhaddergruppTransactions(conn, phaddergruppID, 0, roles.N0lla, "")
	if err != nil {
		return nil, err
	}
	runs := rleTransactions(rows)
	if limit := config.Defaults.Mums.RecentTransactionRuns; limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func emitPhaddergruppPreviewUpdate(c echo.Context) {
	conn := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	runs, err := phaddergruppPreviewRuns(conn, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp transaction read: %v", err)
		return
	}

	templateData := phaddergruppPageData{
		RecentTransactions:    runs,
		HasRecentTransactions: len(runs) > 0,
	}

	var sb strings.Builder
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp#fragment-transaction-log", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return
	}

	_ = httpx.EmitSSE(c, "phaddergrupp-preview-update", sb.String())
}

func GetPhaddergrupp(c echo.Context) error {
	conn := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)
	phaddergruppID := auth.GetPhaddergruppID(c)
	phaddergruppRole := auth.GetPhaddergruppRole(c)
	phaddergruppData := loaders.GetPhaddergrupp(c)

	mumsAvailable, err := db.ReadMumsAvailable(conn, userAccountID, phaddergruppID)
	if err != nil {
		return handleDBError(c, "mums available read", err)
	}

	phaddergruppUserSummaries, err := db.ReadPhaddergruppUserSummariesByPhaddergruppID(conn, phaddergruppID)
	if err != nil {
		return handleDBError(c, "phaddergrupp user summary read", err)
	}

	since := time.Now().UTC().Add(-time.Duration(phaddergruppData.MumsRecencyWindowHours) * time.Hour)
	mumsaTimes, err := db.ReadPhaddergruppMumsaTimesSince(conn, phaddergruppID, since)
	if err != nil {
		return handleDBError(c, "phaddergrupp mumsa times read", err)
	}
	timesByUser := make(map[int64][]time.Time, len(mumsaTimes))
	for _, memberTime := range mumsaTimes {
		timesByUser[memberTime.UserAccountID] = append(timesByUser[memberTime.UserAccountID], memberTime.CreatedAt)
	}
	recencyWindowLabel := fmt.Sprintf("%dh", phaddergruppData.MumsRecencyWindowHours)
	attachMumsaCounts := func(summaries []db.PhaddergruppUserSummary) {
		for i := range summaries {
			times := timesByUser[summaries[i].UserAccountID]
			summaries[i].MumsaCount = len(times)
			summaries[i].MumsaTimesAttr = mumsaTimesAttr(times)
			summaries[i].MumsRecencyWindowLabel = recencyWindowLabel
		}
	}
	attachMumsaCounts(phaddergruppUserSummaries.N0llor)
	attachMumsaCounts(phaddergruppUserSummaries.Phaddrar)

	purchaseQuantities := mumsPurchaseQuantities(mumsAvailable, phaddergruppData)

	recentTransactions := []transactionLogEntry{}
	if phaddergruppRole == roles.Phadder {
		runs, err := phaddergruppPreviewRuns(conn, phaddergruppID)
		if err != nil {
			return handleDBError(c, "phaddergrupp transaction read", err)
		}
		recentTransactions = runs
	}

	inviteTokens, err := db.ReadPhaddergruppInviteTokensByPhaddergruppID(conn, phaddergruppID)
	if err != nil {
		return handleDBError(c, "invite tokens read", err)
	}

	inviteURLN0lla := config.Server.Origin + "/invite/" + inviteTokens.N0lla
	inviteURLPhadder := config.Server.Origin + "/invite/" + inviteTokens.Phadder

	pageData := phaddergruppPageData{
		basePageData: basePageData{
			IsLoggedIn:        auth.GetIsLoggedIn(c),
			AllowedErrorCodes: []int{http.StatusInternalServerError},
			CSRFToken:         csrfToken(c),
		},
		PhaddergruppID:            phaddergruppID,
		IsPhadder:                 phaddergruppRole == roles.Phadder,
		MumsAvailable:             mumsAvailable,
		HasMumsPurchaseQuantities: len(purchaseQuantities) > 0,
		RecentTransactions:        recentTransactions,
		HasRecentTransactions:     len(recentTransactions) > 0,
		UserProfileData:           loaders.GetUserProfile(c),
		PhaddergruppData:          loaders.GetPhaddergrupp(c),
		PhaddergruppUserSummaries: phaddergruppUserSummaries,
		HasMumsAvailable:          mumsAvailable > 0,
		MumsCapacityReached:       mumsAvailable >= phaddergruppData.MumsCapacityPerUser,
		MumsPurchaseQuantities:    purchaseQuantities,
		InviteURLN0lla:            inviteURLN0lla,
		InviteURLPhadder:          inviteURLPhadder,
		RecencyWindowMs:           phaddergruppData.MumsRecencyWindowHours * 3_600_000,
		RecencyWindowLabel:        recencyWindowLabel,
	}
	return c.Render(http.StatusOK, "phaddergrupp", pageData)
}

func DeletePhaddergrupp(c echo.Context) error {
	conn := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	err := db.DeletePhaddergrupp(conn, phaddergruppID)
	if err != nil {
		return handleDBError(c, "phaddergrupp deletion", err)
	}

	return httpx.Redirect(c, http.StatusSeeOther, "/")
}
