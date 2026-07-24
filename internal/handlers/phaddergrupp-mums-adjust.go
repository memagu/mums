package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
)

func PostPhaddergruppMumsAdjust(c echo.Context) error {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	userAccountIDStr := c.QueryParam("user-account-id")
	if userAccountIDStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Bad Request: user-account-id is required")
	}
	userAccountID, err := strconv.ParseInt(userAccountIDStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Bad Request user-account-id must be a valid integer")
	}
	
	deltaStr := c.QueryParam("delta")
	if deltaStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Bad Request: delta is required")
	}
	delta, err := strconv.ParseInt(deltaStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "delta must be a valid integer")
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
		c.Logger().Errorf("Database error during mums available adjustment for user %d in phaddergrupp %d: %v", userAccountID, phaddergruppID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	return c.NoContent(http.StatusNoContent)
}
