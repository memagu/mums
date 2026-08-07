package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/httpx"
)

var errUserNotPhaddergruppMember = errors.New("user is not a member of phaddergrupp")

func PostPhaddergruppMumsAdjust(c echo.Context) error {
	conn := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	userAccountID, err := httpx.QueryParamInt64(c, "user-account-id")
	if err != nil {
		return err
	}

	delta, err := httpx.QueryParamInt64(c, "delta")
	if err != nil {
		return err
	}
	if delta == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "delta must be non-zero")
	}

	err = db.WithTx(conn, func(dbtx db.DBTX) error {
		isMember, err := db.ReadUserAccountIsMemberOfPhaddergrupp(dbtx, userAccountID, phaddergruppID)
		if err != nil {
			return err
		}
		if !isMember {
			return errUserNotPhaddergruppMember
		}
		_, err = db.UpdateAdjustMumsAvailable(dbtx, userAccountID, phaddergruppID, delta)
		if err != nil {
			return err
		}
		_, err = db.CreateMums(dbtx, userAccountID, phaddergruppID, delta, db.Purchase)
		return err
	})
	if err != nil {
		if errors.Is(err, errUserNotPhaddergruppMember) {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("User account %d is not a member of phaddergrupp %d", userAccountID, phaddergruppID))
		}
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "Too large negative adjustment or user not member in phaddergrupp")
		}
		return handleDBError(c, "mums available adjustment", err)
	}

	return c.NoContent(http.StatusNoContent)
}
