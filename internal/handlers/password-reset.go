package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/email"
	"github.com/memagu/mums/pkg/password"
)

type passwordResetPageData struct {
	basePageData
	Email    string
	Sent     bool
	Fallback bool
	Errors   map[string][]string
}

type passwordResetConfirmPageData struct {
	basePageData
	Token  string
	Valid  bool
	Errors map[string][]string
}

func GetPasswordReset(c echo.Context) error {
	pageData := passwordResetPageData{
		basePageData: basePageData{
			AllowedErrorCodes: []int{http.StatusInternalServerError, http.StatusBadRequest},
			CSRFToken:         csrfToken(c),
		},
	}
	return c.Render(http.StatusOK, "password-reset-request", pageData)
}

func PostPasswordReset(rts *auth.PasswordResetTokenStore, sender email.Sender) echo.HandlerFunc {
	return func(c echo.Context) error {
		userEmail := c.FormValue("email")

		requestSent := func() error {
			return c.Render(http.StatusOK, "password-reset-request#fragment-form-fields", passwordResetPageData{
				Email:    userEmail,
				Sent:     true,
				Fallback: config.SMTP.Host == "",
			})
		}

		if userEmail == "" {
			return c.Render(http.StatusBadRequest, "password-reset-request#fragment-form-fields", passwordResetPageData{
				Email:  userEmail,
				Errors: map[string][]string{"Email": {"Email address is required."}},
			})
		}

		database := db.GetDB(c)
		logger := c.Logger()

		userAccountID, err := database.ReadActiveUserAccountIDByEmail(database, userEmail)
		switch {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			c.Logger().Errorf("Database error during password reset request for email %s: %v", userEmail, err)
			return handleDBError(c, "password reset request read", err)
		default:
			resetToken := rts.Create(userAccountID)
			sentAt := time.Now()
			expiresAt := sentAt.Add(config.Auth.PasswordResetTTL)
			resetURL := config.Server.Origin + "/password-reset/" + resetToken
			subject, body := passwordResetEmail(resetURL, sentAt, expiresAt)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Errorf("panic in password reset goroutine: %v", r)
					}
				}()
				if err := sender.Send(userEmail, subject, body); err != nil {
					logger.Errorf("Password reset email to %s could not be sent: %v", userEmail, err)
				}
			}()
		}

		return requestSent()
	}
}

func GetPasswordResetConfirm(rts *auth.PasswordResetTokenStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		resetToken := c.Param("token")
		if resetToken == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "token is required")
		}

		_, ok := rts.Peek(resetToken)

		return c.Render(http.StatusOK, "password-reset-confirm", passwordResetConfirmPageData{
			basePageData: basePageData{
				AllowedErrorCodes: []int{http.StatusInternalServerError, http.StatusBadRequest},
				CSRFToken:         csrfToken(c),
			},
			Token: resetToken,
			Valid: ok,
		})
	}
}

func PostPasswordResetConfirm(ss *auth.SessionStore, rts *auth.PasswordResetTokenStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		resetToken := c.Param("token")
		if resetToken == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "token is required")
		}

		invalidToken := func() error {
			return c.Render(http.StatusBadRequest, "password-reset-confirm#fragment-form-fields", passwordResetConfirmPageData{
				Token: resetToken,
				Valid: false,
			})
		}

		fieldError := func(errorMessages []string) error {
			return c.Render(http.StatusBadRequest, "password-reset-confirm#fragment-form-fields", passwordResetConfirmPageData{
				Token:  resetToken,
				Valid:  true,
				Errors: map[string][]string{"PasswordConfirm": errorMessages},
			})
		}

		if _, ok := rts.Peek(resetToken); !ok {
			return invalidToken()
		}

		newPassword := c.FormValue("new-password")
		confirmPassword := c.FormValue("confirm-password")

		if newPassword == "" {
			return fieldError([]string{"Password is required."})
		}
		if newPassword != confirmPassword {
			return fieldError([]string{"Passwords do not match."})
		}

		hashword, err := password.HashSecure(newPassword)
		if err == bcrypt.ErrPasswordTooLong {
			return fieldError([]string{"Password length exceeds 72."})
		}
		if err != nil {
			c.Logger().Errorf("Password could not be hashed during reset: %v", err)
			return handleDBError(c, "password reset hash", err)
		}

		userAccountID, ok := rts.Consume(resetToken)
		if !ok {
			return invalidToken()
		}

		database := db.GetDB(c)

		credentials, err := database.ReadUserCredentialsByUserAccountID(database, userAccountID)
		if errors.Is(err, sql.ErrNoRows) {
			return invalidToken()
		}
		if err != nil {
			return handleDBError(c, "password reset credentials read", err)
		}

		err = db.WithTx(database, func(e db.Execer) error {
			return database.UpdateUserCredentialsHashword(e, credentials.ID, hashword)
		})
		if err != nil {
			return handleDBError(c, "password reset update", err)
		}

		rts.DeleteByUserAccountID(userAccountID)
		ss.DeleteSessionsByUserAccountID(userAccountID)

		return loginUser(c, ss, userAccountID)
	}
}

const passwordResetEmailSubject = "Reset your mums password"

const passwordResetEmailTimeFormat = "2006-01-02 15:04:05"

func passwordResetEmail(resetURL string, sentAt, expiresAt time.Time) (string, string) {
	body := "Sup dawg!\n" +
		"\n" +
		"A request to change your mums account password was made.\n" +
		"\n" +
		"Open this link to choose a new password:\n" +
		resetURL + "\n" +
		"\n" +
		"requested at: " + sentAt.Local().Format(passwordResetEmailTimeFormat) + "\n" +
		"expires at: " + expiresAt.Local().Format(passwordResetEmailTimeFormat) + "\n" +
		"\n" +
		"If you didn't request this, you can safely ignore this email."
	return passwordResetEmailSubject, body
}
