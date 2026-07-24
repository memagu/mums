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
	"github.com/memagu/mums/pkg/token"
)

type homePageData struct {
	IsLoggedIn                            bool
	AllowedErrorCodes                     []int
	UserProfileName                       string
	UserPhaddergruppSummaries             []db.UserPhaddergruppSummary
	HasMoreThanOneUserPhaddergruppSummary bool
	PhaddergruppName                      string
	SwishRecipientNumber                  string
	SwishRecipientNumberPattern           string
	Errors                                map[string][]string
}

func GetHome(c echo.Context) error {
	database := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)
	userProfile := loaders.GetUserProfile(c)

	userPhaddergruppSummaries, err := database.ReadUserPhaddergruppSummariesByUserAccountID(database, userAccountID)
	if err != nil {
		c.Logger().Errorf("Database error user phaddergrupp summaries read for user %d: %v", userAccountID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	pageData := homePageData{
		IsLoggedIn:                            auth.GetIsLoggedIn(c),
		AllowedErrorCodes:                     []int{http.StatusInternalServerError},
		UserProfileName:                       userProfile.Name,
		UserPhaddergruppSummaries:             userPhaddergruppSummaries,
		HasMoreThanOneUserPhaddergruppSummary: len(userPhaddergruppSummaries) > 1,
		SwishRecipientNumberPattern:           config.SwishRecipientNumberPattern,
	}
	return c.Render(http.StatusOK, "home", pageData)
}

func PostHome(c echo.Context) error {
	phaddergruppName := c.FormValue("phaddergrupp-name")
	swishRecipientNumber := c.FormValue("swish-recipient-number")

	unexpectedFormError := func() error {
		pageData := homePageData{
			PhaddergruppName: phaddergruppName,
			Errors:           map[string][]string{"Generic": {"An unexpected error occurred. Please try again."}},
		}
		return c.Render(http.StatusInternalServerError, "home#form-fields", pageData)
	}

	database := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)

	tx, err := database.Begin()
	if err != nil {
		c.Logger().Errorf("Failed to begin transaction during phaddergrupp creation: %v", err)
		return unexpectedFormError()
	}
	defer tx.Rollback()
	phaddergruppID, err := database.CreatePhaddergrupp(database, phaddergruppName, swishRecipientNumber)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp creation: %v", err)
		return unexpectedFormError()
	}
	err = database.CreatePhaddergruppMapping(tx, userAccountID, phaddergruppID, roles.Phadder)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp mapping creation during phaddergrupp creation: %v", err)
		return unexpectedFormError()
	}
	err = database.CreatePhaddergruppInvite(tx, token.MustGenerateSecure(config.PhaddergruppInviteTokenSize), phaddergruppID, roles.N0lla)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp invite (roles.N0lla) creation during phaddergrupp creation: %v", err)
		return unexpectedFormError()
	}
	err = database.CreatePhaddergruppInvite(tx, token.MustGenerateSecure(config.PhaddergruppInviteTokenSize), phaddergruppID, roles.Phadder)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp invite (roles.Phadder) creation during phaddergrupp creation: %v", err)
		return unexpectedFormError()
	}
	err = tx.Commit()
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp creation: %v", err)
		return unexpectedFormError()
	}

	redirectURL := fmt.Sprintf("/phaddergrupp/%d", phaddergruppID)
	return httpx.Redirect(c, http.StatusSeeOther, redirectURL)
}
