package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/loaders"
)

type phaddergruppSettingsTemplateData struct {
	basePageData
	PhaddergruppID int64
	db.PhaddergruppData
	SwishRecipientNumberPattern string
	Errors                      map[string][]string
}

func GetPhaddergruppSettings(c echo.Context) error {
	templateData := phaddergruppSettingsTemplateData{
		basePageData: basePageData{
			IsLoggedIn:        auth.GetIsLoggedIn(c),
			AllowedErrorCodes: []int{http.StatusInternalServerError, http.StatusBadRequest},
		},
		PhaddergruppID:              auth.GetPhaddergruppID(c),
		PhaddergruppData:            loaders.GetPhaddergrupp(c),
		SwishRecipientNumberPattern: config.SwishRecipientNumberPattern,
	}

	return c.Render(http.StatusOK, "phaddergrupp-settings", templateData)
}

func PatchPhaddergruppSettings(c echo.Context) error {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)
	phaddergruppData := loaders.GetPhaddergrupp(c)

	updatedPhaddergruppData := phaddergruppData
	formErrors := make(map[string][]string)

	// TODO: Validate more!
	if strVal := c.FormValue("name"); strVal != "" {
		updatedPhaddergruppData.Name = strVal
	}
	if strVal := c.FormValue("primary-color"); strVal != "" {
		updatedPhaddergruppData.PrimaryColor = strVal
	}
	if strVal := c.FormValue("secondary-color"); strVal != "" {
		updatedPhaddergruppData.SecondaryColor = strVal
	}
	if strVal := c.FormValue("mums-price-n0lla"); strVal != "" {
		val, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			formErrors["MumsPriceN0lla"] = []string{"Invalid float"}
		} else {
			updatedPhaddergruppData.MumsPriceN0lla = val
		}
	}
	if strVal := c.FormValue("mums-price-phadder"); strVal != "" {
		val, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			formErrors["MumsPricePhadder"] = []string{"Invalid float"}
		} else {
			updatedPhaddergruppData.MumsPricePhadder = val
		}
	}
	if strVal := c.FormValue("swish-recipient-number"); strVal != "" {
		updatedPhaddergruppData.SwishRecipientNumber = strVal
	}
	if strVal := c.FormValue("mums-capacity-per-user"); strVal != "" {
		val, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			formErrors["MumsCapacityPerUser"] = []string{"Invalid integer"}
		} else {
			updatedPhaddergruppData.MumsCapacityPerUser = val
		}
	}

	err := database.UpdatePhaddergrupp(database, phaddergruppID, updatedPhaddergruppData)
	if err != nil {
		return handleDBError(c, "phaddergrupp update", err)
	}

	templateData := phaddergruppSettingsTemplateData{
		PhaddergruppData:            updatedPhaddergruppData,
		SwishRecipientNumberPattern: config.SwishRecipientNumberPattern,
		Errors:                      formErrors,
	}

	var statusCode int
	if len(formErrors) == 0 {
		statusCode = http.StatusOK
	} else {
		statusCode = http.StatusBadRequest
	}

	return c.Render(statusCode, "phaddergrupp-settings#form-fields", templateData)
}
