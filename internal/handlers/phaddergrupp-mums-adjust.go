package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/httpx"
)

func PostPhaddergruppMumsAdjust(c echo.Context) error {
	database := db.GetDB(c)
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

	isNotMember := false
	err = db.WithTx(database, func(e db.Execer) error {
		q := e.(db.Queryer)
		isMember, err := database.ReadUserAccountIsMemberOfPhaddergrupp(q, userAccountID, phaddergruppID)
		if err != nil {
			return err
		}
		if !isMember {
			isNotMember = true
			return fmt.Errorf("user %d is not a member of phaddergrupp %d", userAccountID, phaddergruppID)
		}
		_, err = database.UpdateAdjustMumsAvailable(q, userAccountID, phaddergruppID, delta)
		if err != nil {
			return err
		}
		_, err = database.CreateMums(e, userAccountID, phaddergruppID, delta, db.Purchase)
		return err
	})
	if err != nil {
		if isNotMember {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bad Request: User account %d is not a member of phaddergrupp %d", userAccountID, phaddergruppID))
		}
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "Too large negative adjustment or user not member in phaddergrupp")
		}
		return handleDBError(c, "mums available adjustment", err)
	}

	return c.NoContent(http.StatusNoContent)
}
