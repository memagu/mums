package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/httpx"
)

func GetInvite(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
	    return echo.NewHTTPError(http.StatusBadRequest, "token is required")
	}

	database := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)

	tx, err := database.Begin()
	if err != nil {
		c.Logger().Errorf("Failed to begin transaction during phaddergrupp invite: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	defer tx.Rollback()
	invite, err := database.ReadPhaddergruppInvite(tx, token)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp invite read: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	userIsAlreadyPhaddergruppMember, err := database.ReadUserAccountIsMemberOfPhaddergrupp(tx, userAccountID, invite.PhaddergruppID)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp membership check during phaddergrupp invite: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	if userIsAlreadyPhaddergruppMember {
		return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("User account %d is already a member of phaddergrupp %d", userAccountID, invite.PhaddergruppID))
	}
	err = database.CreatePhaddergruppMapping(tx, userAccountID, invite.PhaddergruppID, invite.PhaddergruppRole)
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp mapping creation during phaddergrupp invite: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}
	err = tx.Commit()
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp invite: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
	}

	redirectURL := fmt.Sprintf("/phaddergrupp/%d", invite.PhaddergruppID)

	return httpx.Redirect(c, http.StatusFound, redirectURL)
}
