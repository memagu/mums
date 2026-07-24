package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
)

func PostPhaddergruppKick(c echo.Context) error {
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

	err = db.WithTx(database, func(e db.Execer) error {
		q := e.(db.Queryer)
		err := database.DeletePhaddergruppMapping(e, userAccountID, phaddergruppID)
		if err != nil {
			return err
		}
		phaddergruppIsEmpty, err := database.ReadPhaddergruppIsEmpty(q, phaddergruppID)
		if err != nil {
			return err
		}
		if phaddergruppIsEmpty {
			return database.DeletePhaddergrupp(e, phaddergruppID)
		}
		return nil
	})
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp kick: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	return c.NoContent(http.StatusOK)
}
