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
	basePageData
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
		return handleDBError(c, "user phaddergrupp summaries read", err)
	}

	pageData := homePageData{
		basePageData: basePageData{
			IsLoggedIn:        auth.GetIsLoggedIn(c),
			AllowedErrorCodes: []int{http.StatusInternalServerError},
		},
		UserProfileName:                       userProfile.Name,
		UserPhaddergruppSummaries:             userPhaddergruppSummaries,
		HasMoreThanOneUserPhaddergruppSummary: len(userPhaddergruppSummaries) > 1,
		SwishRecipientNumberPattern:           config.Swish.NumberPattern,
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
		return c.Render(http.StatusInternalServerError, "home#fragment-form-fields", pageData)
	}

	database := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)

	var phaddergruppID int64
	err := db.WithTx(database, func(e db.Execer) error {
		var err error
		phaddergruppID, err = database.CreatePhaddergrupp(e, phaddergruppName, swishRecipientNumber)
		if err != nil {
			return err
		}
		err = database.CreatePhaddergruppMapping(e, userAccountID, phaddergruppID, roles.Phadder)
		if err != nil {
			return err
		}
		err = database.CreatePhaddergruppInvite(e, token.MustGenerateSecure(config.Auth.InviteTokenSize), phaddergruppID, roles.N0lla)
		if err != nil {
			return err
		}
		err = database.CreatePhaddergruppInvite(e, token.MustGenerateSecure(config.Auth.InviteTokenSize), phaddergruppID, roles.Phadder)
		return err
	})
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp creation: %v", err)
		return unexpectedFormError()
	}

	redirectURL := fmt.Sprintf("/phaddergrupp/%d", phaddergruppID)
	return httpx.Redirect(c, http.StatusSeeOther, redirectURL)
}
