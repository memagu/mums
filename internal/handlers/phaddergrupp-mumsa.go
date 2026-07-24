package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
)

func PostPhaddergruppMumsa(c echo.Context) error {
	database := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	err := db.WithTx(database, func(e db.Execer) error {
		q := e.(db.Queryer)
		_, err := database.UpdateAdjustMumsAvailable(q, userAccountID, phaddergruppID, -1)
		if err != nil {
			return err
		}
		_, err = database.CreateMums(e, userAccountID, phaddergruppID, 1, db.Consumption)
		return err
	})
	if err != nil {
		return handleDBError(c, "mumsning", err)
	}

	return c.NoContent(http.StatusNoContent)
}
