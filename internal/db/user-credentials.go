package db

import (
	"database/sql"
	"errors"
)

const SchemaUserCredentials = `
CREATE TABLE IF NOT EXISTS user_credentials (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email TEXT NOT NULL UNIQUE,
	hashword TEXT NOT NULL
);`

type UserCredentialsData struct {
	ID       int64
	Email    string
	Hashword string
}

func (db *DB) CreateUserCredentials(dbtx DBTX, email string, hashword string) (int64, error) {
	res, err := dbtx.Exec(
		`INSERT INTO user_credentials (email, hashword) VALUES (?, ?)`,
		email,
		hashword,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	dbtx.Emit(DBEvent{
		Table: "user_credentials",
		Type:  DBCreate,
		Data:  nil,
	})

	return id, nil
}

func (db *DB) ReadUserCredentialsIDAndHashwordByEmail(dbtx DBTX, email string) (int64, string, error) {
	row := dbtx.QueryRow(
		`SELECT id, hashword FROM user_credentials WHERE email = ?`,
		email,
	)

	var userCredentialsID int64
	var hashword string
	if err := row.Scan(&userCredentialsID, &hashword); err != nil {
		return 0, "", err
	}

	dbtx.Emit(DBEvent{
		Table: "user_credentials",
		Type:  DBRead,
		Data:  nil,
	})

	return userCredentialsID, hashword, nil
}

func (db *DB) ReadUserCredentialsExistsByEmail(dbtx DBTX, email string) (bool, error) {
	row := dbtx.QueryRow(
		`SELECT 1 FROM user_credentials WHERE email = ?`,
		email,
	)

	var exists bool
	err := row.Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	dbtx.Emit(DBEvent{
		Table: "user_credentials",
		Type:  DBRead,
		Data:  nil,
	})

	return true, nil
}

func (db *DB) UpdateUserCredentialsEmail(dbtx DBTX, userCredentialsID int64, email string) error {
	const sqlQuery = `
		UPDATE user_credentials
		SET email = ?
		WHERE id = ?`

	result, err := dbtx.Exec(sqlQuery, email, userCredentialsID)
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

	dbtx.Emit(DBEvent{
		Table: "user_credentials",
		Type:  DBUpdate,
		Data:  nil,
	})

	return nil
}

func (db *DB) UpdateUserCredentialsHashword(dbtx DBTX, userCredentialsID int64, hashword string) error {
	const sqlQuery = `
		UPDATE user_credentials
		SET hashword = ?
		WHERE id = ?`

	result, err := dbtx.Exec(sqlQuery, hashword, userCredentialsID)
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

	dbtx.Emit(DBEvent{
		Table: "user_credentials",
		Type:  DBUpdate,
		Data:  nil,
	})

	return nil
}
