package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/httpx"
)

func PostPhaddergruppLeave(c echo.Context) error {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)
	userAccountID := auth.GetUserAccountID(c)

	err := db.WithTx(database, func(dbtx db.DBTX) error {
		err := database.DeletePhaddergruppMapping(dbtx, userAccountID, phaddergruppID)
		if err != nil {
			return err
		}
		phaddergruppIsEmpty, err := database.ReadPhaddergruppIsEmpty(dbtx, phaddergruppID)
		if err != nil {
			return err
		}
		if phaddergruppIsEmpty {
			return database.DeletePhaddergrupp(dbtx, phaddergruppID)
		}
		return nil
	})
	if err != nil {
		return handleDBError(c, "phaddergrupp leave", err)
	}

	return httpx.Redirect(c, http.StatusSeeOther, "/")
}
