package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/pkg/password"
)

type registerPageData struct {
	IsLoggedIn        bool
	AllowedErrorCodes []int
	Name              string
	Email             string
	Errors            map[string][]string
}

func GetRegister(c echo.Context) error {
	pageData := registerPageData{
		IsLoggedIn:        auth.GetIsLoggedIn(c),
		AllowedErrorCodes: []int{http.StatusInternalServerError, http.StatusConflict, http.StatusBadRequest},
	}
	return c.Render(http.StatusOK, "register", pageData)
}

func PostRegister(ss *auth.SessionStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		userName := c.FormValue("name")
		userEmail := c.FormValue("email")
		userPassword := c.FormValue("password")
		userConfirmPassword := c.FormValue("confirm-password")

		unexpectedError := func() error {
			pageData := registerPageData{
				IsLoggedIn: false,
				Name:       userName,
				Email:      userEmail,
				Errors:     map[string][]string{"Generic": {"An unexpected error occurred. Please try again."}},
			}
			return c.Render(http.StatusInternalServerError, "register#form-fields", pageData)
		}

		fieldError := func(statusCode int, field string, errorMessages []string) error {
			pageData := registerPageData{
				IsLoggedIn: false,
				Name:       userName,
				Email:      userEmail,
				Errors:     map[string][]string{field: errorMessages},
			}
			return c.Render(statusCode, "register#form-fields", pageData)
		}

		database := db.GetDB(c)

		emailExists, err := database.ReadUserCredentialsExistsByEmail(database, userEmail)
		if err != nil {
			c.Logger().Errorf("Database error during email conflict check for email %s: %v", userEmail, err)
			return unexpectedError()
		}
		if emailExists {
			return fieldError(http.StatusConflict, "Email", []string{"Account with email already exists."})
		}

		if userPassword != userConfirmPassword {
			return fieldError(http.StatusBadRequest, "PasswordConfirm", []string{"Passwords do not match."})
		}
		hashword, err := password.HashSecure(userPassword)
		if err == bcrypt.ErrPasswordTooLong {
			return fieldError(http.StatusBadRequest, "PasswordConfirm", []string{"Passwords length exceeds 72."})
		}
		if err != nil {
			c.Logger().Errorf("Password could not be hashed: %v", err)
			return unexpectedError()
		}

		var userAccountID int64
		err = db.WithTx(database, func(e db.Execer) error {
			userCredentialsID, err := database.CreateUserCredentials(e, userEmail, hashword)
			if err != nil {
				return err
			}
			userProfileID, err := database.CreateUserProfile(e, userName)
			if err != nil {
				return err
			}
			userAccountID, err = database.CreateUserAccount(e, userCredentialsID, userProfileID)
			return err
		})
		if err != nil {
			c.Logger().Errorf("Database error during user creation: %v", err)
			return unexpectedError()
		}
		return loginUser(c, ss, userAccountID)
	}
}
