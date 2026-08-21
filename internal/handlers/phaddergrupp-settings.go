package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/config"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/loaders"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

type formSelectOption struct {
	Value string
	Label string
}

var recencyWindowOptions = []formSelectOption{
	{Value: "1", Label: "1h"},
	{Value: "2", Label: "2h"},
	{Value: "3", Label: "3h"},
	{Value: "4", Label: "4h"},
}

var pauseDurationOptions = []formSelectOption{
	{Value: "15", Label: "15m"},
	{Value: "30", Label: "30m"},
	{Value: "45", Label: "45m"},
	{Value: "60", Label: "1h"},
}

type phaddergruppSettingsTemplateData struct {
	basePageData
	PhaddergruppID int64
	db.PhaddergruppData
	RecencyWindowOptions        []formSelectOption
	PauseDurationOptions        []formSelectOption
	Members                     []db.PhaddergruppUserSummary
	SwishRecipientNumberPattern string
	Errors                      map[string][]string
}

func GetPhaddergruppSettings(c echo.Context) error {
	conn := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)

	summaries, err := db.ReadPhaddergruppUserSummariesByPhaddergruppID(conn, phaddergruppID)
	if err != nil {
		return handleDBError(c, "phaddergrupp user summary read", err)
	}

	var members []db.PhaddergruppUserSummary
	members = append(members, summaries.N0llor...)
	members = append(members, summaries.Phaddrar...)

	templateData := phaddergruppSettingsTemplateData{
		basePageData: basePageData{
			IsLoggedIn:        auth.GetIsLoggedIn(c),
			AllowedErrorCodes: []int{http.StatusInternalServerError, http.StatusBadRequest},
			CSRFToken:         csrfToken(c),
		},
		PhaddergruppID:              auth.GetPhaddergruppID(c),
		PhaddergruppData:            loaders.GetPhaddergrupp(c),
		RecencyWindowOptions:        recencyWindowOptions,
		PauseDurationOptions:        pauseDurationOptions,
		Members:                     members,
		SwishRecipientNumberPattern: config.Swish.NumberPattern,
	}

	return c.Render(http.StatusOK, "phaddergrupp-settings", templateData)
}

