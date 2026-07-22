package context

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
)

func InjectUserProfile() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userAccountID := auth.GetUserAccountID(c)
			database := db.GetDB(c)

			userProfileData, err := database.ReadUserProfileByUserAccountID(database, userAccountID)
			if err != nil {
				c.Logger().Errorf("Database error during user profile read: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
			}
			c.Set(config.CTXKeyUserProfile, userProfileData)

			return next(c)
		}
	}
}

func GetUserProfile(c echo.Context) db.UserProfileData {
	userProfileData, ok := c.Get(config.CTXKeyUserProfile).(db.UserProfileData)
	if !ok {
		panic("config.CTXKeyUserProfile is not set in context, was InjectUserProfile not applied?")
	}

	return userProfileData
}
