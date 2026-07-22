package db

import (
	"github.com/memagu/mums/internal/roles"
)

const SchemaUserAccountRoleMappings = `
CREATE TABLE IF NOT EXISTS user_account_role_mappings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_account_id INTEGER NOT NULL,
	user_account_role TEXT NOT NULL,
	FOREIGN KEY (user_account_id) REFERENCES user_accounts(id) ON DELETE CASCADE,
	UNIQUE (user_account_id, user_account_role)
);`

func (db *DB) CreateUserAccountRoleMapping(exec execer, userAccountID int64, userAccountRole roles.UserAccountRole) (int64, error) {
	res, err := exec.Exec(
		`INSERT INTO user_account_role_mappings (user_account_id, user_account_role) VALUES (?, ?)`,
		userAccountID,
		string(userAccountRole),
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()

	db.Emit(DBEvent{
		"user_account_role_mappings",
		DBCreate,
		nil,
	})

	return id, err
}

func (db *DB) ReadUserAccountRoles(q queryer, userAccountID int64) ([]roles.UserAccountRole, error) {
	rows, err := q.Query(`
		SELECT user_account_role
		FROM user_account_role_mappings
		WHERE user_account_id = ?`, userAccountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userAccountRoles []roles.UserAccountRole

	for rows.Next() {
		var userAccountRole roles.UserAccountRole
		if err := rows.Scan(&userAccountRole); err != nil {
			return nil, err
		}
		userAccountRoles = append(userAccountRoles, userAccountRole)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	db.Emit(DBEvent{
		"user_account_role_mappings",
		DBRead,
		nil,
	})

	return userAccountRoles, nil
}
