package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/loaders"
	"github.com/memagu/mums/internal/roles"
	"github.com/memagu/mums/pkg/httpx"
)

func PostPhaddergruppPause(c echo.Context) error {
	conn := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	userAccountID, err := httpx.QueryParamInt64(c, "user-account-id")
	if err != nil {
		return err
	}

	phaddergruppData := loaders.GetPhaddergrupp(c)

	role, err := db.ReadPhaddergruppRole(conn, userAccountID, phaddergruppID)
	if errors.Is(err, sql.ErrNoRows) {
		return echo.NewHTTPError(http.StatusBadRequest, "User is not a member of this phaddergrupp")
	}
	if err != nil {
		return handleDBError(c, "phaddergrupp role read", err)
	}
	if role != roles.N0lla {
		return echo.NewHTTPError(http.StatusBadRequest, "Only n0llor can be paused")
	}

	pausedUntil := time.Now().UTC().Add(phaddergruppData.MumsPauseDuration)

	if err := db.PauseUser(conn, userAccountID, phaddergruppID, pausedUntil); err != nil {
		return handleDBError(c, "pause user", err)
	}

	return c.NoContent(http.StatusNoContent)
}

func PostPhaddergruppUnpause(c echo.Context) error {
	conn := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	userAccountID, err := httpx.QueryParamInt64(c, "user-account-id")
	if err != nil {
		return err
	}

	if err := db.UnpauseUser(conn, userAccountID, phaddergruppID); err != nil {
		return handleDBError(c, "unpause user", err)
	}

	return c.NoContent(http.StatusNoContent)
}

func PauseGuard() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			conn := db.GetDB(c)
			pausedUntil, err := db.ReadPauseStatus(conn, auth.GetUserAccountID(c), auth.GetPhaddergruppID(c))
			if err != nil {
				return handleDBError(c, "pause status read", err)
			}
			if pausedUntil.After(time.Now().UTC()) {
				return echo.NewHTTPError(http.StatusForbidden, "Account is paused")
			}
			return next(c)
		}
	}
}
