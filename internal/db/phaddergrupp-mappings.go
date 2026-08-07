package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/memagu/mums/internal/roles"
)

const SchemaPhaddergruppMappings = `
CREATE TABLE IF NOT EXISTS phaddergrupp_mappings (
	user_account_id INTEGER NOT NULL,
	phaddergrupp_id INTEGER NOT NULL,
	phaddergrupp_role TEXT NOT NULL,
	mums_available INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (user_account_id, phaddergrupp_id),
	FOREIGN KEY (user_account_id) REFERENCES user_accounts(id) ON DELETE CASCADE,
	FOREIGN KEY (phaddergrupp_id) REFERENCES phaddergrupper(id) ON DELETE CASCADE
);`
const IndexPhaddergruppMappingsOnPhaddergruppID = `
CREATE INDEX IF NOT EXISTS
	idx_phaddergrupp_mappings_phaddergrupp_id
ON
	phaddergrupp_mappings(phaddergrupp_id)
;`

func (db *DB) CreatePhaddergruppMapping(dbtx DBTX, userAccountID, phaddergruppID int64, phaddergruppRole roles.PhaddergruppRole) error {
	_, err := dbtx.Exec(
		`INSERT INTO phaddergrupp_mappings (user_account_id, phaddergrupp_id, phaddergrupp_role) VALUES (?, ?, ?)`,
		userAccountID,
		phaddergruppID,
		string(phaddergruppRole),
	)
	if err != nil {
		return err
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBCreate,
		Data: PhaddergruppMappingEvent{
			UserAccountID:  userAccountID,
			PhaddergruppID: phaddergruppID,
		},
	})

	return nil
}

func (db *DB) ReadUserAccountIsMemberOfPhaddergrupp(dbtx DBTX, userAccountID, phaddergruppID int64) (bool, error) {
	const sqlQuery = `
		SELECT
			EXISTS (
				SELECT
					1
				FROM
					phaddergrupp_mappings AS pm
				JOIN
					phaddergrupper AS pg ON pm.phaddergrupp_id = pg.id AND pg.deleted_at IS NULL
				WHERE
					pm.user_account_id = ? AND pm.phaddergrupp_id = ?
			);
	`

	row := dbtx.QueryRow(sqlQuery, userAccountID, phaddergruppID)

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBRead,
		Data:  nil,
	})

	return exists, nil
}

func (db *DB) ReadPhaddergruppIsEmpty(dbtx DBTX, phaddergruppID int64) (bool, error) {
	const sqlQuery = `
		SELECT NOT EXISTS (
			SELECT 1
			FROM phaddergrupp_mappings AS pm
			JOIN phaddergrupper AS pg ON pm.phaddergrupp_id = pg.id AND pg.deleted_at IS NULL
			WHERE pm.phaddergrupp_id = ?
		)
	`

	row := dbtx.QueryRow(sqlQuery, phaddergruppID)

	var isEmpty bool
	if err := row.Scan(&isEmpty); err != nil {
		return false, err
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBRead,
		Data:  nil,
	})

	return isEmpty, nil
}

func (db *DB) ReadPhaddergruppRole(dbtx DBTX, userAccountID, phaddergruppID int64) (roles.PhaddergruppRole, error) {
	row := dbtx.QueryRow(`
		SELECT pm.phaddergrupp_role
		FROM phaddergrupp_mappings AS pm
		JOIN phaddergrupper AS pg ON pm.phaddergrupp_id = pg.id AND pg.deleted_at IS NULL
		WHERE pm.user_account_id = ? AND pm.phaddergrupp_id = ?`,
		userAccountID, phaddergruppID)

	var phaddergruppRole roles.PhaddergruppRole
	if err := row.Scan(&phaddergruppRole); err != nil {
		return "", err
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBRead,
		Data:  nil,
	})

	return phaddergruppRole, nil
}

