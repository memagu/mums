package handlers

import (
	"net/http"
	"strings"

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
}

func mumsPurchaseQuantities(mumsAvailable int64, pd db.PhaddergruppData) []int64 {
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

func phaddergruppPreviewRuns(database *db.DB, phaddergruppID int64) ([]transactionLogEntry, error) {
	rows, err := database.ReadPhaddergruppTransactions(database, phaddergruppID, 0, roles.N0lla, "")
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
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	runs, err := phaddergruppPreviewRuns(database, phaddergruppID)
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
	database := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)
	phaddergruppID := auth.GetPhaddergruppID(c)
	phaddergruppRole := auth.GetPhaddergruppRole(c)
	phaddergruppData := loaders.GetPhaddergrupp(c)

	mumsAvailable, err := database.ReadMumsAvailable(database, userAccountID, phaddergruppID)
	if err != nil {
		return handleDBError(c, "mums available read", err)
	}

	phaddergruppUserSummaries, err := database.ReadPhaddergruppUserSummariesByPhaddergruppID(database, phaddergruppID)
	if err != nil {
		return handleDBError(c, "phaddergrupp user summary read", err)
	}

	purchaseQuantities := mumsPurchaseQuantities(mumsAvailable, phaddergruppData)

	recentTransactions := []transactionLogEntry{}
	if phaddergruppRole == roles.Phadder {
		runs, err := phaddergruppPreviewRuns(database, phaddergruppID)
		if err != nil {
			return handleDBError(c, "phaddergrupp transaction read", err)
		}
		recentTransactions = runs
	}

	inviteTokens, err := database.ReadPhaddergruppInviteTokensByPhaddergruppID(database, phaddergruppID)
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
	}
	return c.Render(http.StatusOK, "phaddergrupp", pageData)
}

func DeletePhaddergrupp(c echo.Context) error {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	err := database.DeletePhaddergrupp(database, phaddergruppID)
	if err != nil {
		return handleDBError(c, "phaddergrupp deletion", err)
	}

	return httpx.Redirect(c, http.StatusSeeOther, "/")
}
