package db

import "database/sql"

type UserProfileData struct {
	Name string
}

const SchemaUserProfiles = `
CREATE TABLE IF NOT EXISTS user_profiles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL
);`

func (db *DB) CreateUserProfile(dbtx DBTX, name string) (int64, error) {
	res, err := dbtx.Exec(`INSERT INTO user_profiles (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	dbtx.Emit(DBEvent{
		Table: "user_profiles",
		Type:  DBCreate,
		Data:  nil,
	})

	return id, nil
}

func (db *DB) UpdateUserProfileName(dbtx DBTX, userAccountID int64, name string) error {
	const sqlQuery = `
		UPDATE user_profiles
		SET name = ?
		WHERE id = (
			SELECT user_profile_id
			FROM user_accounts
			WHERE id = ? AND deleted_at IS NULL
		)`

	result, err := dbtx.Exec(sqlQuery, name, userAccountID)
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
		Table: "user_profiles",
		Type:  DBUpdate,
		Data:  nil,
	})

	return nil
}
