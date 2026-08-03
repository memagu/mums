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

func rleTransactions(rows []db.MumsTransaction) []transactionLogEntry {
	entries := normalizeTransactions(rows)
	if len(entries) == 0 {
		return entries
	}

	runs := []transactionLogEntry{entries[0]}
	for _, entry := range entries[1:] {
		last := &runs[len(runs)-1]
		if last.UserAccountID == entry.UserAccountID {
			last.Delta += entry.Delta
			continue
		}
		runs = append(runs, entry)
	}
	return runs
}