func (db *DB) ReadMumsAvailable(dbtx DBTX, userAccountID, phaddergruppID int64) (int64, error) {
	sqlQuery := `
		SELECT mums_available
		FROM phaddergrupp_mappings
		WHERE user_account_id = ? AND phaddergrupp_id = ?
	`
	row := dbtx.QueryRow(sqlQuery, userAccountID, phaddergruppID)

	var mumsAvailable int64
	if err := row.Scan(&mumsAvailable); err != nil {
		return 0, err
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBRead,
		Data:  nil,
	})

	return mumsAvailable, nil
}

type UserPhaddergruppSummary struct {
	ID               int64
	Name             string
	LogoPath         sql.NullString
	PrimaryColor     string
	SecondaryColor   string
	PhadderCount     int
	N0llaCount       int
	PhaddergruppRole roles.PhaddergruppRole
	MumsAvailable    int
}

func (db *DB) ReadUserPhaddergruppSummariesByUserAccountID(dbtx DBTX, userAccountID int64) ([]UserPhaddergruppSummary, error) {
	const sqlQuery = `
		WITH GroupCounts AS (
			SELECT
				phaddergrupp_id,
				SUM(CASE WHEN phaddergrupp_role = 'phadder' THEN 1 ELSE 0 END) AS pc,
				SUM(CASE WHEN phaddergrupp_role = 'n0lla' THEN 1 ELSE 0 END) AS nc
			FROM
				phaddergrupp_mappings
			GROUP BY
				phaddergrupp_id
		)
		SELECT
			pg.id,
			pg.name,
			pg.logo_file_path,
			pg.primary_color,
			pg.secondary_color,
			gc.pc,
			gc.nc,
			pm.phaddergrupp_role,
			pm.mums_available
		FROM
			phaddergrupp_mappings AS pm
		JOIN
			phaddergrupper AS pg ON pm.phaddergrupp_id = pg.id AND pg.deleted_at IS NULL
		JOIN
			GroupCounts AS gc ON pm.phaddergrupp_id = gc.phaddergrupp_id
		WHERE
			pm.user_account_id = ?
		ORDER BY
			pg.created_at DESC;
	`

	rows, err := dbtx.Query(sqlQuery, userAccountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []UserPhaddergruppSummary
	for rows.Next() {
		var s UserPhaddergruppSummary
		if err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.LogoPath,
			&s.PrimaryColor,
			&s.SecondaryColor,
			&s.PhadderCount,
			&s.N0llaCount,
			&s.PhaddergruppRole,
			&s.MumsAvailable,
		); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBRead,
		Data:  nil,
	})
	dbtx.Emit(DBEvent{
		Table: "phaddergrupper",
		Type:  DBRead,
		Data:  nil,
	})

	return summaries, nil
}

type PhaddergruppUserSummary struct {
	UserAccountID          int64
	UserProfileName        string
	PhaddergruppRole       roles.PhaddergruppRole
	MumsAvailable          int
	MumsaCount             int
	MumsaTimesAttr         string
	MumsRecencyWindowLabel string
}

type PhaddergruppUserSummaries struct {
	N0llor   []PhaddergruppUserSummary
	Phaddrar []PhaddergruppUserSummary
}

