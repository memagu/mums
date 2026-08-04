package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/handlers"
	"github.com/memagu/mums/internal/loaders"
	"github.com/memagu/mums/internal/roles"
	"github.com/memagu/mums/pkg/email"
)

func RegisterRoutes(e *echo.Echo, ss *auth.SessionStore, rts *auth.PasswordResetTokenStore, sender email.Sender) {
	e.Use(auth.SessionMiddleware(ss))

	e.GET("/login", handlers.GetLogin)
	e.POST("/login", handlers.PostLogin(ss))
	e.GET("/register", handlers.GetRegister)
	e.POST("/register", handlers.PostRegister(ss))
	e.GET("/about", handlers.GetAbout)
	e.GET("/password-reset", handlers.GetPasswordReset)
	e.POST("/password-reset", handlers.PostPasswordReset(rts, sender))
	e.GET("/password-reset/:token", handlers.GetPasswordResetConfirm(rts))
	e.POST("/password-reset/:token", handlers.PostPasswordResetConfirm(ss, rts))
	e.GET("/static/manifest.webmanifest", handlers.GetManifest)
	e.GET("/sw.js", handlers.GetServiceWorker)

	protected := e.Group(
		"",
		auth.RequireSession(),
		auth.UserAccountRBACMiddleware(),
		loaders.UserProfileMiddleware(),
	)

	protected.GET("/", handlers.GetHome)
	protected.POST("/", handlers.PostHome)
	protected.POST("/logout", handlers.PostLogout(ss))
	protected.GET("/settings", handlers.GetAccountSettings)
	protected.PATCH("/settings", handlers.PatchAccountSettings(rts))
	protected.DELETE("/settings", handlers.DeleteAccount(ss, rts))
	protected.GET("/admin", handlers.GetAdmin, auth.RequireUserAccountRole(roles.Admin))
	protected.POST("/admin", handlers.PostAdmin, auth.RequireUserAccountRole(roles.Admin))
	protected.GET("/invite/:token", handlers.GetInvite)

	phaddergrupp := protected.Group(
		"/phaddergrupp/:id",
		auth.PhaddergruppRBACMiddleware(),
		loaders.PhaddergruppMiddleware(),
	)

	phaddergrupp.GET("", handlers.GetPhaddergrupp)
	phaddergrupp.DELETE("", handlers.DeletePhaddergrupp)
	phaddergrupp.GET("/event-stream", handlers.GetPhaddergruppEventStream)
	phaddergrupp.POST("/purchase-mums", handlers.PostPhaddergruppPurchaseMums)
	phaddergrupp.POST("/mumsa", handlers.PostPhaddergruppMumsa)
	phaddergrupp.POST("/kick", handlers.PostPhaddergruppKick, auth.RequirePhaddergruppRole(roles.Phadder))
	phaddergrupp.POST("/mums/adjust", handlers.PostPhaddergruppMumsAdjust, auth.RequirePhaddergruppRole(roles.Phadder))
	phaddergrupp.GET("/settings", handlers.GetPhaddergruppSettings, auth.RequirePhaddergruppRole(roles.Phadder))
	phaddergrupp.PATCH("/settings", handlers.PatchPhaddergruppSettings, auth.RequirePhaddergruppRole(roles.Phadder))
	phaddergrupp.GET("/stats", handlers.GetPhaddergruppStats, auth.RequirePhaddergruppRole(roles.Phadder))
	phaddergrupp.GET("/audit", handlers.GetPhaddergruppAuditLog, auth.RequirePhaddergruppRole(roles.Phadder))
}
