package db

const SchemaUserAccounts = `
CREATE TABLE IF NOT EXISTS user_accounts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	deleted_at DATETIME DEFAULT NULL,
	user_credentials_id INTEGER NOT NULL UNIQUE,
	user_profile_id INTEGER NOT NULL UNIQUE,
	FOREIGN KEY (user_credentials_id) REFERENCES user_credentials(id) ON DELETE CASCADE,
	FOREIGN KEY (user_profile_id) REFERENCES user_profiles(id) ON DELETE CASCADE
);`

func (db *DB) CreateUserAccount(exec execer, userCredentialsID, userProfileID int64) (int64, error) {
	res, err := exec.Exec(
		`INSERT INTO user_accounts (user_credentials_id, user_profile_id) VALUES (?, ?)`,
		userCredentialsID,
		userProfileID,
	)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	db.Emit(DBEvent{
		Table: "user_accounts",
		Type:  DBCreate,
		Data:  nil,
	})

	return id, nil
}

func (db *DB) ReadUserAccountIDByUserCredentialsID(q queryer, userCredentialsID int64) (int64, error) {
	row := q.QueryRow(
		`SELECT id FROM user_accounts WHERE user_credentials_id = ? AND deleted_at IS NULL`,
		userCredentialsID,
	)

	var userAccountID int64
	if err := row.Scan(&userAccountID); err != nil {
		return 0, err
	}

	db.Emit(DBEvent{
		Table: "user_accounts",
		Type:  DBRead,
		Data:  userAccountID,
	})

	return userAccountID, nil
}

func (db *DB) ReadUserProfileByUserAccountID(q queryer, userAccountID int64) (UserProfileData, error) {
	row := q.QueryRow(`
		SELECT p.name
		FROM user_profiles AS p
		JOIN user_accounts AS a ON p.id = a.user_profile_id
		WHERE a.id = ? AND a.deleted_at IS NULL`,
		userAccountID,
	)

	var upd UserProfileData
	if err := row.Scan(
		&upd.Name,
	); err != nil {
		return UserProfileData{}, err
	}

	db.Emit(DBEvent{
		Table: "user_profiles",
		Type:  DBRead,
		Data:  nil,
	})

	return upd, nil
}
