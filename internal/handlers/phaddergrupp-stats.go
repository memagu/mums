package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memagu/mums/internal/auth"
	"github.com/memagu/mums/internal/db"
	"github.com/memagu/mums/internal/loaders"
	"github.com/memagu/mums/internal/roles"
	"github.com/memagu/mums/pkg/httpx"
)

type statsRoleFilter struct {
	Value roles.PhaddergruppRole
	Label string
}

var statsRoleFilters = []statsRoleFilter{
	{Value: roles.Phadder, Label: "Phadder"},
	{Value: roles.N0lla, Label: "N0lla"},
	{Value: "", Label: "All"},
}

// weekdayLetters indexes Monday=0 (Sunday=6) for the chart's x-axis labels.
var weekdayLetters = [...]string{"m", "t", "w", "t", "f", "s", "s"}

type dailyMumsatBar struct {
	Day     string
	Label   string
	Count   int
	Percent int
	IsToday bool
}

type weekdayHourCell struct {
	Count     int
	Intensity int
}

type weekdayHourRow struct {
	Hour  int
	Cells [7]weekdayHourCell
}

type titleCard struct {
	Title       string
	Icon        string
	Description string
	Name        string
	Data        string
}

type phaddergruppStatsTemplateData struct {
	basePageData
	PhaddergruppID int64
	RoleFilter     roles.PhaddergruppRole
	RoleFilters    []statsRoleFilter
	db.PhaddergruppData
	db.PhaddergruppStats
	DailyBars       []dailyMumsatBar
	WeekdayHourRows []weekdayHourRow
	WeekdayLetters  []string
	TitleCards      []titleCard
}

func resolveStatsRole(c echo.Context) roles.PhaddergruppRole {
	switch role := roles.PhaddergruppRole(c.QueryParam("role")); role {
	case roles.Phadder, roles.N0lla:
		return role
	default:
		return ""
	}
}

func readPhaddergruppDailyBars(events []db.ConsumptionEvent, loc *time.Location) []dailyMumsatBar {
	const windowDays = 14

	today := time.Now().In(loc)
	days := make([]time.Time, 0, windowDays)
	dayStrs := make([]string, 0, windowDays)
	for i := windowDays - 1; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		days = append(days, day)
		dayStrs = append(dayStrs, day.Format(time.DateOnly))
	}

	counts := make(map[string]int, windowDays)
	for _, e := range events {
		counts[e.CreatedAt.In(loc).Format(time.DateOnly)]++
	}

	maxCount := 0
	for _, dayStr := range dayStrs {
		if counts[dayStr] > maxCount {
			maxCount = counts[dayStr]
		}
	}

	todayStr := today.Format(time.DateOnly)
	bars := make([]dailyMumsatBar, len(days))
	for i, day := range days {
		count := counts[dayStrs[i]]
		percent := 0
		if maxCount > 0 {
			percent = count * 100 / maxCount
		}
		if count > 0 && percent < 4 {
			percent = 4
		}
		bars[i] = dailyMumsatBar{
			Day:     dayStrs[i],
			Label:   weekdayLetters[(int(day.Weekday())+6)%7],
			Count:   count,
			Percent: percent,
			IsToday: dayStrs[i] == todayStr,
		}
	}
	return bars
}

func readPhaddergruppWeekdayHourRows(events []db.ConsumptionEvent, loc *time.Location) []weekdayHourRow {
	var counts [7][24]int
	for _, e := range events {
		t := e.CreatedAt.In(loc)
		counts[(int(t.Weekday())+6)%7][t.Hour()]++
	}

	maxCount := 0
	for weekday := 0; weekday < 7; weekday++ {
		for hour := 0; hour < 24; hour++ {
			if counts[weekday][hour] > maxCount {
				maxCount = counts[weekday][hour]
			}
		}
	}

	rows := make([]weekdayHourRow, 24)
	for hour := 0; hour < 24; hour++ {
		rows[hour].Hour = hour
		for weekday := 0; weekday < 7; weekday++ {
			count := counts[weekday][hour]
			intensity := 0
			if maxCount > 0 && count > 0 {
				intensity = count * 100 / maxCount
			}
			rows[hour].Cells[weekday] = weekdayHourCell{Count: count, Intensity: intensity}
		}
	}
	return rows
}

