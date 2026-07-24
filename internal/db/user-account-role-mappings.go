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

func (db *DB) CreateUserAccountRoleMapping(e execer, userAccountID int64, userAccountRole roles.UserAccountRole) (int64, error) {
	res, err := e.Exec(
		`INSERT INTO user_account_role_mappings (user_account_id, user_account_role) VALUES (?, ?)`,
		userAccountID,
		string(userAccountRole),
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	e.Emit(DBEvent{
		Table: "user_account_role_mappings",
		Type:  DBCreate,
		Data:  nil,
	})

	return id, nil
}

func (db *DB) ReadUserAccountRoles(q queryer, userAccountID int64) ([]roles.UserAccountRole, error) {
	rows, err := q.Query(`
		SELECT uarm.user_account_role
		FROM user_account_role_mappings AS uarm
		JOIN user_accounts AS ua ON uarm.user_account_id = ua.id AND ua.deleted_at IS NULL
		WHERE uarm.user_account_id = ?`, userAccountID)
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

	q.Emit(DBEvent{
		Table: "user_account_role_mappings",
		Type:  DBRead,
		Data:  nil,
	})

	return userAccountRoles, nil
}
