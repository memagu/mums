package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
)

func PostPhaddergruppMumsa(c echo.Context) error {
	conn := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	err := db.WithTx(conn, func(dbtx db.DBTX) error {
		_, err := db.UpdateAdjustMumsAvailable(dbtx, userAccountID, phaddergruppID, -1)
		if err != nil {
			return err
		}
		_, err = db.CreateMums(dbtx, userAccountID, phaddergruppID, 1, db.Consumption)
		return err
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "No mums available")
		}
		return handleDBError(c, "mumsning", err)
	}

	return c.NoContent(http.StatusNoContent)
}