func readTitleCards(events []db.ConsumptionEvent, members []db.MemberMumsStats, loc *time.Location) []titleCard {
	if len(events) == 0 {
		return []titleCard{}
	}

	latestDay := ""
	for _, e := range events {
		if day := e.CreatedAt.In(loc).Add(-6 * time.Hour).Format(time.DateOnly); day > latestDay {
			latestDay = day
		}
	}

	var first, last db.ConsumptionEvent
	firstSet := false
	for _, e := range events {
		if e.CreatedAt.In(loc).Add(-6*time.Hour).Format(time.DateOnly) != latestDay {
			continue
		}
		if !firstSet {
			first = e
			firstSet = true
		}
		last = e
	}

	cards := []titleCard{
		{Title: "First blood", Icon: "/static/icons/swords.svg", Description: "First mums of the day", Name: first.UserProfileName, Data: first.CreatedAt.In(loc).Format(time.TimeOnly)},
		{Title: "Holdout", Icon: "/static/icons/batman.svg", Description: "Last mums of the day", Name: last.UserProfileName, Data: last.CreatedAt.In(loc).Format(time.TimeOnly)},
	}

	if len(members) > 0 {
		// With one member (or a full tie) Drunkard and Designated driver name the same person.
		most := members[0]
		least := members[len(members)-1]
		cards = append(cards,
			titleCard{Title: "Drunkard", Icon: "/static/icons/zombie.svg", Description: "Most mums overall", Name: most.UserProfileName, Data: strconv.FormatInt(most.Mumsat, 10)},
			titleCard{Title: "Designated driver", Icon: "/static/icons/steering-wheel.svg", Description: "Least mums overall", Name: least.UserProfileName, Data: strconv.FormatInt(least.Mumsat, 10)},
		)
	}

	return cards
}

func loadPhaddergruppStatsData(c echo.Context, database *db.DB, phaddergruppID int64, role roles.PhaddergruppRole, base basePageData) (phaddergruppStatsTemplateData, error) {
	stats, err := database.ReadPhaddergruppStats(database, phaddergruppID, role)
	if err != nil {
		return phaddergruppStatsTemplateData{}, err
	}
	events, err := database.ReadPhaddergruppConsumptionEvents(database, phaddergruppID, role)
	if err != nil {
		return phaddergruppStatsTemplateData{}, err
	}
	loc := httpx.GetClientTimeLocation(c)

	return phaddergruppStatsTemplateData{
		basePageData:      base,
		PhaddergruppID:    phaddergruppID,
		RoleFilter:        role,
		RoleFilters:       statsRoleFilters,
		PhaddergruppData:  loaders.GetPhaddergrupp(c),
		PhaddergruppStats: stats,
		DailyBars:         readPhaddergruppDailyBars(events, loc),
		WeekdayHourRows:   readPhaddergruppWeekdayHourRows(events, loc),
		WeekdayLetters:    weekdayLetters[:],
		TitleCards:        readTitleCards(events, stats.Members, loc),
	}, nil
}

func GetPhaddergruppStats(c echo.Context) error {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)
	role := resolveStatsRole(c)

	templateData, err := loadPhaddergruppStatsData(c, database, phaddergruppID, role, basePageData{
		IsLoggedIn:        auth.GetIsLoggedIn(c),
		AllowedErrorCodes: []int{http.StatusInternalServerError},
		CSRFToken:         csrfToken(c),
	})
	if err != nil {
		return handleDBError(c, "phaddergrupp stats read", err)
	}
	return c.Render(http.StatusOK, "phaddergrupp-stats", templateData)
}

func emitPhaddergruppStatsUpdate(c echo.Context) {
	database := db.GetDB(c)
	phaddergruppID := auth.GetPhaddergruppID(c)
	role := resolveStatsRole(c)

	templateData, err := loadPhaddergruppStatsData(c, database, phaddergruppID, role, basePageData{})
	if err != nil {
		c.Logger().Errorf("Database error during phaddergrupp stats read: %v", err)
		return
	}

	var sb strings.Builder
	if err := c.Echo().Renderer.Render(&sb, "phaddergrupp-stats#fragment-phaddergrupp-stats", templateData, c); err != nil {
		c.Logger().Errorf("template render error: %v", err)
		return
	}

	_ = httpx.EmitSSE(c, "phaddergrupp-stats-update", sb.String())
}
