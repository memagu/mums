package db

import (
	"fmt"

	"github.com/memagu/mums/internal/roles"
)

type PhaddergruppInviteData struct {
	PhaddergruppID 	 int64
	PhaddergruppRole roles.PhaddergruppRole
}

const SchemaPhaddergruppInvites = `
CREATE TABLE IF NOT EXISTS phaddergrupp_invites (
	token TEXT PRIMARY KEY,
	phaddergrupp_id INTEGER NOT NULL,
	phaddergrupp_role TEXT NOT NULL,
	UNIQUE (phaddergrupp_id, phaddergrupp_role),
    FOREIGN KEY (phaddergrupp_id) REFERENCES phaddergrupps(id) ON DELETE CASCADE
);`
const IndexPhaddergruppInvitesOnPhaddergruppID = `
CREATE INDEX IF NOT EXISTS 
	idx_phaddergrupp_invites_phaddergrupp_id 
ON 
	phaddergrupp_invites(phaddergrupp_id)
;`

func (db *DB) CreatePhaddergruppInvite(exec execer, token string, phaddergruppID int64, phaddergruppRole roles.PhaddergruppRole) (error) {
	sqlQuery := `
		INSERT INTO phaddergrupp_invites
			(token, phaddergrupp_id, phaddergrupp_role)
		VALUES
			(?, ?, ?)
	`

	_, err := exec.Exec(sqlQuery, token, phaddergruppID, phaddergruppRole)
	if err != nil {
		return err
	}

	db.Emit(DBEvent{
		"phaddergrupp_invites",
		DBCreate,
		nil,
	})

	return nil
}

func (db *DB) ReadPhaddergruppInvite(q queryer, token string) (PhaddergruppInviteData, error) {
	sqlQuery := `
		SELECT
			phaddergrupp_id,
			phaddergrupp_role
		FROM
			phaddergrupp_invites
		WHERE
			token = ?
	`

	row := q.QueryRow(sqlQuery, token) 

	var pid PhaddergruppInviteData
	if err := row.Scan(
		&pid.PhaddergruppID,
		&pid.PhaddergruppRole,
	); err != nil {
		return PhaddergruppInviteData{}, err
	}

	db.Emit(DBEvent{
		"phaddergrupp_invites",
		DBRead,
		nil,
	})

	return pid, nil
}

type PhaddergruppInviteTokens struct {
	N0lla   string
	Phadder string
}

func (db *DB) ReadPhaddergruppInviteTokensByPhaddergruppID(q queryer, phaddergruppID int64) (PhaddergruppInviteTokens, error) {
	sqlQuery := `
		SELECT
			token, phaddergrupp_role
		FROM
			phaddergrupp_invites
		WHERE
			phaddergrupp_id = ?
	`

	rows, err := q.Query(sqlQuery, phaddergruppID)
	if err != nil {
		return PhaddergruppInviteTokens{}, err
	}
	defer rows.Close()

	var tokens PhaddergruppInviteTokens

	for rows.Next() {
		var token string
		var role roles.PhaddergruppRole
		if err := rows.Scan(&token, &role); err != nil {
			return PhaddergruppInviteTokens{}, err
		}

		switch role {
		case roles.N0lla:
			tokens.N0lla = token
		case roles.Phadder:
			tokens.Phadder = token
		default:
			return PhaddergruppInviteTokens{}, fmt.Errorf("unexpected phaddergrupp role: %s", role)
		}
	}

	if err := rows.Err(); err != nil {
		return PhaddergruppInviteTokens{}, err
	}

	db.Emit(DBEvent{
		"phaddergrupp_invites",
		DBRead,
		nil,
	})

	return tokens, nil
}
