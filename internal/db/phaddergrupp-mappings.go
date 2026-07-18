package db

import (
	"fmt"
	"database/sql"

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
    FOREIGN KEY (phaddergrupp_id) REFERENCES phaddergrupps(id) ON DELETE CASCADE
);`
const IndexPhaddergruppMappingsOnPhaddergruppID = `
CREATE INDEX IF NOT EXISTS 
	idx_phaddergrupp_mappings_phaddergrupp_id 
ON 
	phaddergrupp_mappings(phaddergrupp_id)
;`

func (db *DB) CreatePhaddergruppMapping(exec execer, userAccountID, phaddergruppID int64, phaddergruppRole roles.PhaddergruppRole) error {
	_, err := exec.Exec(
		`INSERT INTO phaddergrupp_mappings (user_account_id, phaddergrupp_id, phaddergrupp_role) VALUES (?, ?, ?)`,
		userAccountID,
		phaddergruppID,
		string(phaddergruppRole),
	)

	db.Emit(DBEvent{
		"phaddergrupp_mappings",
		DBCreate,
		nil,
	})

	return err
}

func (db *DB) ReadUserAccountIsMemberOfPhaddergrupp(q queryer, userAccountID, phaddergruppID int64) (bool, error) {
	const sqlQuery = `
		SELECT
			EXISTS (
				SELECT
					1
				FROM
					phaddergrupp_mappings
				WHERE
					user_account_id = ? AND phaddergrupp_id = ?
			);
	`

	row := q.QueryRow(sqlQuery, userAccountID, phaddergruppID)

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}

	db.Emit(DBEvent{
		"phaddergrupp_mappings",
		DBRead,
		nil,
	})

	return exists, nil
}

func (db *DB) ReadPhaddergruppIsEmpty(q queryer, phaddergruppID int64) (bool, error) {
	const sqlQuery = `
		SELECT NOT EXISTS (
			SELECT 1
			FROM phaddergrupp_mappings
			WHERE phaddergrupp_id = ?
		)
	`

	row := q.QueryRow(sqlQuery, phaddergruppID)

	var isEmpty bool
	if err := row.Scan(&isEmpty); err != nil {
		return false, err
	}

	db.Emit(DBEvent{
		"phaddergrupp_mappings",
		DBRead,
		nil,
	})

	return isEmpty, nil
}

func (db *DB) ReadPhaddergruppRole(q queryer, userAccountID, phaddergruppID int64) (roles.PhaddergruppRole, error) {
	row := q.QueryRow(`
		SELECT phaddergrupp_role
		FROM phaddergrupp_mappings
		WHERE user_account_id = ? AND phaddergrupp_id = ?`,
		userAccountID, phaddergruppID)

	var phaddergruppRole roles.PhaddergruppRole
	if err := row.Scan(&phaddergruppRole); err != nil {
		return "", err
	}

	db.Emit(DBEvent{
		"phaddergrupp_mappings",
		DBRead,
		nil,
	})

	return phaddergruppRole, nil
}

func (db *DB) ReadMumsAvailable(q queryer, userAccountID, phaddergruppID int64) (int64, error) {
	sqlQuery := `
		SELECT mums_available
		FROM phaddergrupp_mappings
		WHERE user_account_id = ? AND phaddergrupp_id = ?
	`
	row := q.QueryRow(sqlQuery, userAccountID, phaddergruppID)

	var mumsAvailable int64 
	if err := row.Scan(&mumsAvailable); err != nil {
		return 0, err
	}

	db.Emit(DBEvent{
		"phaddergrupp_mappings",
		DBRead,
		nil,
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

func (db *DB) ReadUserPhaddergruppSummariesByUserAccountID(q queryer, userAccountID int64) ([]UserPhaddergruppSummary, error) {
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
			phaddergrupps AS pg ON pm.phaddergrupp_id = pg.id
		JOIN
			GroupCounts AS gc ON pm.phaddergrupp_id = gc.phaddergrupp_id
		WHERE
			pm.user_account_id = ?
		ORDER BY
			pg.created_at DESC;
	`

	rows, err := q.Query(sqlQuery, userAccountID)
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

	db.Emit(DBEvent{
		"phaddergrupp_mappings",
		DBRead,
		nil,
	})
	db.Emit(DBEvent{
		"phaddergrupps",
		DBRead,
		nil,
	})

	return summaries, nil
}


