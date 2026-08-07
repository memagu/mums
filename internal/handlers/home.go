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
	conn := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)
	userProfile := loaders.GetUserProfile(c)

	userPhaddergruppSummaries, err := db.ReadUserPhaddergruppSummariesByUserAccountID(conn, userAccountID)
	if err != nil {
		return handleDBError(c, "user phaddergrupp summaries read", err)
	}

	pageData := homePageData{
		basePageData: basePageData{
			IsLoggedIn:        auth.GetIsLoggedIn(c),
			AllowedErrorCodes: []int{http.StatusInternalServerError, http.StatusBadRequest},
			CSRFToken:         csrfToken(c),
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

	formErrors := make(map[string][]string)
	if phaddergruppName == "" {
		formErrors["PhaddergruppName"] = []string{"Phaddergrupp name is required."}
	}
	if swishRecipientNumber == "" {
		formErrors["SwishRecipientNumber"] = []string{"Swish recipient's number is required."}
	} else if !config.Swish.NumberPatternRegex.MatchString(swishRecipientNumber) {
		formErrors["SwishRecipientNumber"] = []string{"Must be a valid Swish number"}
	}
	if len(formErrors) > 0 {
		return c.Render(http.StatusBadRequest, "home#fragment-form-fields", homePageData{
			PhaddergruppName:            phaddergruppName,
			SwishRecipientNumber:        swishRecipientNumber,
			SwishRecipientNumberPattern: config.Swish.NumberPattern,
			Errors:                      formErrors,
		})
	}

	unexpectedFormError := func() error {
		pageData := homePageData{
			PhaddergruppName:            phaddergruppName,
			SwishRecipientNumber:        swishRecipientNumber,
			SwishRecipientNumberPattern: config.Swish.NumberPattern,
			Errors:                      map[string][]string{"Generic": {"An unexpected error occurred. Please try again."}},
		}
		return c.Render(http.StatusInternalServerError, "home#fragment-form-fields", pageData)
	}

	conn := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)

	var phaddergruppID int64
	err := db.WithTx(conn, func(dbtx db.DBTX) error {
		var err error
		phaddergruppID, err = db.CreatePhaddergrupp(dbtx, phaddergruppName, swishRecipientNumber)
		if err != nil {
			return err
		}
		err = db.CreatePhaddergruppMapping(dbtx, userAccountID, phaddergruppID, roles.Phadder)
		if err != nil {
			return err
		}
		err = db.CreatePhaddergruppInvite(dbtx, token.MustGenerateSecure(config.Auth.InviteTokenSize), phaddergruppID, roles.N0lla)
		if err != nil {
			return err
		}
		err = db.CreatePhaddergruppInvite(dbtx, token.MustGenerateSecure(config.Auth.InviteTokenSize), phaddergruppID, roles.Phadder)
		return err
	})
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp creation: %v", err)
		return unexpectedFormError()
	}

	redirectURL := fmt.Sprintf("/phaddergrupp/%d", phaddergruppID)
	return httpx.Redirect(c, http.StatusSeeOther, redirectURL)
}
