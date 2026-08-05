package db

import (
	"database/sql"
	"time"

	"github.com/memagu/mums/internal/config"
)

type PhaddergruppData struct {
	CreatedAt                   time.Time
	Name                        string
	LogoFilePath                sql.NullString
	PrimaryColor                string
	SecondaryColor              string
	MumsPriceN0lla              float64
	MumsPricePhadder            float64
	MumsCurrency                string
	SwishRecipientNumber        string
	MumsCapacityPerUser         int64
	MumsMinPurchaseQuantity     int64
	MumsMaxPurchaseQuantity     int64
	MumsPurchaseQuantityStep    int64
	MumsDefaultPurchaseQuantity int64
}

const SchemaPhaddergrupper = `
CREATE TABLE IF NOT EXISTS phaddergrupper (
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
	mums_capacity_per_user INTEGER NOT NULL,
	mums_min_purchase_quantity INTEGER NOT NULL,
	mums_max_purchase_quantity INTEGER NOT NULL,
	mums_purchase_quantity_step INTEGER NOT NULL,
	mums_default_purchase_quantity INTEGER NOT NULL
);`

func (db *DB) CreatePhaddergrupp(e execer, name, swishRecipientNumber string) (int64, error) {
	res, err := e.Exec(
		`INSERT INTO phaddergrupper (name, primary_color, secondary_color, mums_price_n0lla, mums_price_phadder, mums_currency, swish_recipient_number, mums_capacity_per_user, mums_min_purchase_quantity, mums_max_purchase_quantity, mums_purchase_quantity_step, mums_default_purchase_quantity) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		name,
		config.Defaults.Phaddergrupp.PrimaryColor,
		config.Defaults.Phaddergrupp.SecondaryColor,
		config.Defaults.Mums.PriceN0lla,
		config.Defaults.Mums.PricePhadder,
		config.Defaults.Mums.Currency,
		swishRecipientNumber,
		config.Defaults.Mums.CapacityPerUser,
		config.Defaults.Mums.MinPurchaseQuantity,
		config.Defaults.Mums.MaxPurchaseQuantity,
		config.Defaults.Mums.StepPurchaseQuantity,
		config.Defaults.Mums.DefaultPurchaseQuantity,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	e.Emit(DBEvent{
		Table: "phaddergrupper",
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
			mums_capacity_per_user,
			mums_min_purchase_quantity,
			mums_max_purchase_quantity,
			mums_purchase_quantity_step,
			mums_default_purchase_quantity
		FROM
			phaddergrupper
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
		&pd.MumsMinPurchaseQuantity,
		&pd.MumsMaxPurchaseQuantity,
		&pd.MumsPurchaseQuantityStep,
		&pd.MumsDefaultPurchaseQuantity,
	); err != nil {
		return PhaddergruppData{}, err
	}

	q.Emit(DBEvent{
		Table: "phaddergrupper",
		Type:  DBRead,
		Data:  nil,
	})

	return pd, nil
}

func (db *DB) UpdatePhaddergrupp(e execer, phaddergruppID int64, phaddergruppData PhaddergruppData) error {
	const sqlQuery = `
		UPDATE phaddergrupper SET
			name = ?,
			logo_file_path = ?,
			primary_color = ?,
			secondary_color = ?,
			mums_price_n0lla = ?,
			mums_price_phadder = ?,
			mums_currency = ?,
			swish_recipient_number = ?,
			mums_capacity_per_user = ?,
			mums_min_purchase_quantity = ?,
			mums_max_purchase_quantity = ?,
			mums_purchase_quantity_step = ?,
			mums_default_purchase_quantity = ?
		WHERE
			id = ? AND deleted_at IS NULL
	`

	result, err := e.Exec(sqlQuery,
		phaddergruppData.Name,
		phaddergruppData.LogoFilePath,
		phaddergruppData.PrimaryColor,
		phaddergruppData.SecondaryColor,
		phaddergruppData.MumsPriceN0lla,
		phaddergruppData.MumsPricePhadder,
		phaddergruppData.MumsCurrency,
		phaddergruppData.SwishRecipientNumber,
		phaddergruppData.MumsCapacityPerUser,
		phaddergruppData.MumsMinPurchaseQuantity,
		phaddergruppData.MumsMaxPurchaseQuantity,
		phaddergruppData.MumsPurchaseQuantityStep,
		phaddergruppData.MumsDefaultPurchaseQuantity,
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

	e.Emit(DBEvent{
		Table: "phaddergrupper",
		Type:  DBUpdate,
		Data:  nil,
	})

	return nil
}

func (db *DB) DeletePhaddergrupp(e execer, phaddergruppID int64) error {
	const sqlQuery = `
		UPDATE phaddergrupper
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := e.Exec(sqlQuery, phaddergruppID)
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

	e.Emit(DBEvent{
		Table: "phaddergrupper",
		Type:  DBDelete,
		Data:  nil,
	})

	return nil
}
