package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/roles"
	"github.com/memagu/mums/pkg/httpx"
)

const (
	ctxKeyUserAccountRoles = "userAccountRoles"
	ctxKeyIsSuperAdmin     = "isSuperAdmin"
	ctxKeyPhaddergruppID   = "phaddergruppID"
	ctxKeyPhaddergruppRole = "phaddergruppRole"
)

func UserAccountRBACMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			database := db.GetDB(c)
			userAccountRoles, err := database.ReadUserAccountRoles(database, GetUserAccountID(c))
			if err != nil {
				c.Logger().Errorf("Database error during user account roles read: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "An unexpected error occurred. Please try again.")
			}
			c.Set(ctxKeyUserAccountRoles, userAccountRoles)

			isSuperAdmin := slices.Contains(userAccountRoles, roles.SuperAdmin)
			c.Set(ctxKeyIsSuperAdmin, isSuperAdmin)

			return next(c)
		}
	}
}

func GetUserAccountRoles(c echo.Context) []roles.UserAccountRole {
	return httpx.MustGet[[]roles.UserAccountRole](c, ctxKeyUserAccountRoles, "UserAccountRBACMiddleware")
}

func GetIsSuperAdmin(c echo.Context) bool {
	return httpx.MustGet[bool](c, ctxKeyIsSuperAdmin, "UserAccountRBACMiddleware")
}

func RequireUserAccountRole(allowedUserAccountRoles ...roles.UserAccountRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if GetIsSuperAdmin(c) {
				return next(c)
			}

			for _, userAccountRole := range GetUserAccountRoles(c) {
				if slices.Contains(allowedUserAccountRoles, userAccountRole) {
					return next(c)
				}
			}

			return echo.NewHTTPError(http.StatusForbidden, "User is missing a required user account role")
		}
	}
}

func PhaddergruppRBACMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			phaddergruppIDString := c.Param("id")
			if phaddergruppIDString == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "Missing phaddergrupp-id parameter")
			}
			phaddergruppID, err := strconv.ParseInt(phaddergruppIDString, 10, 64)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid phaddergrupp-id")
			}
			database := db.GetDB(c)
			phaddergruppRole, err := database.ReadPhaddergruppRole(database, GetUserAccountID(c), phaddergruppID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return echo.NewHTTPError(http.StatusForbidden, "User account does not have access to this phaddergrupp")
				}
				return httpx.InternalError(c, "phaddergrupp role read", err)
			}

			c.Set(ctxKeyPhaddergruppID, phaddergruppID)
			c.Set(ctxKeyPhaddergruppRole, phaddergruppRole)

			return next(c)
		}
	}
}

func GetPhaddergruppID(c echo.Context) int64 {
	return httpx.MustGet[int64](c, ctxKeyPhaddergruppID, "PhaddergruppRBACMiddleware")
}

func GetPhaddergruppRole(c echo.Context) roles.PhaddergruppRole {
	return httpx.MustGet[roles.PhaddergruppRole](c, ctxKeyPhaddergruppRole, "PhaddergruppRBACMiddleware")
}

func RequirePhaddergruppRole(allowedPhaddergruppRoles ...roles.PhaddergruppRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if GetIsSuperAdmin(c) {
				return next(c)
			}

			if slices.Contains(allowedPhaddergruppRoles, GetPhaddergruppRole(c)) {
				return next(c)
			}

			return echo.NewHTTPError(http.StatusForbidden, "User is missing a required phaddergrupp role")
		}
	}
}
