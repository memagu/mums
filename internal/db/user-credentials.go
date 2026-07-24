package db

import "database/sql"

const SchemaUserCredentials = `
CREATE TABLE IF NOT EXISTS user_credentials (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email TEXT NOT NULL UNIQUE,
	hashword TEXT NOT NULL
);`

func (db *DB) CreateUserCredentials(exec execer, email string, hashword string) (int64, error) {
	res, err := exec.Exec(
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

	db.Emit(DBEvent{
		Table: "user_credentials",
		Type:  DBCreate,
		Data:  nil,
	})

	return id, nil
}

func (db *DB) ReadUserCredentialsIDAndHashwordByEmail(q queryer, email string) (int64, string, error) {
	row := q.QueryRow(
		`SELECT id, hashword FROM user_credentials WHERE email = ?`,
		email,
	)

	var userCredentialsID int64
	var hashword string
	if err := row.Scan(&userCredentialsID, &hashword); err != nil {
		return 0, "", err
	}

	db.Emit(DBEvent{
		Table: "user_credentials",
		Type:  DBRead,
		Data:  nil,
	})

	return userCredentialsID, hashword, nil
}

func (db *DB) ReadUserCredentialsExistsByEmail(q queryer, email string) (bool, error) {
	row := q.QueryRow(
		`SELECT 1 FROM user_credentials WHERE email = ?`,
		email,
	)

	var exists bool
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	db.Emit(DBEvent{
		Table: "user_credentials",
		Type:  DBRead,
		Data:  nil,
	})

	return true, nil
}
