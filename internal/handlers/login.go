package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/httpx"
	"github.com/memagu/mums/pkg/password"
)

// Dummy hash compared against when a login email is unknown, so the bcrypt
// cost (and thus response timing) matches the existing-account path.
const dummyLoginHash = "$2a$10$L4e4vdRfPGB5jirdsZ/ggu5Ob7ODsug23O92oNmA9dp.jnVKUHWza"

type loginPageData struct {
	basePageData
	Email  string
	Errors map[string][]string
}

func GetLogin(c echo.Context) error {
	pageData := loginPageData{
		basePageData: basePageData{
			AllowedErrorCodes: []int{http.StatusUnauthorized, http.StatusInternalServerError},
			CSRFToken:         csrfToken(c),
		},
	}
	return c.Render(http.StatusOK, "login", pageData)
}

func loginUser(c echo.Context, ss *auth.SessionStore, userAccountID int64) error {
	auth.LoginUser(c, ss, userAccountID)

	conn := db.GetDB(c)

	if pendingInvite, err := c.Cookie(config.Auth.PendingInviteCookie); err == nil {
		redirectURL, joinErr := joinPhaddergruppInvite(conn, userAccountID, pendingInvite.Value)
		if joinErr != nil && !errors.Is(joinErr, sql.ErrNoRows) {
			return handleDBError(c, "phaddergrupp invite", joinErr)
		}
		clearPendingInviteCookie(c)
		if joinErr == nil {
			return httpx.Redirect(c, http.StatusSeeOther, redirectURL)
		}
	}

	var redirectURL string
	switch phaddergruppID, createdAt, err := db.ReadLastCreatedPhaddergruppIDByUserAccountID(conn, userAccountID); err {
	case nil:
		if time.Since(createdAt) < config.Auth.LoginRedirectMaxAge {
			redirectURL = fmt.Sprintf("/phaddergrupp/%d", phaddergruppID)
		} else {
			redirectURL = "/"
		}
	case sql.ErrNoRows:
		redirectURL = "/"
	default:
		return handleDBError(c, "last created phaddergrupp read", err)
	}

	return httpx.Redirect(c, http.StatusSeeOther, redirectURL)
}

func PostLogin(ss *auth.SessionStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userEmail := c.FormValue("email")
		userPassword := c.FormValue("password")

		unexpectedError := func() error {
			pageData := loginPageData{
				basePageData: basePageData{IsLoggedIn: false},
				Email:        userEmail,
				Errors:       map[string][]string{"Generic": {"An unexpected error occurred. Please try again."}},
			}
			return c.Render(http.StatusInternalServerError, "login#fragment-form-fields", pageData)
		}

		invalidCredentials := func() error {
			pageData := loginPageData{
				basePageData: basePageData{IsLoggedIn: false},
				Email:        userEmail,
				Errors:       map[string][]string{"Generic": {"Invalid email or password."}},
			}
			return c.Render(http.StatusUnauthorized, "login#fragment-form-fields", pageData)
		}

		conn := db.GetDB(c)

		userCredentialsID, hashword, err := db.ReadUserCredentialsIDAndHashwordByEmail(conn, userEmail)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			password.Check(userPassword, dummyLoginHash)
			return invalidCredentials()
		case err != nil:
			c.Logger().Errorf("Database error during login for email %s: %v", userEmail, err)
			return unexpectedError()
		case !password.Check(userPassword, hashword):
			return invalidCredentials()
		}

		userAccountID, err := db.ReadUserAccountIDByUserCredentialsID(conn, userCredentialsID)
		if err != nil {
			c.Logger().Errorf("CRITICAL: Credentials found (ID: %d) but no matching user account.", userCredentialsID)
			return unexpectedError()
		}

		return loginUser(c, ss, userAccountID)
	}
}