type PhaddergruppUserSummary struct {
    UserAccountID    int64
    UserProfileName  string
    PhaddergruppRole roles.PhaddergruppRole
    MumsAvailable    int
}

type PhaddergruppUserSummaries struct {
	N0llas   []PhaddergruppUserSummary
	Phadders []PhaddergruppUserSummary
}

func (db *DB) ReadPhaddergruppUserSummariesByPhaddergruppID(q queryer, phaddergruppID int64) (PhaddergruppUserSummaries, error) {
    const sqlQuery = `
        SELECT
            ua.id,
            up.name,
            pm.phaddergrupp_role,
            pm.mums_available
        FROM
            phaddergrupp_mappings AS pm
        JOIN
            user_accounts AS ua ON ua.id = pm.user_account_id
        JOIN
            user_profiles AS up ON up.id = ua.user_profile_id
        WHERE
            pm.phaddergrupp_id = ?
        ORDER BY
            up.name;
    `

    rows, err := q.Query(sqlQuery, phaddergruppID)
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
			summaries.N0llas = append(summaries.N0llas, summary)
		case roles.Phadder:
			summaries.Phadders = append(summaries.Phadders, summary)
		default:
			return PhaddergruppUserSummaries{}, fmt.Errorf("unknown phaddergrupp role: %v for user %d", summary.PhaddergruppRole, summary.UserAccountID)
		}
    }

    if err := rows.Err(); err != nil {
        return PhaddergruppUserSummaries{}, err
    }

    db.Emit(DBEvent{
        "phaddergrupp_mappings",
        DBRead,
        nil,
    })
    db.Emit(DBEvent{
        "user_accounts",
        DBRead,
        nil,
    })
    db.Emit(DBEvent{
        "user_profiles",
        DBRead,
        nil,
    })

    return summaries, nil
}

func (db *DB) ReadLastCreatedPhaddergruppIDByUserAccountID(q queryer, userAccountID int64) (int64, error) {
	const sqlQuery =`
		SELECT
			p.id
		FROM
			phaddergrupp_mappings AS pm
		JOIN
			phaddergrupps AS p ON p.id = pm.phaddergrupp_id
		WHERE
			pm.user_account_id = ?
		ORDER BY
			p.created_at DESC;
	`

	row := q.QueryRow(sqlQuery, userAccountID)

	var phaddergruppID int64
	if err := row.Scan(&phaddergruppID); err != nil {
		return 0, err
	}

    db.Emit(DBEvent{
        "phaddergrupp_mappings",
        DBRead,
        nil,
    })
	db.Emit(DBEvent{
		"phaddergrupps",
		DBRead,
		nil,
	})

	return phaddergruppID, nil
}

type MumsAvailableUpdate struct {
	UserAccountID  int64
	PhaddergruppID int64
	MumsAvailable  int64
}

// Returns zero if no rows were affected (not found = 0 as well)
func (db *DB) UpdateAdjustMumsAvailable(q queryer, userAccountID, phaddergruppID, amount int64) (int64, error) {
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
	row := q.QueryRow(sqlQuery, amount, userAccountID, phaddergruppID, amount)
	if err := row.Scan(&mumsAvailable); err != nil {
		return 0, err
	}

	db.Emit(DBEvent{
		Table: "phaddergrupp_mappings",
		Type:  DBUpdate,
		Data: MumsAvailableUpdate{
			UserAccountID: userAccountID,
			PhaddergruppID: phaddergruppID,
			MumsAvailable: mumsAvailable,
		},
	})

	return mumsAvailable, nil
}

func (db *DB) DeletePhaddergruppMapping(exec execer, userAccountID, phaddergruppID int64) error {
	const sqlQuery = `
		DELETE FROM
			phaddergrupp_mappings
		WHERE
			user_account_id = ? AND phaddergrupp_id = ?
	`
	result, err := exec.Exec(sqlQuery, userAccountID, phaddergruppID)
	if err != nil {
		return err
	}

	_, err = result.RowsAffected()
	if err != nil {
		return err
	}

	db.Emit(DBEvent{
		"phaddergrupp_mappings",
		DBDelete,
		nil,
	})

	return nil
}