func PatchPhaddergruppSettings(c echo.Context) error {
	conn := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)
	phaddergruppData := loaders.GetPhaddergrupp(c)

	updatedPhaddergruppData := phaddergruppData
	formErrors := make(map[string][]string)

	if strVal := c.FormValue("name"); strVal != "" {
		updatedPhaddergruppData.Name = strVal
	}
	if strVal := c.FormValue("primary-color"); strVal != "" {
		if !hexColorPattern.MatchString(strVal) {
			formErrors["PrimaryColor"] = []string{"Must be a valid hex color (e.g. #f280a1)"}
		} else {
			updatedPhaddergruppData.PrimaryColor = strVal
		}
	}
	if strVal := c.FormValue("secondary-color"); strVal != "" {
		if !hexColorPattern.MatchString(strVal) {
			formErrors["SecondaryColor"] = []string{"Must be a valid hex color (e.g. #f280a1)"}
		} else {
			updatedPhaddergruppData.SecondaryColor = strVal
		}
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
		if !config.Swish.NumberPatternRegex.MatchString(strVal) {
			formErrors["SwishRecipientNumber"] = []string{"Must be a valid Swish number"}
		} else {
			updatedPhaddergruppData.SwishRecipientNumber = strVal
		}
	}
	if strVal := c.FormValue("mums-capacity-per-user"); strVal != "" {
		val, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			formErrors["MumsCapacityPerUser"] = []string{"Invalid integer"}
		} else {
			updatedPhaddergruppData.MumsCapacityPerUser = val
		}
	}
	if strVal := c.FormValue("mums-min-purchase-quantity"); strVal != "" {
		val, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			formErrors["MumsMinPurchaseQuantity"] = []string{"Invalid integer"}
		} else {
			updatedPhaddergruppData.MumsMinPurchaseQuantity = val
		}
	}
	if strVal := c.FormValue("mums-max-purchase-quantity"); strVal != "" {
		val, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			formErrors["MumsMaxPurchaseQuantity"] = []string{"Invalid integer"}
		} else {
			updatedPhaddergruppData.MumsMaxPurchaseQuantity = val
		}
	}
	if strVal := c.FormValue("mums-purchase-quantity-step"); strVal != "" {
		val, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			formErrors["MumsPurchaseQuantityStep"] = []string{"Invalid integer"}
		} else {
			updatedPhaddergruppData.MumsPurchaseQuantityStep = val
		}
	}
	if strVal := c.FormValue("mums-default-purchase-quantity"); strVal != "" {
		val, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			formErrors["MumsDefaultPurchaseQuantity"] = []string{"Invalid integer"}
		} else {
			updatedPhaddergruppData.MumsDefaultPurchaseQuantity = val
		}
	}
	if strVal := c.FormValue("mums-recency-window-hours"); strVal != "" {
		val, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			formErrors["MumsRecencyWindowHours"] = []string{"Invalid integer"}
		} else {
			updatedPhaddergruppData.MumsRecencyWindowDuration = time.Duration(val) * time.Hour
		}
	}
	if strVal := c.FormValue("mums-pause-duration"); strVal != "" {
		val, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			formErrors["MumsPauseDuration"] = []string{"Invalid integer"}
		} else {
			updatedPhaddergruppData.MumsPauseDuration = time.Duration(val) * time.Minute
		}
	}

	if updatedPhaddergruppData.MumsMinPurchaseQuantity < 1 {
		formErrors["MumsMinPurchaseQuantity"] = []string{"Must be at least 1"}
	}
	if updatedPhaddergruppData.MumsPurchaseQuantityStep < 1 {
		formErrors["MumsPurchaseQuantityStep"] = []string{"Must be at least 1"}
	}
	if updatedPhaddergruppData.MumsMaxPurchaseQuantity < updatedPhaddergruppData.MumsMinPurchaseQuantity {
		formErrors["MumsMaxPurchaseQuantity"] = []string{"Must be greater than or equal to the min purchase quantity"}
	}
	if updatedPhaddergruppData.MumsMaxPurchaseQuantity > updatedPhaddergruppData.MumsCapacityPerUser {
		formErrors["MumsMaxPurchaseQuantity"] = []string{"Must not exceed the mums capacity per user"}
	}
	if d := updatedPhaddergruppData.MumsDefaultPurchaseQuantity; d < updatedPhaddergruppData.MumsMinPurchaseQuantity ||
		d > updatedPhaddergruppData.MumsMaxPurchaseQuantity ||
		(updatedPhaddergruppData.MumsPurchaseQuantityStep >= 1 &&
			(d-updatedPhaddergruppData.MumsMinPurchaseQuantity)%updatedPhaddergruppData.MumsPurchaseQuantityStep != 0) {
		formErrors["MumsDefaultPurchaseQuantity"] = []string{"Must be within min-max and on the step size"}
	}
	if updatedPhaddergruppData.MumsRecencyWindowDuration < time.Hour || updatedPhaddergruppData.MumsRecencyWindowDuration > 4*time.Hour {
		formErrors["MumsRecencyWindowHours"] = []string{"Must be between 1 and 4 hours"}
	}
	if updatedPhaddergruppData.MumsPauseDuration < 15*time.Minute || updatedPhaddergruppData.MumsPauseDuration > 60*time.Minute {
		formErrors["MumsPauseDuration"] = []string{"Must be between 15 and 60 minutes"}
	}

	summaries, err := db.ReadPhaddergruppUserSummariesByPhaddergruppID(conn, phaddergruppID)
	if err != nil {
		return handleDBError(c, "phaddergrupp user summary read", err)
	}
	var members []db.PhaddergruppUserSummary
	members = append(members, summaries.N0llor...)
	members = append(members, summaries.Phaddrar...)

	templateData := phaddergruppSettingsTemplateData{
		PhaddergruppData:            updatedPhaddergruppData,
		RecencyWindowOptions:        recencyWindowOptions,
		PauseDurationOptions:        pauseDurationOptions,
		Members:                     members,
		SwishRecipientNumberPattern: config.Swish.NumberPattern,
		Errors:                      formErrors,
	}

	if len(formErrors) > 0 {
		return c.Render(http.StatusBadRequest, "phaddergrupp-settings#fragment-form-fields", templateData)
	}

	if err := db.UpdatePhaddergrupp(conn, phaddergruppID, updatedPhaddergruppData); err != nil {
		return handleDBError(c, "phaddergrupp update", err)
	}

	return c.Render(http.StatusOK, "phaddergrupp-settings#fragment-form-fields", templateData)
}
