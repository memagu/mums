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
	basePageData
	Name   string
	Email  string
	Errors map[string][]string
}

func GetRegister(c echo.Context) error {
	pageData := registerPageData{
		basePageData: basePageData{
			IsLoggedIn:        auth.GetIsLoggedIn(c),
			AllowedErrorCodes: []int{http.StatusInternalServerError, http.StatusConflict, http.StatusBadRequest},
			CSRFToken:         csrfToken(c),
		},
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
				basePageData: basePageData{IsLoggedIn: false},
				Name:         userName,
				Email:        userEmail,
				Errors:       map[string][]string{"Generic": {"An unexpected error occurred. Please try again."}},
			}
			return c.Render(http.StatusInternalServerError, "register#fragment-form-fields", pageData)
		}

		fieldError := func(statusCode int, field string, errorMessages []string) error {
			pageData := registerPageData{
				basePageData: basePageData{IsLoggedIn: false},
				Name:         userName,
				Email:        userEmail,
				Errors:       map[string][]string{field: errorMessages},
			}
			return c.Render(statusCode, "register#fragment-form-fields", pageData)
		}

		conn := db.GetDB(c)

		emailExists, err := db.ReadUserCredentialsExistsByEmail(conn, userEmail)
		if err != nil {
			c.Logger().Errorf("Database error during email conflict check for email %s: %v", userEmail, err)
			return unexpectedError()
		}
		if emailExists {
			return fieldError(http.StatusConflict, "Email", []string{"Account with email already exists."})
		}

		if userPassword == "" {
			return fieldError(http.StatusBadRequest, "PasswordConfirm", []string{"Password is required."})
		}
		if userPassword != userConfirmPassword {
			return fieldError(http.StatusBadRequest, "PasswordConfirm", []string{"Passwords do not match."})
		}
		hashword, err := password.HashSecure(userPassword)
		if err == bcrypt.ErrPasswordTooLong {
			return fieldError(http.StatusBadRequest, "PasswordConfirm", []string{"Password length exceeds 72."})
		}
		if err != nil {
			c.Logger().Errorf("Password could not be hashed: %v", err)
			return unexpectedError()
		}

		var userAccountID int64
		err = db.WithTx(conn, func(dbtx db.DBTX) error {
			userCredentialsID, err := db.CreateUserCredentials(dbtx, userEmail, hashword)
			if err != nil {
				return err
			}
			userProfileID, err := db.CreateUserProfile(dbtx, userName)
			if err != nil {
				return err
			}
			userAccountID, err = db.CreateUserAccount(dbtx, userCredentialsID, userProfileID)
			return err
		})
		if err != nil {
			c.Logger().Errorf("Database error during user creation: %v", err)
			return unexpectedError()
		}
		return loginUser(c, ss, userAccountID)
	}
}
