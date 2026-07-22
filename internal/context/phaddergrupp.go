package context

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
)

func InjectPhaddergrupp() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			phaddergruppID := auth.GetPhaddergruppID(c)
			database := db.GetDB(c)

			phaddergruppData, err := database.ReadPhaddergrupp(database, phaddergruppID)
			if err != nil {
				if err == sql.ErrNoRows {
					return echo.NewHTTPError(http.StatusNotFound, "Phaddergrupp not found")
				}
				c.Logger().Errorf("Database error during phaddergrupp read: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err))
			}
			c.Set(config.CTXKeyPhaddergrupp, phaddergruppData)

			return next(c)
		}
	}
}

func GetPhaddergrupp(c echo.Context) db.PhaddergruppData {
	phaddergruppData, ok := c.Get(config.CTXKeyPhaddergrupp).(db.PhaddergruppData)
	if !ok {
		panic("config.CTXKeyPhaddergrupp is not set in context, was InjectPhaddergrupp not applied?")
	}

	return phaddergruppData
}
