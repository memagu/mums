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

func (db *DB) CreateUserAccount(e execer, userCredentialsID, userProfileID int64) (int64, error) {
	res, err := e.Exec(
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

	e.Emit(DBEvent{
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

	q.Emit(DBEvent{
		Table: "user_accounts",
		Type:  DBRead,
		Data:  userAccountID,
	})

	return userAccountID, nil
}

func (db *DB) ReadActiveUserAccountIDByEmail(q queryer, email string) (int64, error) {
	row := q.QueryRow(`
		SELECT a.id
		FROM user_accounts AS a
		JOIN user_credentials AS c ON a.user_credentials_id = c.id
		WHERE c.email = ? AND a.deleted_at IS NULL`,
		email,
	)

	var userAccountID int64
	if err := row.Scan(&userAccountID); err != nil {
		return 0, err
	}

	q.Emit(DBEvent{
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

	q.Emit(DBEvent{
		Table: "user_profiles",
		Type:  DBRead,
		Data:  nil,
	})

	return upd, nil
}

func (db *DB) ReadUserCredentialsByUserAccountID(q queryer, userAccountID int64) (UserCredentialsData, error) {
	row := q.QueryRow(`
		SELECT c.id, c.email, c.hashword
		FROM user_credentials AS c
		JOIN user_accounts AS a ON c.id = a.user_credentials_id
		WHERE a.id = ? AND a.deleted_at IS NULL`,
		userAccountID,
	)

	var ucd UserCredentialsData
	if err := row.Scan(
		&ucd.ID,
		&ucd.Email,
		&ucd.Hashword,
	); err != nil {
		return UserCredentialsData{}, err
	}

	q.Emit(DBEvent{
		Table: "user_credentials",
		Type:  DBRead,
		Data:  nil,
	})

	return ucd, nil
}

func (db *DB) DeleteUserAccount(e execer, userAccountID int64) error {
	const sqlQuery = `
		UPDATE user_accounts
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := e.Exec(sqlQuery, userAccountID)
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
		Table: "user_accounts",
		Type:  DBDelete,
		Data:  nil,
	})

	return nil
}
