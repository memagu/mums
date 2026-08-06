package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/httpx"
)

var errUserAlreadyPhaddergruppMember = errors.New("user is already a member of phaddergrupp")

func setPendingInviteCookie(c echo.Context, token string) {
	httpx.SetCookie(c, config.Auth.PendingInviteCookie, token, int(config.Auth.PendingInviteTTL.Seconds()), config.Server.CookieSecure)
}

func clearPendingInviteCookie(c echo.Context) {
	httpx.ClearCookie(c, config.Auth.PendingInviteCookie, config.Server.CookieSecure)
}

func joinPhaddergruppInvite(database *db.DB, userAccountID int64, token string) (string, error) {
	var invite db.PhaddergruppInviteData
	err := db.WithTx(database, func(e db.Execer) error {
		q := e.(db.Queryer)
		var err error
		invite, err = database.ReadPhaddergruppInvite(q, token)
		if err != nil {
			return err
		}
		userIsAlreadyPhaddergruppMember, err := database.ReadUserAccountIsMemberOfPhaddergrupp(q, userAccountID, invite.PhaddergruppID)
		if err != nil {
			return err
		}
		if userIsAlreadyPhaddergruppMember {
			return errUserAlreadyPhaddergruppMember
		}
		return database.CreatePhaddergruppMapping(e, userAccountID, invite.PhaddergruppID, invite.PhaddergruppRole)
	})
	if err != nil && !errors.Is(err, errUserAlreadyPhaddergruppMember) {
		return "", err
	}

	return fmt.Sprintf("/phaddergrupp/%d", invite.PhaddergruppID), nil
}

func handleInviteError(c echo.Context, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.Redirect(c, http.StatusSeeOther, "/")
	}
	return handleDBError(c, "phaddergrupp invite", err)
}

func GetInvite(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "token is required")
	}

	database := db.GetDB(c)

	if !auth.GetIsLoggedIn(c) {
		if _, err := database.ReadPhaddergruppInvite(database, token); err != nil {
			return handleInviteError(c, err)
		}
		setPendingInviteCookie(c, token)
		return httpx.Redirect(c, http.StatusSeeOther, "/register")
	}

	redirectURL, err := joinPhaddergruppInvite(database, auth.GetUserAccountID(c), token)
	if err != nil {
		return handleInviteError(c, err)
	}

	return httpx.Redirect(c, http.StatusSeeOther, redirectURL)
}
