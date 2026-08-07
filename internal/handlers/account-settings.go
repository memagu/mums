package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/loaders"
	"github.com/memagu/mums/pkg/httpx"
	"github.com/memagu/mums/pkg/password"
	"github.com/memagu/mums/pkg/token"
)

type accountSettingsPageData struct {
	basePageData
	Name   string
	Email  string
	Errors map[string][]string
}

func GetAccountSettings(c echo.Context) error {
	database := db.GetDB(c)
	userAccountID := auth.GetUserAccountID(c)

	credentials, err := db.ReadUserCredentialsByUserAccountID(database, userAccountID)
	if err != nil {
		return handleDBError(c, "account settings read", err)
	}

	pageData := accountSettingsPageData{
		basePageData: basePageData{
			IsLoggedIn:        auth.GetIsLoggedIn(c),
			AllowedErrorCodes: []int{http.StatusInternalServerError, http.StatusBadRequest},
			CSRFToken:         csrfToken(c),
		},
		Name:  loaders.GetUserProfile(c).Name,
		Email: credentials.Email,
	}
	return c.Render(http.StatusOK, "account-settings", pageData)
}

func PatchAccountSettings(rts *auth.PasswordResetTokenStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.FormValue("name")
		email := c.FormValue("email")
		currentPassword := c.FormValue("current-password")
		newPassword := c.FormValue("new-password")
		confirmPassword := c.FormValue("confirm-password")

		formErrors := make(map[string][]string)

		if name == "" {
			formErrors["Name"] = []string{"Name is required."}
		}
		if email == "" {
			formErrors["Email"] = []string{"Email address is required."}
		}
		if newPassword != confirmPassword {
			formErrors["PasswordConfirm"] = []string{"Passwords do not match."}
		}

		database := db.GetDB(c)
		userAccountID := auth.GetUserAccountID(c)

		credentials, err := db.ReadUserCredentialsByUserAccountID(database, userAccountID)
		if err != nil {
			return handleDBError(c, "account settings read", err)
		}

		nameChanged := name != loaders.GetUserProfile(c).Name
		emailChanged := email != credentials.Email
		passwordChanged := newPassword != ""
		changesPending := nameChanged || emailChanged || passwordChanged

		if changesPending {
			if currentPassword == "" {
				formErrors["CurrentPassword"] = []string{"Current password is required."}
			} else if !password.Check(currentPassword, credentials.Hashword) {
				formErrors["CurrentPassword"] = []string{"Current password is incorrect."}
			}
		}

		if emailChanged {
			_, _, err := db.ReadUserCredentialsIDAndHashwordByEmail(database, email)
			switch {
			case errors.Is(err, sql.ErrNoRows):
			case err != nil:
				return handleDBError(c, "email conflict check", err)
			default:
				formErrors["Email"] = []string{"Account with email already exists."}
			}
		}

		var hashword string
		if passwordChanged && len(formErrors) == 0 {
			hashword, err = password.HashSecure(newPassword)
			if err == bcrypt.ErrPasswordTooLong {
				formErrors["PasswordConfirm"] = []string{"Password length exceeds 72."}
			} else if err != nil {
				c.Logger().Errorf("Password could not be hashed: %v", err)
				return handleDBError(c, "account settings password hash", err)
			}
		}

		if len(formErrors) > 0 {
			return c.Render(http.StatusBadRequest, "account-settings#fragment-form-fields", accountSettingsPageData{
				Name:   name,
				Email:  email,
				Errors: formErrors,
			})
		}

		err = db.WithTx(database, func(dbtx db.DBTX) error {
			if nameChanged {
				if err := db.UpdateUserProfileName(dbtx, userAccountID, name); err != nil {
					return err
				}
			}
			if emailChanged {
				if err := db.UpdateUserCredentialsEmail(dbtx, credentials.ID, email); err != nil {
					return err
				}
			}
			if passwordChanged {
				if err := db.UpdateUserCredentialsHashword(dbtx, credentials.ID, hashword); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return handleDBError(c, "account settings update", err)
		}

		if passwordChanged {
			rts.DeleteByUserAccountID(userAccountID)
		}

		settingsMessage := "Settings saved."
		if !changesPending {
			settingsMessage = "No changes to save."
		}
		c.Response().Header().Set("X-Settings-Message", settingsMessage)

		return c.Render(http.StatusOK, "account-settings#fragment-form-fields", accountSettingsPageData{
			Name:  name,
			Email: email,
		})
	}
}

func DeleteAccount(ss *auth.SessionStore, rts *auth.PasswordResetTokenStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		database := db.GetDB(c)
		userAccountID := auth.GetUserAccountID(c)

		credentials, err := db.ReadUserCredentialsByUserAccountID(database, userAccountID)
		if err != nil {
			return handleDBError(c, "account read for deletion", err)
		}

		currentPassword := c.FormValue("current-password")
		invalidPassword := func(message string) error {
			return c.Render(http.StatusBadRequest, "account-settings#fragment-form-fields", accountSettingsPageData{
				Name:   loaders.GetUserProfile(c).Name,
				Email:  credentials.Email,
				Errors: map[string][]string{"CurrentPassword": {message}},
			})
		}
		if currentPassword == "" {
			return invalidPassword("Current password is required.")
		}
		if !password.Check(currentPassword, credentials.Hashword) {
			return invalidPassword("Current password is incorrect.")
		}

		anonymousEmail := fmt.Sprintf("deleted-%d-%s@invalid.local", userAccountID, token.MustGenerateSecure(16))
		err = db.WithTx(database, func(dbtx db.DBTX) error {
			if err := db.UpdateUserCredentialsEmail(dbtx, credentials.ID, anonymousEmail); err != nil {
				return err
			}
			if err := db.UpdateUserProfileName(dbtx, userAccountID, "Deleted user"); err != nil {
				return err
			}
			return db.DeleteUserAccount(dbtx, userAccountID)
		})
		if err != nil {
			return handleDBError(c, "account deletion", err)
		}

		ss.DeleteSessionsByUserAccountID(userAccountID)
		rts.DeleteByUserAccountID(userAccountID)
		auth.LogoutUser(c, ss)

		return httpx.Redirect(c, http.StatusSeeOther, "/login")
	}
}
