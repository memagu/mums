package handlers

import (
	"fmt"
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
	IsLoggedIn        bool
	AllowedErrorCodes []int
	PhaddergruppID    int64
	IsPhadder         bool
	MumsAvailable     int64
	db.UserProfileData
	db.PhaddergruppData
	PhaddergruppUserSummaries db.PhaddergruppUserSummaries
	HasMumsAvailable          bool
	MumsCapacityReached       bool
	MumsPurchaseQuantities    []int
	InviteURLN0lla            string
	InviteURLPhadder          string
}

func mumsPurchaseQuantities(mumsAvailable, mumsCapacityPerUser int64) []int {
	remainingMumsCapacity := mumsCapacityPerUser - mumsAvailable
	maxQty := min(config.MumsMaxPurchaseQuantity, int(remainingMumsCapacity))

	var purchaseQuantities []int
	for qty := maxQty; qty >= 1; qty-- {
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
		c.Logger().Errorf("Database error during mums available read: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	phaddergruppUserSummaries, err := database.ReadPhaddergruppUserSummariesByPhaddergruppID(database, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp user summary read: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	purchaseQuantities := mumsPurchaseQuantities(mumsAvailable, phaddergruppData.MumsCapacityPerUser)

	inviteTokens, err := database.ReadPhaddergruppInviteTokensByPhaddergruppID(database, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during invite tokens read: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	inviteURLN0lla := config.PhaddergruppInviteURLBase + inviteTokens.N0lla
	inviteURLPhadder := config.PhaddergruppInviteURLBase + inviteTokens.Phadder

	pageData := phaddergruppPageData{
		IsLoggedIn:                auth.GetIsLoggedIn(c),
		AllowedErrorCodes:         []int{http.StatusInternalServerError},
		PhaddergruppID:            phaddergruppID,
		IsPhadder:                 phaddergruppRole == roles.Phadder,
		MumsAvailable:             mumsAvailable,
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
		c.Logger().Errorf("Database error during phaddergrupp deletion: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	return httpx.Redirect(c, http.StatusSeeOther, "/")
}
