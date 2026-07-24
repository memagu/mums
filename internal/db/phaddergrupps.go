package db

import (
	"database/sql"
	"time"

	"github.com/memagu/mums/internal/config"
)

type PhaddergruppData struct {
	CreatedAt            time.Time
	Name                 string
	LogoFilePath         sql.NullString
	PrimaryColor         string
	SecondaryColor       string
	MumsPriceN0lla       float64
	MumsPricePhadder     float64
	MumsCurrency         string
	SwishRecipientNumber string
	MumsCapacityPerUser  int64
}

const SchemaPhaddergrupps = `
CREATE TABLE IF NOT EXISTS phaddergrupps (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at DATETIME DEFAULT NULL,
	name TEXT NOT NULL,
	logo_file_path TEXT DEFAULT NULL,
	primary_color TEXT NOT NULL,
	secondary_color TEXT NOT NULL,
	mums_price_n0lla REAL NOT NULL,
	mums_price_phadder REAL NOT NULL,
	mums_currency TEXT NOT NULL,
	swish_recipient_number TEXT NOT NULL,
	mums_capacity_per_user INTEGER NOT NULL
);`

func (db *DB) CreatePhaddergrupp(exec execer, name, swishRecipientNumber string) (int64, error) {
	res, err := exec.Exec(
		`INSERT INTO phaddergrupps (name, primary_color, secondary_color, mums_price_n0lla, mums_price_phadder, mums_currency, swish_recipient_number, mums_capacity_per_user) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		name,
		config.DefaultPrimaryPhaddergruppColor,
		config.DefaultSecondaryPhaddergruppColor,
		config.DefaultMumsPriceN0lla,
		config.DefaultMumsPricePhadder,
		config.DefaultMumsCurrency,
		swishRecipientNumber,
		config.DefaultMumsCapacityPerUser,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	db.Emit(DBEvent{
		Table: "phaddergrupps",
		Type:  DBCreate,
		Data:  nil,
	})

	return id, nil
}

func (db *DB) ReadPhaddergrupp(q queryer, phaddergruppID int64) (PhaddergruppData, error) {
	const sqlQuery = `
		SELECT
			created_at,
			name,
			logo_file_path,
			primary_color,
			secondary_color,
			mums_price_n0lla,
			mums_price_phadder,
			mums_currency,
			swish_recipient_number,
			mums_capacity_per_user
		FROM
			phaddergrupps
		WHERE
			id = ? AND deleted_at IS NULL
	`
	
	row := q.QueryRow(sqlQuery, phaddergruppID)

	var pd PhaddergruppData
	if err := row.Scan(
		&pd.CreatedAt,
		&pd.Name,
		&pd.LogoFilePath,
		&pd.PrimaryColor,
		&pd.SecondaryColor,
		&pd.MumsPriceN0lla,
		&pd.MumsPricePhadder,
		&pd.MumsCurrency,
		&pd.SwishRecipientNumber,
		&pd.MumsCapacityPerUser,
	); err != nil {
		return PhaddergruppData{}, err
	}

	db.Emit(DBEvent{
		Table: "phaddergrupps",
		Type:  DBRead,
		Data:  nil,
	})

	return pd, nil
}

func (db *DB) UpdatePhaddergrupp(exec execer, phaddergruppID int64, phaddergruppData PhaddergruppData) error {
	const sqlQuery = `
		UPDATE phaddergrupps SET
			name = ?,
			logo_file_path = ?,
			primary_color = ?,
			secondary_color = ?,
			mums_price_n0lla = ?,
			mums_price_phadder = ?,
			mums_currency = ?,
			swish_recipient_number = ?,
			mums_capacity_per_user = ?
		WHERE
			id = ? AND deleted_at IS NULL
	`

	result, err := exec.Exec(sqlQuery,
		phaddergruppData.Name,
		phaddergruppData.LogoFilePath,
		phaddergruppData.PrimaryColor,
		phaddergruppData.SecondaryColor,
		phaddergruppData.MumsPriceN0lla,
		phaddergruppData.MumsPricePhadder,
		phaddergruppData.MumsCurrency,
		phaddergruppData.SwishRecipientNumber,
		phaddergruppData.MumsCapacityPerUser,
		phaddergruppID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	db.Emit(DBEvent{
		Table: "phaddergrupps",
		Type:  DBUpdate,
		Data:  nil,
	})

	return nil
}

func (db *DB) DeletePhaddergrupp(exec execer, phaddergruppID int64) error {
	const sqlQuery = `
		UPDATE phaddergrupps
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := exec.Exec(sqlQuery, phaddergruppID)
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

	db.Emit(DBEvent{
		Table: "phaddergrupps",
		Type:  DBDelete,
		Data:  nil,
	})

	return nil
}
