package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/routes"
	"github.com/memagu/mums/internal/templates"
)

func main() {
	config.Load()

	e := echo.New()

	e.Server.ReadHeaderTimeout = config.Server.ReadHeaderTimeout
	e.Server.IdleTimeout = config.Server.IdleTimeout

	e.Use(middleware.RequestLogger())

	database, err := db.NewDB(config.DB.FilePath)
	if err != nil {
		panic(err)
	}
	e.Use(db.DBMiddleware(database))

	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		ContextKey:     "csrf",
		CookieName:     "_csrf",
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSecure:   config.Server.CookieSecure,
		CookieSameSite: http.SameSiteLaxMode,
		TokenLookup:    "header:HX-CSRF-Token",
	}))

	templates.LoadTemplates(e)

	ss := auth.NewSessionStore()
	routes.RegisterRoutes(e, ss)

	e.Static("/static", "web/static")

	e.Start(config.Server.Address)
}
