package handlers

import (
	"net/http"

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
