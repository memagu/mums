package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/httpx"
)

func PostPhaddergruppKick(c echo.Context) error {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	userAccountID, err := httpx.QueryParamInt64(c, "user-account-id")
	if err != nil {
		return err
	}

	err = db.WithTx(database, func(dbtx db.DBTX) error {
		err := db.DeletePhaddergruppMapping(dbtx, userAccountID, phaddergruppID)
		if err != nil {
			return err
		}
		phaddergruppIsEmpty, err := db.ReadPhaddergruppIsEmpty(dbtx, phaddergruppID)
		if err != nil {
			return err
		}
		if phaddergruppIsEmpty {
			return db.DeletePhaddergrupp(dbtx, phaddergruppID)
		}
		return nil
	})
	if err != nil {
		return handleDBError(c, "phaddergrupp kick", err)
	}

	return c.NoContent(http.StatusOK)
}
