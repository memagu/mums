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

	tx, err := database.Begin()
	if err != nil {
		c.Logger().Errorf("Failed to begin transaction during phaddergrupp kick: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	defer tx.Rollback()
	err = database.DeletePhaddergruppMapping(tx, userAccountID, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp kick of user %d in phaddergrupp %d: %v", userAccountID, phaddergruppID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	phaddergruppIsEmpty, err := database.ReadPhaddergruppIsEmpty(tx, phaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp empty read of phaddergrupp %d during phaddergrupp kick: %v", phaddergruppID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	if phaddergruppIsEmpty {
		err := database.DeletePhaddergrupp(tx, phaddergruppID) 
		if err != nil {
			c.Logger().Errorf("Database error during phaddergrupp %d deletion during phaddergrupp kick: %v", phaddergruppID, err)
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
		}
	}
	err = tx.Commit()
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp kick: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	return c.NoContent(http.StatusOK)
}
