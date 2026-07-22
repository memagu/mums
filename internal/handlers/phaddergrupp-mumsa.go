package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
)

func PostPhaddergruppMumsa(c echo.Context) error {
	database := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	tx, err := database.Begin()
	if err != nil {
		c.Logger().Errorf("Failed to begin transaction during mumsning: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	defer tx.Rollback()
	_, err = database.UpdateAdjustMumsAvailable(tx, userAccountID, phaddergruppID, -1)
	if err != nil {
		c.Logger().Errorf("Database error during mums available update for user %d in phaddergrupp %d: %v", userAccountID, phaddergruppID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	_, err = database.CreateMums(tx, userAccountID, phaddergruppID, 1, db.Consumption)
	if err != nil {
		c.Logger().Errorf("Database error during mums creation for user %d in phaddergrupp %d: %v", userAccountID, phaddergruppID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	err = tx.Commit()
	if err != nil {
		c.Logger().Errorf("Database error during mumsning: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	return c.NoContent(http.StatusNoContent)
}
