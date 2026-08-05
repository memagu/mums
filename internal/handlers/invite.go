package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/httpx"
)

var errUserAlreadyPhaddergruppMember = errors.New("user is already a member of phaddergrupp")

func GetInvite(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "token is required")
	}

	database := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)

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
	if err != nil {
		if errors.Is(err, errUserAlreadyPhaddergruppMember) {
			return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("User account %d is already a member of phaddergrupp %d", userAccountID, invite.PhaddergruppID))
		}
		return handleDBError(c, "phaddergrupp invite", err)
	}

	redirectURL := fmt.Sprintf("/phaddergrupp/%d", invite.PhaddergruppID)

	return httpx.Redirect(c, http.StatusSeeOther, redirectURL)
}
