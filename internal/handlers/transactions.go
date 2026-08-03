package handlers

import (
	"time"

	"github.com/memagu/mums/internal/db"
)

type transactionLogEntry struct {
	UserAccountID int64
	Name          string
	Delta         int64
	Time          time.Time
}

func normalizeTransactions(rows []db.MumsTransaction) []transactionLogEntry {
	entries := make([]transactionLogEntry, 0, len(rows))
	for _, row := range rows {
		delta := row.MumsQuantity
		if row.MumsType == db.Consumption {
			delta = -delta
		}
		entries = append(entries, transactionLogEntry{
			UserAccountID: row.UserAccountID,
			Name:          row.UserProfileName,
			Delta:         delta,
			Time:          row.CreatedAt,
		})
	}
	return entries
}
