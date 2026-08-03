package loaders

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/httpx"
)

const (
	ctxKeyUserProfile  = "userProfile"
	ctxKeyPhaddergrupp = "phaddergrupp"
)

func UserProfileMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userAccountID := auth.GetUserAccountID(c)
			database := db.GetDB(c)

			userProfileData, err := database.ReadUserProfileByUserAccountID(database, userAccountID)
			if err != nil {
				c.Logger().Errorf("Database error during user profile read: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "An unexpected error occurred. Please try again.")
			}
			c.Set(ctxKeyUserProfile, userProfileData)

			return next(c)
		}
	}
}

func GetUserProfile(c echo.Context) db.UserProfileData {
	return httpx.MustGet[db.UserProfileData](c, ctxKeyUserProfile, "UserProfileMiddleware")
}

func PhaddergruppMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			phaddergruppID := auth.GetPhaddergruppID(c)
			database := db.GetDB(c)

			phaddergruppData, err := database.ReadPhaddergrupp(database, phaddergruppID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return echo.NewHTTPError(http.StatusNotFound, "Phaddergrupp not found")
				}
				return httpx.InternalError(c, "phaddergrupp read", err)
			}
			c.Set(ctxKeyPhaddergrupp, phaddergruppData)

			return next(c)
		}
	}
}

func GetPhaddergrupp(c echo.Context) db.PhaddergruppData {
	return httpx.MustGet[db.PhaddergruppData](c, ctxKeyPhaddergrupp, "PhaddergruppMiddleware")
}