func (db *DB) ReadPhaddergruppUserSummariesByPhaddergruppID(dbtx DBTX, phaddergruppID int64) (PhaddergruppUserSummaries, error) {
	const sqlQuery = `
		SELECT
			ua.id,
			up.name,
			pm.phaddergrupp_role,
			pm.mums_available
		FROM
			phaddergrupp_mappings AS pm
		JOIN
			phaddergrupper AS pg ON pg.id = pm.phaddergrupp_id AND pg.deleted_at IS NULL
		JOIN
			user_accounts AS ua ON ua.id = pm.user_account_id AND ua.deleted_at IS NULL
		JOIN
			user_profiles AS up ON up.id = ua.user_profile_id
		WHERE
			pm.phaddergrupp_id = ?
		ORDER BY
			up.name;
	`

	rows, err := dbtx.Query(sqlQuery, phaddergruppID)
	if err != nil {
		return PhaddergruppUserSummaries{}, err
	}
	defer rows.Close()

	var summaries PhaddergruppUserSummaries

	for rows.Next() {
		var summary PhaddergruppUserSummary
		if err := rows.Scan(
			&summary.UserAccountID,
			&summary.UserProfileName,
			&summary.PhaddergruppRole,
			&summary.MumsAvailable,
		); err != nil {
			return PhaddergruppUserSummaries{}, err
		}

		switch summary.PhaddergruppRole {
		case roles.N0lla:
			summaries.N0llor = append(summaries.N0llor, summary)
		case roles.Phadder:
			summaries.Phaddrar = append(summaries.Phaddrar, summary)
		default:
			return PhaddergruppUserSummaries{}, fmt.Errorf("unknown phaddergrupp role: %v for user %d", summary.PhaddergruppRole, summary.UserAccountID)
		}
	}

	if err := rows.Err(); err != nil {
		return PhaddergruppUserSummaries{}, err
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBRead,
		Data:  nil,
	})
	dbtx.Emit(DBEvent{
		Table: "user_accounts",
		Type:  DBRead,
		Data:  nil,
	})
	dbtx.Emit(DBEvent{
		Table: "user_profiles",
		Type:  DBRead,
		Data:  nil,
	})

	return summaries, nil
}

func (db *DB) ReadLastCreatedPhaddergruppIDByUserAccountID(dbtx DBTX, userAccountID int64) (int64, time.Time, error) {
	const sqlQuery = `
		SELECT
			p.id,
			p.created_at
		FROM
			phaddergrupp_mappings AS pm
		JOIN
			phaddergrupper AS p ON p.id = pm.phaddergrupp_id AND p.deleted_at IS NULL
		WHERE
			pm.user_account_id = ?
		ORDER BY
			p.created_at DESC;
	`

	row := dbtx.QueryRow(sqlQuery, userAccountID)

	var phaddergruppID int64
	var createdAt time.Time
	if err := row.Scan(&phaddergruppID, &createdAt); err != nil {
		return 0, time.Time{}, err
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBRead,
		Data:  nil,
	})
	dbtx.Emit(DBEvent{
		Table: "phaddergrupper",
		Type:  DBRead,
		Data:  nil,
	})

	return phaddergruppID, createdAt, nil
} // Returns zero if no rows were affected (not found = 0 as well)

func (db *DB) UpdateAdjustMumsAvailable(dbtx DBTX, userAccountID, phaddergruppID, amount int64) (int64, error) {
	const sqlQuery = `
		UPDATE
			phaddergrupp_mappings
		SET
			mums_available = mums_available + ?
		WHERE
			user_account_id = ? AND phaddergrupp_id = ? AND mums_available + ? >= 0
		RETURNING
			mums_available;
	`

	var mumsAvailable int64
	row := dbtx.QueryRow(sqlQuery, amount, userAccountID, phaddergruppID, amount)
	if err := row.Scan(&mumsAvailable); err != nil {
		return 0, err
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBUpdate,
		Data: MumsAvailableUpdate{
			UserAccountID:  userAccountID,
			PhaddergruppID: phaddergruppID,
			MumsAvailable:  mumsAvailable,
		},
	})

	return mumsAvailable, nil
}

func (db *DB) DeletePhaddergruppMapping(dbtx DBTX, userAccountID, phaddergruppID int64) error {
	const sqlQuery = `
		DELETE FROM
			phaddergrupp_mappings
		WHERE
			user_account_id = ? AND phaddergrupp_id = ?
	`
	result, err := dbtx.Exec(sqlQuery, userAccountID, phaddergruppID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return nil
	}

	dbtx.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBDelete,
		Data: PhaddergruppMappingEvent{
			UserAccountID:  userAccountID,
			PhaddergruppID: phaddergruppID,
		},
	})

	return nil
}
