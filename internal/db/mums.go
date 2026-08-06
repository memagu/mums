package db

import (
	"database/sql"
	"strings"
	"time"

	"github.com/memagu/mums/internal/roles"
)

type MumsType string

const (
	Purchase    MumsType = "purchase"
	Consumption MumsType = "consumption"
)

const SchemaMums = `
CREATE TABLE IF NOT EXISTS mums (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	user_account_id INTEGER NOT NULL,
	phaddergrupp_id INTEGER NOT NULL,
	mums_quantity INTEGER NOT NULL,
	mums_type TEXT NOT NULL,
    FOREIGN KEY (user_account_id) REFERENCES user_accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (phaddergrupp_id) REFERENCES phaddergrupper(id) ON DELETE CASCADE
);`

const IndexMumsOnPhaddergruppID = `
CREATE INDEX IF NOT EXISTS idx_mums_phaddergrupp_id ON mums(phaddergrupp_id);`

const IndexMumsOnUserAccountID = `
CREATE INDEX IF NOT EXISTS idx_mums_user_account_id ON mums(user_account_id);`

func (db *DB) CreateMums(e execer, userAccountID, phaddergruppID, mumsQuantity int64, mumsType MumsType) (int64, error) {
	res, err := e.Exec(
		`INSERT INTO mums (user_account_id, phaddergrupp_id, mums_quantity, mums_type) VALUES (?, ?, ?, ?)`,
		userAccountID,
		phaddergruppID,
		mumsQuantity,
		mumsType,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	e.Emit(DBEvent{
		Table: "mums",
		Type:  DBCreate,
		Data:  nil,
	})

	return id, nil
}

type MumsTransaction struct {
	UserAccountID   int64
	UserProfileName string
	MumsQuantity    int64
	MumsType        MumsType
	CreatedAt       time.Time
}

func (db *DB) ReadPhaddergruppTransactions(q queryer, phaddergruppID, memberID int64, role roles.PhaddergruppRole, mumsType MumsType) ([]MumsTransaction, error) {
	query := `
		SELECT
			m.user_account_id,
			up.name,
			m.mums_quantity,
			m.mums_type,
			m.created_at
		FROM
			mums AS m
		JOIN
			user_accounts AS ua ON ua.id = m.user_account_id
		JOIN
			user_profiles AS up ON up.id = ua.user_profile_id
	`
	conditions := []string{"m.phaddergrupp_id = ?"}
	args := []any{phaddergruppID}

	if memberID > 0 {
		conditions = append(conditions, "m.user_account_id = ?")
		args = append(args, memberID)
	}
	if role != "" {
		query += `JOIN phaddergrupp_mappings AS pm ON pm.user_account_id = m.user_account_id AND pm.phaddergrupp_id = m.phaddergrupp_id`
		conditions = append(conditions, "pm.phaddergrupp_role = ?")
		args = append(args, string(role))
	}
	if mumsType != "" {
		conditions = append(conditions, "m.mums_type = ?")
		args = append(args, string(mumsType))
	}

	query += " WHERE " + strings.Join(conditions, " AND ") + " ORDER BY m.created_at DESC, m.id DESC"

	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []MumsTransaction
	for rows.Next() {
		var transaction MumsTransaction
		if err := rows.Scan(
			&transaction.UserAccountID,
			&transaction.UserProfileName,
			&transaction.MumsQuantity,
			&transaction.MumsType,
			&transaction.CreatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	q.Emit(DBEvent{
		Table: "mums",
		Type:  DBRead,
		Data:  nil,
	})

	return transactions, nil
}

type MemberMumsStats struct {
	UserAccountID   int64
	UserProfileName string
	Mumsat          int64
	Bought          int64
}

type PhaddergruppStats struct {
	Members     []MemberMumsStats
	TotalMumsat int64
	TotalBought int64
}

func (db *DB) ReadPhaddergruppStats(q queryer, phaddergruppID int64) (PhaddergruppStats, error) {
	const sqlQuery = `
		SELECT
			ua.id,
			up.name,
			COALESCE(SUM(CASE WHEN m.mums_type = 'consumption' OR m.mums_quantity < 0 THEN ABS(m.mums_quantity) ELSE 0 END), 0) AS mumsat,
			COALESCE(SUM(CASE WHEN m.mums_type = 'purchase' AND m.mums_quantity > 0 THEN m.mums_quantity ELSE 0 END), 0) AS bought
		FROM
			phaddergrupp_mappings AS pm
		JOIN
			user_accounts AS ua ON ua.id = pm.user_account_id AND ua.deleted_at IS NULL
		JOIN
			user_profiles AS up ON up.id = ua.user_profile_id
		LEFT JOIN
			mums AS m ON m.user_account_id = pm.user_account_id AND m.phaddergrupp_id = pm.phaddergrupp_id
		WHERE
			pm.phaddergrupp_id = ?
		GROUP BY
			ua.id, up.name
		ORDER BY
			mumsat DESC, up.name
	`

	rows, err := q.Query(sqlQuery, phaddergruppID)
	if err != nil {
		return PhaddergruppStats{}, err
	}
	defer rows.Close()

	var stats PhaddergruppStats
	for rows.Next() {
		var memberStats MemberMumsStats
		if err := rows.Scan(
			&memberStats.UserAccountID,
			&memberStats.UserProfileName,
			&memberStats.Mumsat,
			&memberStats.Bought,
		); err != nil {
			return PhaddergruppStats{}, err
		}
		stats.Members = append(stats.Members, memberStats)
		stats.TotalMumsat += memberStats.Mumsat
		stats.TotalBought += memberStats.Bought
	}

	if err := rows.Err(); err != nil {
		return PhaddergruppStats{}, err
	}

	q.Emit(DBEvent{
		Table: "mums",
		Type:  DBRead,
		Data:  nil,
	})

	return stats, nil
}

func parseSQLiteTime(src sql.NullString) (sql.NullTime, error) {
	if !src.Valid {
		return sql.NullTime{}, nil
	}
	t, err := time.Parse(time.DateTime, src.String)
	if err != nil {
		return sql.NullTime{}, err
	}
	return sql.NullTime{Time: t, Valid: true}, nil
}

func (db *DB) ReadMemberLastMumsaAt(q queryer, userAccountID, phaddergruppID int64) (sql.NullTime, error) {
	row := q.QueryRow(`
		SELECT
			MAX(created_at)
		FROM
			mums
		WHERE
			user_account_id = ? AND phaddergrupp_id = ?
			AND (mums_type = 'consumption' OR mums_quantity < 0)`,
		userAccountID,
		phaddergruppID,
	)

	var lastMumsaAtStr sql.NullString
	if err := row.Scan(&lastMumsaAtStr); err != nil {
		return sql.NullTime{}, err
	}

	lastMumsaAt, err := parseSQLiteTime(lastMumsaAtStr)
	if err != nil {
		return sql.NullTime{}, err
	}

	q.Emit(DBEvent{
		Table: "mums",
		Type:  DBRead,
		Data:  nil,
	})

	return lastMumsaAt, nil
}
